package evaluator

import (
	"fmt"

	"molt/internal/ast"
	"molt/internal/builtins"
	"molt/internal/diagnostic"
	"molt/internal/runtime"
	"molt/internal/source"
)

func (e *Evaluator) evalNamedFunction(env *runtime.Environment, expr *ast.NamedFunctionExpr) runtime.Value {
	function := e.makeFunctionValue(env, expr.Name.Name, expr.Parameters, expr.Body)
	env.Define(expr.Name.Name, function)
	return function
}

func (e *Evaluator) evalMutationLiteral(expr *ast.MutationLiteralExpr) (runtime.Value, error) {
	rules := runtime.CloneRules(expr.Rules)
	if err := runtime.ValidateMutationRules(rules); err != nil {
		return nil, e.runtimeError(expr, err.Error())
	}

	return &runtime.MutationValue{Rules: rules}, nil
}

func (e *Evaluator) evalApplyMutation(env *runtime.Environment, expr *ast.ApplyMutationExpr) (runtime.Value, error) {
	target, err := e.evalExpr(env, expr.Target)
	if err != nil {
		return nil, err
	}

	mutationValue, err := e.evalExpr(env, expr.Mutation)
	if err != nil {
		return nil, err
	}

	mutation, ok := mutationValue.(*runtime.MutationValue)
	if !ok {
		return nil, e.runtimeError(expr.Mutation, fmt.Sprintf("expected mutation value, got %q", mutationValue.TypeName()))
	}

	rewritten, err := runtime.ApplyMutationValue(target, mutation)
	if err != nil {
		return nil, e.runtimeError(expr.Target, err.Error())
	}

	return rewritten, nil
}

func (e *Evaluator) makeFunctionValue(env *runtime.Environment, name string, parameters []*ast.Identifier, body ast.Expr) *runtime.UserFunctionValue {
	names := make([]string, 0, len(parameters))
	for _, parameter := range parameters {
		names = append(names, parameter.Name)
	}

	return &runtime.UserFunctionValue{
		Name:       name,
		Parameters: names,
		Body:       body,
		Env:        env,
	}
}

func (e *Evaluator) evalCall(env *runtime.Environment, expr *ast.CallExpr) (runtime.Value, error) {
	callee, err := e.evalExpr(env, expr.Callee)
	if err != nil {
		return nil, err
	}

	args := make([]runtime.Value, 0, len(expr.Arguments))
	for _, argumentExpr := range expr.Arguments {
		argument, err := e.evalExpr(env, argumentExpr)
		if err != nil {
			return nil, err
		}

		args = append(args, argument)
	}

	return e.invokeValue(env, callee, args, expr.Span())
}

func (e *Evaluator) evalCodeValue(code *runtime.CodeValue) (runtime.Value, error) {
	if code == nil {
		return nil, fmt.Errorf("nil code value")
	}

	captured := code.Env
	if captured == nil {
		captured = runtime.NewEnvironment(nil)
	}

	builtins.Install(captured)

	frame := runtime.NewEnvironment(captured)
	builtins.Install(frame)

	value, err := e.evalExpr(frame, code.Body)
	if err != nil {
		return nil, e.wrapLoopControlError(err)
	}

	return value, nil
}

func (e *Evaluator) invokeValue(env *runtime.Environment, callee runtime.Value, args []runtime.Value, span source.Span) (runtime.Value, error) {
	switch fn := callee.(type) {
	case *runtime.UserFunctionValue:
		if len(args) != len(fn.Parameters) {
			return nil, diagnostic.NewRuntimeError(arityMessage(len(fn.Parameters), len(args)), span)
		}

		callEnv := runtime.NewEnvironment(fn.Env)
		for i, parameter := range fn.Parameters {
			callEnv.Define(parameter, args[i])
		}

		value, err := e.evalExpr(callEnv, fn.Body)
		if err != nil {
			return nil, e.wrapLoopControlError(err)
		}

		return value, nil
	case runtime.NativeFunction:
		if native, ok := callee.(*runtime.NativeFunctionValue); ok && native.Arity >= 0 && len(args) != native.Arity {
			return nil, diagnostic.NewRuntimeError(arityMessage(native.Arity, len(args)), span)
		}

		return fn.Call(&runtime.CallContext{
			FunctionName: fn.Name(),
			Environment:  env,
			Arguments:    e.arguments(),
			CallSpan:     span,
			EvalCode:     e.evalCodeValue,
			Invoke: func(callee runtime.Value, args []runtime.Value, env *runtime.Environment, span source.Span) (runtime.Value, error) {
				return e.invokeValue(env, callee, args, span)
			},
			ReadFile:  e.readFileFunc(),
			WriteFile: e.writeFileFunc(),
			Input:     e.inputReader(),
			Output:    e.outputWriter(),
		}, args)
	default:
		return nil, diagnostic.NewRuntimeError(
			fmt.Sprintf("cannot call value of type %q", callee.TypeName()),
			span,
		)
	}
}

func arityMessage(expected, got int) string {
	return fmt.Sprintf("expected %d arguments but got %d", expected, got)
}
