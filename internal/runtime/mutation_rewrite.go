package runtime

import (
	"fmt"

	"molt/internal/ast"
)

func rewriteWithRule(expr ast.Expr, rule *ast.MutationRule) (ast.Expr, bool) {
	if EqualExpr(expr, rule.Pattern) {
		return CloneExpr(rule.Replacement), true
	}

	switch node := expr.(type) {
	case *ast.ExportExpr:
		name, changed := rewriteIdentifier(node.Name, rule)
		if !changed {
			return expr, false
		}

		return &ast.ExportExpr{
			SourceSpan: node.SourceSpan,
			Name:       name,
		}, true
	case *ast.ImportExpr:
		path, changed := rewriteWithRule(node.Path, rule)
		if !changed {
			return expr, false
		}

		return &ast.ImportExpr{
			SourceSpan: node.SourceSpan,
			Path:       path.(*ast.StringLiteral),
		}, true
	case *ast.UnaryExpr:
		changed := false
		operator := node.Operator
		if replacement, ok := rewriteUnaryOperator(node.Operator, rule); ok {
			operator = replacement
			changed = true
		}

		operand, operandChanged := rewriteWithRule(node.Operand, rule)
		changed = changed || operandChanged
		if !changed {
			return expr, false
		}

		return &ast.UnaryExpr{
			SourceSpan: node.SourceSpan,
			Operator:   operator,
			Operand:    operand,
		}, true
	case *ast.BinaryExpr:
		changed := false
		operator := node.Operator
		if replacement, ok := rewriteBinaryOperator(node.Operator, rule); ok {
			operator = replacement
			changed = true
		}

		left, leftChanged := rewriteWithRule(node.Left, rule)
		right, rightChanged := rewriteWithRule(node.Right, rule)
		changed = changed || leftChanged || rightChanged
		if !changed {
			return expr, false
		}

		return &ast.BinaryExpr{
			SourceSpan: node.SourceSpan,
			Left:       left,
			Operator:   operator,
			Right:      right,
		}, true
	case *ast.GroupExpr:
		inner, changed := rewriteWithRule(node.Inner, rule)
		if !changed {
			return expr, false
		}

		return &ast.GroupExpr{SourceSpan: node.SourceSpan, Inner: inner}, true
	case *ast.ListLiteral:
		elements, changed := rewriteExprSlice(node.Elements, rule)
		if !changed {
			return expr, false
		}

		return &ast.ListLiteral{SourceSpan: node.SourceSpan, Elements: elements}, true
	case *ast.ListBindingPattern:
		elements, changed := rewriteBindingPatternSlice(node.Elements, rule)
		if !changed {
			return expr, false
		}

		return &ast.ListBindingPattern{SourceSpan: node.SourceSpan, Elements: elements}, true
	case *ast.RecordLiteral:
		fields, changed := rewriteRecordFieldSlice(node.Fields, rule)
		if !changed {
			return expr, false
		}

		return &ast.RecordLiteral{SourceSpan: node.SourceSpan, Fields: fields}, true
	case *ast.RecordBindingPattern:
		fields, changed := rewriteRecordBindingFieldSlice(node.Fields, rule)
		if !changed {
			return expr, false
		}

		return &ast.RecordBindingPattern{SourceSpan: node.SourceSpan, Fields: fields}, true
	case *ast.BlockExpr:
		expressions, changed := rewriteExprSlice(node.Expressions, rule)
		if !changed {
			return expr, false
		}

		return &ast.BlockExpr{SourceSpan: node.SourceSpan, Expressions: expressions}, true
	case *ast.AssignmentExpr:
		target, targetChanged := rewriteWithRule(node.Target, rule)
		value, valueChanged := rewriteWithRule(node.Value, rule)
		if !targetChanged && !valueChanged {
			return expr, false
		}

		return &ast.AssignmentExpr{
			SourceSpan: node.SourceSpan,
			Target:     target.(ast.AssignmentTarget),
			Value:      value,
		}, true
	case *ast.IndexExpr:
		target, targetChanged := rewriteWithRule(node.Target, rule)
		index, indexChanged := rewriteWithRule(node.Index, rule)
		if !targetChanged && !indexChanged {
			return expr, false
		}

		return &ast.IndexExpr{
			SourceSpan: node.SourceSpan,
			Target:     target,
			Index:      index,
		}, true
	case *ast.FieldAccessExpr:
		target, targetChanged := rewriteWithRule(node.Target, rule)
		name, nameChanged := rewriteIdentifier(node.Name, rule)
		if !targetChanged && !nameChanged {
			return expr, false
		}

		return &ast.FieldAccessExpr{
			SourceSpan: node.SourceSpan,
			Target:     target,
			Name:       name,
		}, true
	case *ast.ConditionalExpr:
		condition, conditionChanged := rewriteWithRule(node.Condition, rule)
		thenBranch, thenChanged := rewriteWithRule(node.ThenBranch, rule)
		var elseBranch ast.Expr
		elseChanged := false
		if node.ElseBranch != nil {
			elseBranch, elseChanged = rewriteWithRule(node.ElseBranch, rule)
		}
		if !conditionChanged && !thenChanged && !elseChanged {
			return expr, false
		}

		return &ast.ConditionalExpr{
			SourceSpan: node.SourceSpan,
			Condition:  condition,
			ThenBranch: thenBranch,
			ElseBranch: elseBranch,
		}, true
	case *ast.WhileExpr:
		condition, conditionChanged := rewriteWithRule(node.Condition, rule)
		body, bodyChanged := rewriteWithRule(node.Body, rule)
		if !conditionChanged && !bodyChanged {
			return expr, false
		}

		return &ast.WhileExpr{
			SourceSpan: node.SourceSpan,
			Condition:  condition,
			Body:       body,
		}, true
	case *ast.MatchExpr:
		subject, subjectChanged := rewriteWithRule(node.Subject, rule)
		cases, casesChanged := rewriteMatchCases(node.Cases, rule)
		if !subjectChanged && !casesChanged {
			return expr, false
		}

		return &ast.MatchExpr{
			SourceSpan: node.SourceSpan,
			Subject:    subject,
			Cases:      cases,
		}, true
	case *ast.ForInExpr:
		binding, bindingChanged := rewriteWithRule(node.Binding, rule)
		iterable, iterableChanged := rewriteWithRule(node.Iterable, rule)
		body, bodyChanged := rewriteWithRule(node.Body, rule)
		if !bindingChanged && !iterableChanged && !bodyChanged {
			return expr, false
		}

		return &ast.ForInExpr{
			SourceSpan: node.SourceSpan,
			Binding:    binding.(ast.BindingPattern),
			Iterable:   iterable,
			Body:       body,
		}, true
	case *ast.CallExpr:
		callee, calleeChanged := rewriteWithRule(node.Callee, rule)
		args, argsChanged := rewriteExprSlice(node.Arguments, rule)
		if !calleeChanged && !argsChanged {
			return expr, false
		}

		return &ast.CallExpr{
			SourceSpan: node.SourceSpan,
			Callee:     callee,
			Arguments:  args,
		}, true
	case *ast.NamedFunctionExpr:
		name, nameChanged := rewriteIdentifier(node.Name, rule)
		params, paramsChanged := rewriteIdentifierSlice(node.Parameters, rule)
		body, bodyChanged := rewriteWithRule(node.Body, rule)
		if !nameChanged && !paramsChanged && !bodyChanged {
			return expr, false
		}

		return &ast.NamedFunctionExpr{
			SourceSpan: node.SourceSpan,
			Name:       name,
			Parameters: params,
			Body:       body,
		}, true
	case *ast.FunctionLiteralExpr:
		params, paramsChanged := rewriteIdentifierSlice(node.Parameters, rule)
		body, bodyChanged := rewriteWithRule(node.Body, rule)
		if !paramsChanged && !bodyChanged {
			return expr, false
		}

		return &ast.FunctionLiteralExpr{
			SourceSpan: node.SourceSpan,
			Parameters: params,
			Body:       body,
		}, true
	case *ast.QuoteExpr:
		body, changed := rewriteWithRule(node.Body, rule)
		if !changed {
			return expr, false
		}

		return &ast.QuoteExpr{SourceSpan: node.SourceSpan, Body: body}, true
	case *ast.UnquoteExpr:
		inner, changed := rewriteWithRule(node.Expression, rule)
		if !changed {
			return expr, false
		}

		return &ast.UnquoteExpr{SourceSpan: node.SourceSpan, Expression: inner}, true
	case *ast.SpliceExpr:
		inner, changed := rewriteWithRule(node.Expression, rule)
		if !changed {
			return expr, false
		}

		return &ast.SpliceExpr{SourceSpan: node.SourceSpan, Expression: inner}, true
	case *ast.MutationLiteralExpr:
		rules, changed := rewriteRuleSlice(node.Rules, rule)
		if !changed {
			return expr, false
		}

		return &ast.MutationLiteralExpr{SourceSpan: node.SourceSpan, Rules: rules}, true
	case *ast.ApplyMutationExpr:
		target, targetChanged := rewriteWithRule(node.Target, rule)
		mutation, mutationChanged := rewriteWithRule(node.Mutation, rule)
		if !targetChanged && !mutationChanged {
			return expr, false
		}

		return &ast.ApplyMutationExpr{
			SourceSpan: node.SourceSpan,
			Target:     target,
			Mutation:   mutation,
		}, true
	default:
		return expr, false
	}
}

