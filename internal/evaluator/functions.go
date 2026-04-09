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

func (e *Evaluator) evalQuote(env *runtime.Environment, expr *ast.QuoteExpr) (runtime.Value, error) {
	if err := e.validateQuoteTemplate(expr); err != nil {
		return nil, err
	}

	body, err := e.interpolateQuoteBody(env, expr)
	if err != nil {
		return nil, err
	}

	return &runtime.CodeValue{
		Body:     body,
		Template: runtime.CloneExpr(expr.Body),
		Env:      env,
	}, nil
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

type quoteSpliceContext string

const (
	quoteSpliceContextList  quoteSpliceContext = "list"
	quoteSpliceContextCall  quoteSpliceContext = "call"
	quoteSpliceContextBlock quoteSpliceContext = "block"
)

type quoteValidationContext string

const (
	quoteValidationExpr    quoteValidationContext = "expression"
	quoteValidationList    quoteValidationContext = "list"
	quoteValidationCall    quoteValidationContext = "call"
	quoteValidationBlock   quoteValidationContext = "block"
	quoteValidationAssign  quoteValidationContext = "assignment-target"
	quoteValidationBinding quoteValidationContext = "binding"
)

func (e *Evaluator) validateQuoteTemplate(quote *ast.QuoteExpr) error {
	if quote == nil {
		return nil
	}

	return e.validateQuoteExpr(quote.Body, quoteValidationBlock)
}

func (e *Evaluator) validateQuoteExpr(expr ast.Expr, context quoteValidationContext) error {
	if expr == nil {
		return nil
	}

	switch node := expr.(type) {
	case *ast.UnquoteExpr:
		switch context {
		case quoteValidationAssign:
			return e.runtimeError(node, "unquote is not allowed in assignment targets inside quotes")
		case quoteValidationBinding:
			return e.runtimeError(node, "unquote is not allowed in binding positions inside quotes")
		default:
			return nil
		}
	case *ast.SpliceExpr:
		switch context {
		case quoteValidationList, quoteValidationCall, quoteValidationBlock:
			return nil
		case quoteValidationAssign:
			return e.runtimeError(node, "splice is not allowed in assignment targets inside quotes")
		case quoteValidationBinding:
			return e.runtimeError(node, "splice is not allowed in binding positions inside quotes")
		default:
			return e.runtimeError(node, "splice is only allowed in list, call, or block positions inside quotes")
		}
	case *ast.OperatorLiteral,
		*ast.NumberLiteral,
		*ast.StringLiteral,
		*ast.BooleanLiteral,
		*ast.NilLiteral,
		*ast.BreakExpr,
		*ast.ContinueExpr,
		*ast.Identifier:
		return nil
	case *ast.ExportExpr:
		return e.validateQuoteExpr(node.Name, quoteValidationExpr)
	case *ast.ImportExpr:
		return e.validateQuoteExpr(node.Path, quoteValidationExpr)
	case *ast.GroupExpr:
		return e.validateQuoteExpr(node.Inner, quoteValidationExpr)
	case *ast.ListLiteral:
		for _, item := range node.Elements {
			if err := e.validateQuoteExpr(item, quoteValidationList); err != nil {
				return err
			}
		}
		return nil
	case *ast.ListBindingPattern:
		for _, item := range node.Elements {
			if err := e.validateQuoteExpr(item, quoteValidationBinding); err != nil {
				return err
			}
		}
		return nil
	case *ast.RecordLiteral:
		for _, field := range node.Fields {
			if err := e.validateQuoteExpr(field.Value, quoteValidationExpr); err != nil {
				return err
			}
		}
		return nil
	case *ast.RecordBindingPattern:
		for _, field := range node.Fields {
			if err := e.validateQuoteExpr(field.Value, quoteValidationBinding); err != nil {
				return err
			}
		}
		return nil
	case *ast.BlockExpr:
		for _, item := range node.Expressions {
			if err := e.validateQuoteExpr(item, quoteValidationBlock); err != nil {
				return err
			}
		}
		return nil
	case *ast.AssignmentExpr:
		if err := e.validateQuoteExpr(node.Target, quoteValidationAssign); err != nil {
			return err
		}
		return e.validateQuoteExpr(node.Value, quoteValidationExpr)
	case *ast.IndexExpr:
		if err := e.validateQuoteExpr(node.Target, quoteValidationExpr); err != nil {
			return err
		}
		return e.validateQuoteExpr(node.Index, quoteValidationExpr)
	case *ast.FieldAccessExpr:
		return e.validateQuoteExpr(node.Target, quoteValidationExpr)
	case *ast.UnaryExpr:
		return e.validateQuoteExpr(node.Operand, quoteValidationExpr)
	case *ast.BinaryExpr:
		if err := e.validateQuoteExpr(node.Left, quoteValidationExpr); err != nil {
			return err
		}
		return e.validateQuoteExpr(node.Right, quoteValidationExpr)
	case *ast.ConditionalExpr:
		if err := e.validateQuoteExpr(node.Condition, quoteValidationExpr); err != nil {
			return err
		}
		if err := e.validateQuoteExpr(node.ThenBranch, quoteValidationExpr); err != nil {
			return err
		}
		return e.validateQuoteExpr(node.ElseBranch, quoteValidationExpr)
	case *ast.WhileExpr:
		if err := e.validateQuoteExpr(node.Condition, quoteValidationExpr); err != nil {
			return err
		}
		return e.validateQuoteExpr(node.Body, quoteValidationExpr)
	case *ast.TryCatchExpr:
		if err := e.validateQuoteExpr(node.Body, quoteValidationExpr); err != nil {
			return err
		}
		return e.validateQuoteExpr(node.CatchBranch, quoteValidationExpr)
	case *ast.MatchExpr:
		if err := e.validateQuoteExpr(node.Subject, quoteValidationExpr); err != nil {
			return err
		}
		for _, matchCase := range node.Cases {
			if err := e.validateQuoteExpr(matchCase.Pattern, quoteValidationExpr); err != nil {
				return err
			}
			if err := e.validateQuoteExpr(matchCase.Branch, quoteValidationExpr); err != nil {
				return err
			}
		}
		return nil
	case *ast.ForInExpr:
		if err := e.validateQuoteExpr(node.Binding, quoteValidationBinding); err != nil {
			return err
		}
		if err := e.validateQuoteExpr(node.Iterable, quoteValidationExpr); err != nil {
			return err
		}
		return e.validateQuoteExpr(node.Body, quoteValidationExpr)
	case *ast.CallExpr:
		if err := e.validateQuoteExpr(node.Callee, quoteValidationExpr); err != nil {
			return err
		}
		for _, arg := range node.Arguments {
			if err := e.validateQuoteExpr(arg, quoteValidationCall); err != nil {
				return err
			}
		}
		return nil
	case *ast.NamedFunctionExpr:
		return e.validateQuoteExpr(node.Body, quoteValidationExpr)
	case *ast.FunctionLiteralExpr:
		return e.validateQuoteExpr(node.Body, quoteValidationExpr)
	case *ast.QuoteExpr:
		return e.validateQuoteTemplate(node)
	case *ast.MutationLiteralExpr:
		for _, rule := range node.Rules {
			if err := e.validateQuoteExpr(rule.Pattern, quoteValidationExpr); err != nil {
				return err
			}
			if err := e.validateQuoteExpr(rule.Replacement, quoteValidationExpr); err != nil {
				return err
			}
		}
		return nil
	case *ast.ApplyMutationExpr:
		if err := e.validateQuoteExpr(node.Target, quoteValidationExpr); err != nil {
			return err
		}
		return e.validateQuoteExpr(node.Mutation, quoteValidationExpr)
	default:
		return fmt.Errorf("unsupported quote validation expression type %T", expr)
	}
}

func (e *Evaluator) interpolateQuoteBody(env *runtime.Environment, quote *ast.QuoteExpr) (ast.Expr, error) {
	if quote == nil {
		return nil, nil
	}

	return e.interpolateQuoteSequence(env, quote.Body, quote.Span(), quoteSpliceContextBlock)
}

func (e *Evaluator) interpolateQuoteSequence(env *runtime.Environment, expr ast.Expr, span source.Span, context quoteSpliceContext) (ast.Expr, error) {
	switch node := expr.(type) {
	case *ast.BlockExpr:
		items, err := e.interpolateQuoteExprSlice(env, node.Expressions, context)
		if err != nil {
			return nil, err
		}

		return quoteSequenceToExpr(node.SourceSpan, items), nil
	case *ast.SpliceExpr:
		items, err := e.interpolateQuoteSplice(env, node, context)
		if err != nil {
			return nil, err
		}

		return quoteSequenceToExpr(span, items), nil
	default:
		return e.interpolateQuoteExpr(env, expr)
	}
}

func (e *Evaluator) interpolateQuoteExpr(env *runtime.Environment, expr ast.Expr) (ast.Expr, error) {
	if expr == nil {
		return nil, nil
	}

	switch node := expr.(type) {
	case *ast.OperatorLiteral,
		*ast.NumberLiteral,
		*ast.StringLiteral,
		*ast.BooleanLiteral,
		*ast.NilLiteral,
		*ast.BreakExpr,
		*ast.ContinueExpr,
		*ast.Identifier:
		return runtime.CloneExpr(node), nil
	case *ast.ExportExpr:
		return &ast.ExportExpr{
			SourceSpan: node.SourceSpan,
			Name:       runtime.CloneExpr(node.Name).(*ast.Identifier),
		}, nil
	case *ast.ImportExpr:
		return &ast.ImportExpr{
			SourceSpan: node.SourceSpan,
			Path:       runtime.CloneExpr(node.Path).(*ast.StringLiteral),
		}, nil
	case *ast.GroupExpr:
		inner, err := e.interpolateQuoteExpr(env, node.Inner)
		if err != nil {
			return nil, err
		}

		return &ast.GroupExpr{SourceSpan: node.SourceSpan, Inner: inner}, nil
	case *ast.ListLiteral:
		elements, err := e.interpolateQuoteExprSlice(env, node.Elements, quoteSpliceContextList)
		if err != nil {
			return nil, err
		}

		return &ast.ListLiteral{SourceSpan: node.SourceSpan, Elements: elements}, nil
	case *ast.ListBindingPattern:
		elements, err := e.interpolateQuoteBindingPatterns(env, node.Elements)
		if err != nil {
			return nil, err
		}

		return &ast.ListBindingPattern{SourceSpan: node.SourceSpan, Elements: elements}, nil
	case *ast.RecordLiteral:
		fields := make([]*ast.RecordField, 0, len(node.Fields))
		for _, field := range node.Fields {
			value, err := e.interpolateQuoteExpr(env, field.Value)
			if err != nil {
				return nil, err
			}

			fields = append(fields, &ast.RecordField{
				SourceSpan: field.SourceSpan,
				Name:       runtime.CloneExpr(field.Name).(*ast.Identifier),
				Value:      value,
			})
		}

		return &ast.RecordLiteral{SourceSpan: node.SourceSpan, Fields: fields}, nil
	case *ast.RecordBindingPattern:
		fields := make([]*ast.RecordBindingField, 0, len(node.Fields))
		for _, field := range node.Fields {
			value, err := e.interpolateQuoteExpr(env, field.Value)
			if err != nil {
				return nil, err
			}

			pattern, ok := value.(ast.BindingPattern)
			if !ok {
				return nil, e.runtimeError(field.Value, "quote interpolation cannot be used in this binding position")
			}

			fields = append(fields, &ast.RecordBindingField{
				SourceSpan: field.SourceSpan,
				Name:       runtime.CloneExpr(field.Name).(*ast.Identifier),
				Value:      pattern,
			})
		}

		return &ast.RecordBindingPattern{SourceSpan: node.SourceSpan, Fields: fields}, nil
	case *ast.BlockExpr:
		expressions, err := e.interpolateQuoteExprSlice(env, node.Expressions, quoteSpliceContextBlock)
		if err != nil {
			return nil, err
		}

		return &ast.BlockExpr{SourceSpan: node.SourceSpan, Expressions: expressions}, nil
	case *ast.AssignmentExpr:
		target, err := e.interpolateQuoteExpr(env, node.Target)
		if err != nil {
			return nil, err
		}

		assignmentTarget, ok := target.(ast.AssignmentTarget)
		if !ok {
			return nil, e.runtimeError(node.Target, "quote interpolation cannot be used in this assignment target")
		}

		value, err := e.interpolateQuoteExpr(env, node.Value)
		if err != nil {
			return nil, err
		}

		return &ast.AssignmentExpr{
			SourceSpan: node.SourceSpan,
			Target:     assignmentTarget,
			Value:      value,
		}, nil
	case *ast.IndexExpr:
		target, err := e.interpolateQuoteExpr(env, node.Target)
		if err != nil {
			return nil, err
		}

		index, err := e.interpolateQuoteExpr(env, node.Index)
		if err != nil {
			return nil, err
		}

		return &ast.IndexExpr{
			SourceSpan: node.SourceSpan,
			Target:     target,
			Index:      index,
		}, nil
	case *ast.FieldAccessExpr:
		target, err := e.interpolateQuoteExpr(env, node.Target)
		if err != nil {
			return nil, err
		}

		return &ast.FieldAccessExpr{
			SourceSpan: node.SourceSpan,
			Target:     target,
			Name:       runtime.CloneExpr(node.Name).(*ast.Identifier),
		}, nil
	case *ast.UnaryExpr:
		operand, err := e.interpolateQuoteExpr(env, node.Operand)
		if err != nil {
			return nil, err
		}

		return &ast.UnaryExpr{
			SourceSpan: node.SourceSpan,
			Operator:   node.Operator,
			Operand:    operand,
		}, nil
	case *ast.BinaryExpr:
		left, err := e.interpolateQuoteExpr(env, node.Left)
		if err != nil {
			return nil, err
		}

		right, err := e.interpolateQuoteExpr(env, node.Right)
		if err != nil {
			return nil, err
		}

		return &ast.BinaryExpr{
			SourceSpan: node.SourceSpan,
			Left:       left,
			Operator:   node.Operator,
			Right:      right,
		}, nil
	case *ast.ConditionalExpr:
		condition, err := e.interpolateQuoteExpr(env, node.Condition)
		if err != nil {
			return nil, err
		}

		thenBranch, err := e.interpolateQuoteExpr(env, node.ThenBranch)
		if err != nil {
			return nil, err
		}

		elseBranch, err := e.interpolateQuoteExpr(env, node.ElseBranch)
		if err != nil {
			return nil, err
		}

		return &ast.ConditionalExpr{
			SourceSpan: node.SourceSpan,
			Condition:  condition,
			ThenBranch: thenBranch,
			ElseBranch: elseBranch,
		}, nil
	case *ast.WhileExpr:
		condition, err := e.interpolateQuoteExpr(env, node.Condition)
		if err != nil {
			return nil, err
		}

		body, err := e.interpolateQuoteExpr(env, node.Body)
		if err != nil {
			return nil, err
		}

		return &ast.WhileExpr{
			SourceSpan: node.SourceSpan,
			Condition:  condition,
			Body:       body,
		}, nil
	case *ast.TryCatchExpr:
		body, err := e.interpolateQuoteExpr(env, node.Body)
		if err != nil {
			return nil, err
		}

		catchBranch, err := e.interpolateQuoteExpr(env, node.CatchBranch)
		if err != nil {
			return nil, err
		}

		return &ast.TryCatchExpr{
			SourceSpan:   node.SourceSpan,
			Body:         body,
			CatchBinding: runtime.CloneExpr(node.CatchBinding).(*ast.Identifier),
			CatchBranch:  catchBranch,
		}, nil
	case *ast.MatchExpr:
		subject, err := e.interpolateQuoteExpr(env, node.Subject)
		if err != nil {
			return nil, err
		}

		cases := make([]*ast.MatchCase, 0, len(node.Cases))
		for _, matchCase := range node.Cases {
			pattern, err := e.interpolateQuoteExpr(env, matchCase.Pattern)
			if err != nil {
				return nil, err
			}

			branch, err := e.interpolateQuoteExpr(env, matchCase.Branch)
			if err != nil {
				return nil, err
			}

			cases = append(cases, &ast.MatchCase{
				SourceSpan: matchCase.SourceSpan,
				Pattern:    pattern,
				Branch:     branch,
			})
		}

		return &ast.MatchExpr{
			SourceSpan: node.SourceSpan,
			Subject:    subject,
			Cases:      cases,
		}, nil
	case *ast.ForInExpr:
		binding, err := e.interpolateQuoteExpr(env, node.Binding)
		if err != nil {
			return nil, err
		}

		bindingPattern, ok := binding.(ast.BindingPattern)
		if !ok {
			return nil, e.runtimeError(node.Binding, "quote interpolation cannot be used in this loop binding")
		}

		iterable, err := e.interpolateQuoteExpr(env, node.Iterable)
		if err != nil {
			return nil, err
		}

		body, err := e.interpolateQuoteExpr(env, node.Body)
		if err != nil {
			return nil, err
		}

		return &ast.ForInExpr{
			SourceSpan: node.SourceSpan,
			Binding:    bindingPattern,
			Iterable:   iterable,
			Body:       body,
		}, nil
	case *ast.CallExpr:
		callee, err := e.interpolateQuoteExpr(env, node.Callee)
		if err != nil {
			return nil, err
		}

		args, err := e.interpolateQuoteExprSlice(env, node.Arguments, quoteSpliceContextCall)
		if err != nil {
			return nil, err
		}

		return &ast.CallExpr{
			SourceSpan: node.SourceSpan,
			Callee:     callee,
			Arguments:  args,
		}, nil
	case *ast.NamedFunctionExpr:
		body, err := e.interpolateQuoteExpr(env, node.Body)
		if err != nil {
			return nil, err
		}

		return &ast.NamedFunctionExpr{
			SourceSpan: node.SourceSpan,
			Name:       runtime.CloneExpr(node.Name).(*ast.Identifier),
			Parameters: cloneQuoteIdentifiers(node.Parameters),
			Body:       body,
		}, nil
	case *ast.FunctionLiteralExpr:
		body, err := e.interpolateQuoteExpr(env, node.Body)
		if err != nil {
			return nil, err
		}

		return &ast.FunctionLiteralExpr{
			SourceSpan: node.SourceSpan,
			Parameters: cloneQuoteIdentifiers(node.Parameters),
			Body:       body,
		}, nil
	case *ast.QuoteExpr:
		return runtime.CloneExpr(node), nil
	case *ast.UnquoteExpr:
		value, err := e.evalExpr(env, node.Expression)
		if err != nil {
			return nil, err
		}

		code, ok := value.(*runtime.CodeValue)
		if !ok {
			return nil, e.runtimeError(node.Expression, fmt.Sprintf("unquote expects code value, got %q", value.TypeName()))
		}

		return runtime.CloneExpr(code.Body), nil
	case *ast.SpliceExpr:
		return nil, e.runtimeError(node, "splice is only valid in list, call, or block positions inside quotes")
	case *ast.MutationLiteralExpr:
		rules := make([]*ast.MutationRule, 0, len(node.Rules))
		for _, rule := range node.Rules {
			pattern, err := e.interpolateQuoteExpr(env, rule.Pattern)
			if err != nil {
				return nil, err
			}

			replacement, err := e.interpolateQuoteExpr(env, rule.Replacement)
			if err != nil {
				return nil, err
			}

			rules = append(rules, &ast.MutationRule{
				SourceSpan:  rule.SourceSpan,
				Pattern:     pattern,
				Replacement: replacement,
			})
		}

		return &ast.MutationLiteralExpr{
			SourceSpan: node.SourceSpan,
			Rules:      rules,
		}, nil
	case *ast.ApplyMutationExpr:
		target, err := e.interpolateQuoteExpr(env, node.Target)
		if err != nil {
			return nil, err
		}

		mutation, err := e.interpolateQuoteExpr(env, node.Mutation)
		if err != nil {
			return nil, err
		}

		return &ast.ApplyMutationExpr{
			SourceSpan: node.SourceSpan,
			Target:     target,
			Mutation:   mutation,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported quoted expression type %T", expr)
	}
}

func (e *Evaluator) interpolateQuoteExprSlice(env *runtime.Environment, items []ast.Expr, context quoteSpliceContext) ([]ast.Expr, error) {
	expressions := make([]ast.Expr, 0, len(items))
	for _, item := range items {
		if splice, ok := item.(*ast.SpliceExpr); ok {
			parts, err := e.interpolateQuoteSplice(env, splice, context)
			if err != nil {
				return nil, err
			}

			expressions = append(expressions, parts...)
			continue
		}

		expr, err := e.interpolateQuoteExpr(env, item)
		if err != nil {
			return nil, err
		}

		expressions = append(expressions, expr)
	}

	return expressions, nil
}

func (e *Evaluator) interpolateQuoteBindingPatterns(env *runtime.Environment, items []ast.BindingPattern) ([]ast.BindingPattern, error) {
	patterns := make([]ast.BindingPattern, 0, len(items))
	for _, item := range items {
		expr, err := e.interpolateQuoteExpr(env, item)
		if err != nil {
			return nil, err
		}

		pattern, ok := expr.(ast.BindingPattern)
		if !ok {
			return nil, e.runtimeError(item, "quote interpolation cannot be used in this binding position")
		}

		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

func (e *Evaluator) interpolateQuoteSplice(env *runtime.Environment, splice *ast.SpliceExpr, context quoteSpliceContext) ([]ast.Expr, error) {
	value, err := e.evalExpr(env, splice.Expression)
	if err != nil {
		return nil, err
	}

	code, ok := value.(*runtime.CodeValue)
	if !ok {
		return nil, e.runtimeError(splice.Expression, fmt.Sprintf("splice expects code value, got %q", value.TypeName()))
	}

	switch context {
	case quoteSpliceContextList, quoteSpliceContextCall:
		list, ok := code.Body.(*ast.ListLiteral)
		if !ok {
			return nil, e.runtimeError(splice.Expression, fmt.Sprintf("splice in %s position expects quoted list", context))
		}

		return cloneQuoteExprs(list.Elements), nil
	case quoteSpliceContextBlock:
		block, ok := code.Body.(*ast.BlockExpr)
		if !ok {
			return nil, e.runtimeError(splice.Expression, "splice in block position expects quoted block")
		}

		return cloneQuoteExprs(block.Expressions), nil
	default:
		return nil, fmt.Errorf("unsupported quote splice context %q", context)
	}
}

func cloneQuoteIdentifiers(items []*ast.Identifier) []*ast.Identifier {
	cloned := make([]*ast.Identifier, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, runtime.CloneExpr(item).(*ast.Identifier))
	}

	return cloned
}

func cloneQuoteExprs(items []ast.Expr) []ast.Expr {
	cloned := make([]ast.Expr, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, runtime.CloneExpr(item))
	}

	return cloned
}

func quoteSequenceToExpr(span source.Span, expressions []ast.Expr) ast.Expr {
	switch len(expressions) {
	case 0:
		return &ast.BlockExpr{
			SourceSpan:  span,
			Expressions: nil,
		}
	case 1:
		return expressions[0]
	default:
		return &ast.BlockExpr{
			SourceSpan:  span,
			Expressions: expressions,
		}
	}
}
