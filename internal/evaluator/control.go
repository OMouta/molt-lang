package evaluator

import (
	"errors"
	"fmt"

	"molt/internal/ast"
	"molt/internal/builtins"
	"molt/internal/diagnostic"
	"molt/internal/runtime"
)

func (e *Evaluator) evalConditional(env *runtime.Environment, expr *ast.ConditionalExpr) (runtime.Value, error) {
	condition, err := e.evalExpr(env, expr.Condition)
	if err != nil {
		return nil, err
	}

	boolean, ok := condition.(*runtime.BooleanValue)
	if !ok {
		return nil, e.runtimeError(expr.Condition, fmt.Sprintf("if condition must be boolean, got %q", condition.TypeName()))
	}

	if boolean.Value {
		return e.evalExpr(env, expr.ThenBranch)
	}

	if expr.ElseBranch == nil {
		return runtime.Nil, nil
	}

	return e.evalExpr(env, expr.ElseBranch)
}

func (e *Evaluator) evalWhile(env *runtime.Environment, expr *ast.WhileExpr) (runtime.Value, error) {
	for {
		condition, err := e.evalExpr(env, expr.Condition)
		if err != nil {
			return nil, err
		}

		boolean, ok := condition.(*runtime.BooleanValue)
		if !ok {
			return nil, e.runtimeError(expr.Condition, fmt.Sprintf("while condition must be boolean, got %q", condition.TypeName()))
		}

		if !boolean.Value {
			return runtime.Nil, nil
		}

		iterationEnv := runtime.NewEnvironment(env)
		if _, err := e.evalExpr(iterationEnv, expr.Body); err != nil {
			signal, ok := asLoopControlSignal(err)
			if !ok {
				return nil, err
			}

			switch signal.kind {
			case loopControlBreak:
				return runtime.Nil, nil
			case loopControlContinue:
				continue
			default:
				return nil, err
			}
		}
	}
}

func (e *Evaluator) evalTryCatch(env *runtime.Environment, expr *ast.TryCatchExpr) (runtime.Value, error) {
	value, err := e.evalExpr(env, expr.Body)
	if err == nil {
		return value, nil
	}

	if _, ok := asLoopControlSignal(err); ok {
		return nil, err
	}

	caught, ok := caughtErrorValue(err)
	if !ok {
		return nil, err
	}

	catchEnv := runtime.NewEnvironment(env)
	catchEnv.Define(expr.CatchBinding.Name, caught)
	return e.evalExpr(catchEnv, expr.CatchBranch)
}

func (e *Evaluator) evalMatch(env *runtime.Environment, expr *ast.MatchExpr) (runtime.Value, error) {
	subject, err := e.evalExpr(env, expr.Subject)
	if err != nil {
		return nil, err
	}

	for _, matchCase := range expr.Cases {
		branchEnv, matched := e.matchCaseEnvironment(env, subject, matchCase)
		if !matched {
			continue
		}

		return e.evalExpr(branchEnv, matchCase.Branch)
	}

	return runtime.Nil, nil
}

func (e *Evaluator) matchCaseEnvironment(env *runtime.Environment, subject runtime.Value, matchCase *ast.MatchCase) (*runtime.Environment, bool) {
	branchEnv := runtime.NewEnvironment(env)

	switch pattern := matchCase.Pattern.(type) {
	case *ast.NumberLiteral:
		return branchEnv, valuesEqual(subject, &runtime.NumberValue{Value: pattern.Value})
	case *ast.StringLiteral:
		return branchEnv, valuesEqual(subject, &runtime.StringValue{Value: pattern.Value})
	case *ast.BooleanLiteral:
		return branchEnv, valuesEqual(subject, &runtime.BooleanValue{Value: pattern.Value})
	case *ast.NilLiteral:
		return branchEnv, valuesEqual(subject, runtime.Nil)
	case *ast.Identifier:
		if pattern.Name == "_" {
			return branchEnv, true
		}

		branchEnv.Define(pattern.Name, subject)
		return branchEnv, true
	default:
		return env, false
	}
}

func (e *Evaluator) evalForIn(env *runtime.Environment, expr *ast.ForInExpr) (runtime.Value, error) {
	iterable, err := e.evalExpr(env, expr.Iterable)
	if err != nil {
		return nil, err
	}

	switch value := iterable.(type) {
	case *runtime.ListValue:
		for _, element := range value.Elements {
			iterationEnv := runtime.NewEnvironment(env)
			if err := e.defineBindingPattern(iterationEnv, expr.Binding, element); err != nil {
				return nil, err
			}
			if _, err := e.evalExpr(iterationEnv, expr.Body); err != nil {
				signal, ok := asLoopControlSignal(err)
				if !ok {
					return nil, err
				}

				switch signal.kind {
				case loopControlBreak:
					return runtime.Nil, nil
				case loopControlContinue:
					continue
				default:
					return nil, err
				}
			}
		}
	case *runtime.StringValue:
		for _, r := range []rune(value.Value) {
			iterationEnv := runtime.NewEnvironment(env)
			if err := e.defineBindingPattern(iterationEnv, expr.Binding, &runtime.StringValue{Value: string(r)}); err != nil {
				return nil, err
			}
			if _, err := e.evalExpr(iterationEnv, expr.Body); err != nil {
				signal, ok := asLoopControlSignal(err)
				if !ok {
					return nil, err
				}

				switch signal.kind {
				case loopControlBreak:
					return runtime.Nil, nil
				case loopControlContinue:
					continue
				default:
					return nil, err
				}
			}
		}
	default:
		return nil, e.runtimeError(expr.Iterable, fmt.Sprintf("for loop expects list or string, got %q", iterable.TypeName()))
	}

	return runtime.Nil, nil
}

func (e *Evaluator) runtimeError(node ast.Expr, message string) error {
	return diagnostic.NewRuntimeError(message, node.Span())
}

func (e *Evaluator) wrapLoopControlError(err error) error {
	signal, ok := asLoopControlSignal(err)
	if !ok {
		return err
	}

	return diagnostic.NewRuntimeError(fmt.Sprintf("%s is only allowed inside loops", signal.kind), signal.span)
}

func (e *Evaluator) wrapRaisedError(err error) error {
	raised, ok := builtins.AsThrown(err)
	if !ok {
		return err
	}

	notes := []diagnostic.Note(nil)
	if raised.Value != nil && raised.Value.HasData {
		notes = append(notes, diagnostic.Note{
			Message: "error data: " + runtime.ShowValue(raised.Value.Data),
		})
	}

	message := "thrown error"
	if raised.Value != nil {
		message = raised.Value.Message
	}

	return diagnostic.NewRuntimeError(message, raised.Span, notes...)
}

func (e *Evaluator) wrapControlFlowError(err error) error {
	err = e.wrapLoopControlError(err)
	return e.wrapRaisedError(err)
}

func asLoopControlSignal(err error) (loopControlSignal, bool) {
	signal, ok := err.(loopControlSignal)
	return signal, ok
}

func caughtErrorValue(err error) (*runtime.ErrorValue, bool) {
	if raised, ok := builtins.AsThrown(err); ok {
		if raised.Value != nil {
			return raised.Value, true
		}

		return runtime.NewErrorValue("thrown error", nil, false), true
	}

	var runtimeErr diagnostic.RuntimeError
	if errors.As(err, &runtimeErr) {
		return runtime.NewErrorValue(runtimeErr.Diagnostic().Message, nil, false), true
	}

	return nil, false
}