func validateMutationExpr(expr ast.Expr) error {
	if expr == nil {
		return nil
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
		return nil
	case *ast.ExportExpr:
		return validateMutationExpr(node.Name)
	case *ast.ImportExpr:
		return validateMutationExpr(node.Path)
	case *ast.GroupExpr:
		return validateMutationExpr(node.Inner)
	case *ast.ListLiteral:
		for _, element := range node.Elements {
			if err := validateMutationExpr(element); err != nil {
				return err
			}
		}
		return nil
	case *ast.ListBindingPattern:
		for _, element := range node.Elements {
			if err := validateMutationExpr(element); err != nil {
				return err
			}
		}
		return nil
	case *ast.RecordLiteral:
		for _, field := range node.Fields {
			if err := validateMutationExpr(field.Name); err != nil {
				return err
			}
			if err := validateMutationExpr(field.Value); err != nil {
				return err
			}
		}
		return nil
	case *ast.RecordBindingPattern:
		for _, field := range node.Fields {
			if err := validateMutationExpr(field.Name); err != nil {
				return err
			}
			if err := validateMutationExpr(field.Value); err != nil {
				return err
			}
		}
		return nil
	case *ast.BlockExpr:
		for _, inner := range node.Expressions {
			if err := validateMutationExpr(inner); err != nil {
				return err
			}
		}
		return nil
	case *ast.AssignmentExpr:
		if err := validateMutationExpr(node.Target); err != nil {
			return err
		}
		return validateMutationExpr(node.Value)
	case *ast.IndexExpr:
		if err := validateMutationExpr(node.Target); err != nil {
			return err
		}
		return validateMutationExpr(node.Index)
	case *ast.FieldAccessExpr:
		if err := validateMutationExpr(node.Target); err != nil {
			return err
		}
		return validateMutationExpr(node.Name)
	case *ast.UnaryExpr:
		return validateMutationExpr(node.Operand)
	case *ast.BinaryExpr:
		if err := validateMutationExpr(node.Left); err != nil {
			return err
		}
		return validateMutationExpr(node.Right)
	case *ast.ConditionalExpr:
		if err := validateMutationExpr(node.Condition); err != nil {
			return err
		}
		if err := validateMutationExpr(node.ThenBranch); err != nil {
			return err
		}
		return validateMutationExpr(node.ElseBranch)
	case *ast.WhileExpr:
		if err := validateMutationExpr(node.Condition); err != nil {
			return err
		}
		return validateMutationExpr(node.Body)
	case *ast.MatchExpr:
		if err := validateMutationExpr(node.Subject); err != nil {
			return err
		}
		for _, matchCase := range node.Cases {
			if err := validateMutationExpr(matchCase.Pattern); err != nil {
				return err
			}
			if err := validateMutationExpr(matchCase.Branch); err != nil {
				return err
			}
		}
		return nil
	case *ast.ForInExpr:
		if err := validateMutationExpr(node.Binding); err != nil {
			return err
		}
		if err := validateMutationExpr(node.Iterable); err != nil {
			return err
		}
		return validateMutationExpr(node.Body)
	case *ast.CallExpr:
		if err := validateMutationExpr(node.Callee); err != nil {
			return err
		}
		for _, argument := range node.Arguments {
			if err := validateMutationExpr(argument); err != nil {
				return err
			}
		}
		return nil
	case *ast.NamedFunctionExpr:
		if err := validateMutationExpr(node.Name); err != nil {
			return err
		}
		for _, parameter := range node.Parameters {
			if err := validateMutationExpr(parameter); err != nil {
				return err
			}
		}
		return validateMutationExpr(node.Body)
	case *ast.FunctionLiteralExpr:
		for _, parameter := range node.Parameters {
			if err := validateMutationExpr(parameter); err != nil {
				return err
			}
		}
		return validateMutationExpr(node.Body)
	case *ast.QuoteExpr:
		return validateMutationExpr(node.Body)
	case *ast.UnquoteExpr:
		return validateMutationExpr(node.Expression)
	case *ast.SpliceExpr:
		return validateMutationExpr(node.Expression)
	case *ast.MutationLiteralExpr:
		return fmt.Errorf("nested mutation literals are not supported in mutation rules")
	case *ast.ApplyMutationExpr:
		return fmt.Errorf("mutation applications are not supported in mutation rules")
	default:
		return fmt.Errorf("unsupported mutation expression type %T", expr)
	}
}

func rewriteUnaryOperator(operator ast.UnaryOperator, rule *ast.MutationRule) (ast.UnaryOperator, bool) {
	pattern, ok := rule.Pattern.(*ast.OperatorLiteral)
	if !ok || string(operator) != pattern.Symbol {
		return "", false
	}

	replacement := rule.Replacement.(*ast.OperatorLiteral)
	switch replacement.Symbol {
	case string(ast.UnaryNegate):
		return ast.UnaryNegate, true
	case string(ast.UnaryNot):
		return ast.UnaryNot, true
	default:
		return "", false
	}
}

func rewriteBinaryOperator(operator ast.BinaryOperator, rule *ast.MutationRule) (ast.BinaryOperator, bool) {
	pattern, ok := rule.Pattern.(*ast.OperatorLiteral)
	if !ok || string(operator) != pattern.Symbol {
		return "", false
	}

	replacement := rule.Replacement.(*ast.OperatorLiteral)
	switch replacement.Symbol {
	case string(ast.BinaryAdd):
		return ast.BinaryAdd, true
	case string(ast.BinarySubtract):
		return ast.BinarySubtract, true
	case string(ast.BinaryMultiply):
		return ast.BinaryMultiply, true
	case string(ast.BinaryDivide):
		return ast.BinaryDivide, true
	case string(ast.BinaryModulo):
		return ast.BinaryModulo, true
	case string(ast.BinaryEqual):
		return ast.BinaryEqual, true
	case string(ast.BinaryNotEqual):
		return ast.BinaryNotEqual, true
	case string(ast.BinaryLess):
		return ast.BinaryLess, true
	case string(ast.BinaryLessEqual):
		return ast.BinaryLessEqual, true
	case string(ast.BinaryGreater):
		return ast.BinaryGreater, true
	case string(ast.BinaryGreaterEqual):
		return ast.BinaryGreaterEqual, true
	case string(ast.BinaryAnd):
		return ast.BinaryAnd, true
	case string(ast.BinaryOr):
		return ast.BinaryOr, true
	default:
		return "", false
	}
}

func rewriteExprSlice(items []ast.Expr, rule *ast.MutationRule) ([]ast.Expr, bool) {
	changed := false
	rewritten := make([]ast.Expr, 0, len(items))
	for _, item := range items {
		next, itemChanged := rewriteWithRule(item, rule)
		rewritten = append(rewritten, next)
		changed = changed || itemChanged
	}

	return rewritten, changed
}

func rewriteIdentifier(identifier *ast.Identifier, rule *ast.MutationRule) (*ast.Identifier, bool) {
	rewritten, changed := rewriteWithRule(identifier, rule)
	if !changed {
		return identifier, false
	}

	next, ok := rewritten.(*ast.Identifier)
	if !ok {
		return identifier, false
	}

	return next, true
}

func rewriteIdentifierSlice(items []*ast.Identifier, rule *ast.MutationRule) ([]*ast.Identifier, bool) {
	changed := false
	rewritten := make([]*ast.Identifier, 0, len(items))
	for _, item := range items {
		next, itemChanged := rewriteIdentifier(item, rule)
		rewritten = append(rewritten, next)
		changed = changed || itemChanged
	}

	return rewritten, changed
}

func rewriteRecordFieldSlice(items []*ast.RecordField, rule *ast.MutationRule) ([]*ast.RecordField, bool) {
	changed := false
	rewritten := make([]*ast.RecordField, 0, len(items))
	for _, item := range items {
		name, nameChanged := rewriteIdentifier(item.Name, rule)
		value, valueChanged := rewriteWithRule(item.Value, rule)
		rewritten = append(rewritten, &ast.RecordField{
			SourceSpan: item.SourceSpan,
			Name:       name,
			Value:      value,
		})
		changed = changed || nameChanged || valueChanged
	}

	return rewritten, changed
}

func rewriteBindingPatternSlice(items []ast.BindingPattern, rule *ast.MutationRule) ([]ast.BindingPattern, bool) {
	changed := false
	rewritten := make([]ast.BindingPattern, 0, len(items))
	for _, item := range items {
		next, itemChanged := rewriteWithRule(item, rule)
		rewritten = append(rewritten, next.(ast.BindingPattern))
		changed = changed || itemChanged
	}

	return rewritten, changed
}

func rewriteRecordBindingFieldSlice(items []*ast.RecordBindingField, rule *ast.MutationRule) ([]*ast.RecordBindingField, bool) {
	changed := false
	rewritten := make([]*ast.RecordBindingField, 0, len(items))
	for _, item := range items {
		name, nameChanged := rewriteIdentifier(item.Name, rule)
		value, valueChanged := rewriteWithRule(item.Value, rule)
		rewritten = append(rewritten, &ast.RecordBindingField{
			SourceSpan: item.SourceSpan,
			Name:       name,
			Value:      value.(ast.BindingPattern),
		})
		changed = changed || nameChanged || valueChanged
	}

	return rewritten, changed
}

func rewriteRuleSlice(items []*ast.MutationRule, rule *ast.MutationRule) ([]*ast.MutationRule, bool) {
	changed := false
	rewritten := make([]*ast.MutationRule, 0, len(items))
	for _, item := range items {
		pattern, patternChanged := rewriteWithRule(item.Pattern, rule)
		replacement, replacementChanged := rewriteWithRule(item.Replacement, rule)
		rewritten = append(rewritten, &ast.MutationRule{
			SourceSpan:  item.SourceSpan,
			Pattern:     pattern,
			Replacement: replacement,
		})
		changed = changed || patternChanged || replacementChanged
	}

	return rewritten, changed
}

func rewriteMatchCases(items []*ast.MatchCase, rule *ast.MutationRule) ([]*ast.MatchCase, bool) {
	changed := false
	rewritten := make([]*ast.MatchCase, 0, len(items))
	for _, item := range items {
		pattern, patternChanged := rewriteWithRule(item.Pattern, rule)
		branch, branchChanged := rewriteWithRule(item.Branch, rule)
		rewritten = append(rewritten, &ast.MatchCase{
			SourceSpan: item.SourceSpan,
			Pattern:    pattern,
			Branch:     branch,
		})
		changed = changed || patternChanged || branchChanged
	}

	return rewritten, changed
}

func isOperatorLiteral(expr ast.Expr) bool {
	_, ok := expr.(*ast.OperatorLiteral)
	return ok
}
