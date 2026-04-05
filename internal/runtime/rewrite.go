package runtime

import (
	"fmt"

	"molt/internal/ast"
)

func ValidateMutationRules(rules []*ast.MutationRule) error {
	for i, rule := range rules {
		if err := ValidateMutationRule(rule); err != nil {
			return fmt.Errorf("invalid mutation rule %d: %w", i+1, err)
		}
	}

	return nil
}

func ValidateMutationRule(rule *ast.MutationRule) error {
	if rule == nil {
		return fmt.Errorf("mutation rule cannot be nil")
	}

	patternOperator := isOperatorLiteral(rule.Pattern)
	replacementOperator := isOperatorLiteral(rule.Replacement)

	switch {
	case patternOperator && !replacementOperator:
		return fmt.Errorf("operator replacement rules must replace one operator with another")
	case !patternOperator && replacementOperator:
		return fmt.Errorf("operator literals are only valid in operator replacement rules")
	}

	if err := validateMutationExpr(rule.Pattern); err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	if err := validateMutationExpr(rule.Replacement); err != nil {
		return fmt.Errorf("invalid replacement: %w", err)
	}

	return nil
}

func Rewrite(expr ast.Expr, mutation *MutationValue) (ast.Expr, error) {
	if mutation == nil {
		return nil, fmt.Errorf("mutation value cannot be nil")
	}

	if err := ValidateMutationRules(mutation.Rules); err != nil {
		return nil, err
	}

	rewritten := CloneExpr(expr)
	for _, rule := range mutation.Rules {
		var err error
		rewritten, err = ApplyRule(rewritten, rule)
		if err != nil {
			return nil, err
		}
	}

	return rewritten, nil
}

func ApplyRule(expr ast.Expr, rule *ast.MutationRule) (ast.Expr, error) {
	if err := ValidateMutationRule(rule); err != nil {
		return nil, err
	}

	rewritten, _ := rewriteWithRule(expr, rule)
	return rewritten, nil
}

func CloneExpr(expr ast.Expr) ast.Expr {
	switch node := expr.(type) {
	case *ast.OperatorLiteral:
		return &ast.OperatorLiteral{SourceSpan: node.SourceSpan, Symbol: node.Symbol}
	case *ast.NumberLiteral:
		return &ast.NumberLiteral{SourceSpan: node.SourceSpan, Value: node.Value}
	case *ast.StringLiteral:
		return &ast.StringLiteral{SourceSpan: node.SourceSpan, Value: node.Value}
	case *ast.BooleanLiteral:
		return &ast.BooleanLiteral{SourceSpan: node.SourceSpan, Value: node.Value}
	case *ast.NilLiteral:
		return &ast.NilLiteral{SourceSpan: node.SourceSpan}
	case *ast.Identifier:
		return &ast.Identifier{SourceSpan: node.SourceSpan, Name: node.Name}
	case *ast.GroupExpr:
		return &ast.GroupExpr{SourceSpan: node.SourceSpan, Inner: CloneExpr(node.Inner)}
	case *ast.ListLiteral:
		return &ast.ListLiteral{SourceSpan: node.SourceSpan, Elements: cloneExprs(node.Elements)}
	case *ast.BlockExpr:
		return &ast.BlockExpr{SourceSpan: node.SourceSpan, Expressions: cloneExprs(node.Expressions)}
	case *ast.AssignmentExpr:
		return &ast.AssignmentExpr{
			SourceSpan: node.SourceSpan,
			Target:     cloneIdentifier(node.Target),
			Value:      CloneExpr(node.Value),
		}
	case *ast.IndexExpr:
		return &ast.IndexExpr{
			SourceSpan: node.SourceSpan,
			Target:     CloneExpr(node.Target),
			Index:      CloneExpr(node.Index),
		}
	case *ast.UnaryExpr:
		return &ast.UnaryExpr{
			SourceSpan: node.SourceSpan,
			Operator:   node.Operator,
			Operand:    CloneExpr(node.Operand),
		}
	case *ast.BinaryExpr:
		return &ast.BinaryExpr{
			SourceSpan: node.SourceSpan,
			Left:       CloneExpr(node.Left),
			Operator:   node.Operator,
			Right:      CloneExpr(node.Right),
		}
	case *ast.ConditionalExpr:
		return &ast.ConditionalExpr{
			SourceSpan: node.SourceSpan,
			Condition:  CloneExpr(node.Condition),
			ThenBranch: CloneExpr(node.ThenBranch),
			ElseBranch: CloneExpr(node.ElseBranch),
		}
	case *ast.CallExpr:
		return &ast.CallExpr{
			SourceSpan: node.SourceSpan,
			Callee:     CloneExpr(node.Callee),
			Arguments:  cloneExprs(node.Arguments),
		}
	case *ast.NamedFunctionExpr:
		return &ast.NamedFunctionExpr{
			SourceSpan: node.SourceSpan,
			Name:       cloneIdentifier(node.Name),
			Parameters: cloneIdentifiers(node.Parameters),
			Body:       CloneExpr(node.Body),
		}
	case *ast.FunctionLiteralExpr:
		return &ast.FunctionLiteralExpr{
			SourceSpan: node.SourceSpan,
			Parameters: cloneIdentifiers(node.Parameters),
			Body:       CloneExpr(node.Body),
		}
	case *ast.QuoteExpr:
		return &ast.QuoteExpr{SourceSpan: node.SourceSpan, Body: CloneExpr(node.Body)}
	case *ast.MutationLiteralExpr:
		return &ast.MutationLiteralExpr{SourceSpan: node.SourceSpan, Rules: CloneRules(node.Rules)}
	case *ast.ApplyMutationExpr:
		return &ast.ApplyMutationExpr{
			SourceSpan: node.SourceSpan,
			Target:     CloneExpr(node.Target),
			Mutation:   CloneExpr(node.Mutation),
		}
	default:
		panic(fmt.Sprintf("unsupported expression clone type %T", expr))
	}
}

func CloneRules(rules []*ast.MutationRule) []*ast.MutationRule {
	cloned := make([]*ast.MutationRule, 0, len(rules))
	for _, rule := range rules {
		cloned = append(cloned, &ast.MutationRule{
			SourceSpan:  rule.SourceSpan,
			Pattern:     CloneExpr(rule.Pattern),
			Replacement: CloneExpr(rule.Replacement),
		})
	}

	return cloned
}

func EqualExpr(left, right ast.Expr) bool {
	switch l := left.(type) {
	case *ast.OperatorLiteral:
		r, ok := right.(*ast.OperatorLiteral)
		return ok && l.Symbol == r.Symbol
	case *ast.NumberLiteral:
		r, ok := right.(*ast.NumberLiteral)
		return ok && l.Value == r.Value
	case *ast.StringLiteral:
		r, ok := right.(*ast.StringLiteral)
		return ok && l.Value == r.Value
	case *ast.BooleanLiteral:
		r, ok := right.(*ast.BooleanLiteral)
		return ok && l.Value == r.Value
	case *ast.NilLiteral:
		_, ok := right.(*ast.NilLiteral)
		return ok
	case *ast.Identifier:
		r, ok := right.(*ast.Identifier)
		return ok && l.Name == r.Name
	case *ast.GroupExpr:
		r, ok := right.(*ast.GroupExpr)
		return ok && EqualExpr(l.Inner, r.Inner)
	case *ast.ListLiteral:
		r, ok := right.(*ast.ListLiteral)
		return ok && equalExprSlices(l.Elements, r.Elements)
	case *ast.BlockExpr:
		r, ok := right.(*ast.BlockExpr)
		return ok && equalExprSlices(l.Expressions, r.Expressions)
	case *ast.AssignmentExpr:
		r, ok := right.(*ast.AssignmentExpr)
		return ok && EqualExpr(l.Target, r.Target) && EqualExpr(l.Value, r.Value)
	case *ast.IndexExpr:
		r, ok := right.(*ast.IndexExpr)
		return ok && EqualExpr(l.Target, r.Target) && EqualExpr(l.Index, r.Index)
	case *ast.UnaryExpr:
		r, ok := right.(*ast.UnaryExpr)
		return ok && l.Operator == r.Operator && EqualExpr(l.Operand, r.Operand)
	case *ast.BinaryExpr:
		r, ok := right.(*ast.BinaryExpr)
		return ok && l.Operator == r.Operator && EqualExpr(l.Left, r.Left) && EqualExpr(l.Right, r.Right)
	case *ast.ConditionalExpr:
		r, ok := right.(*ast.ConditionalExpr)
		return ok &&
			EqualExpr(l.Condition, r.Condition) &&
			EqualExpr(l.ThenBranch, r.ThenBranch) &&
			EqualExpr(l.ElseBranch, r.ElseBranch)
	case *ast.CallExpr:
		r, ok := right.(*ast.CallExpr)
		return ok && EqualExpr(l.Callee, r.Callee) && equalExprSlices(l.Arguments, r.Arguments)
	case *ast.NamedFunctionExpr:
		r, ok := right.(*ast.NamedFunctionExpr)
		return ok &&
			EqualExpr(l.Name, r.Name) &&
			equalIdentifiers(l.Parameters, r.Parameters) &&
			EqualExpr(l.Body, r.Body)
	case *ast.FunctionLiteralExpr:
		r, ok := right.(*ast.FunctionLiteralExpr)
		return ok && equalIdentifiers(l.Parameters, r.Parameters) && EqualExpr(l.Body, r.Body)
	case *ast.QuoteExpr:
		r, ok := right.(*ast.QuoteExpr)
		return ok && EqualExpr(l.Body, r.Body)
	case *ast.MutationLiteralExpr:
		r, ok := right.(*ast.MutationLiteralExpr)
		return ok && equalRules(l.Rules, r.Rules)
	case *ast.ApplyMutationExpr:
		r, ok := right.(*ast.ApplyMutationExpr)
		return ok && EqualExpr(l.Target, r.Target) && EqualExpr(l.Mutation, r.Mutation)
	default:
		return false
	}
}

func rewriteWithRule(expr ast.Expr, rule *ast.MutationRule) (ast.Expr, bool) {
	if EqualExpr(expr, rule.Pattern) {
		return CloneExpr(rule.Replacement), true
	}

	switch node := expr.(type) {
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
	case *ast.BlockExpr:
		expressions, changed := rewriteExprSlice(node.Expressions, rule)
		if !changed {
			return expr, false
		}

		return &ast.BlockExpr{SourceSpan: node.SourceSpan, Expressions: expressions}, true
	case *ast.AssignmentExpr:
		target, targetChanged := rewriteIdentifier(node.Target, rule)
		value, valueChanged := rewriteWithRule(node.Value, rule)
		if !targetChanged && !valueChanged {
			return expr, false
		}

		return &ast.AssignmentExpr{
			SourceSpan: node.SourceSpan,
			Target:     target,
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
	case *ast.ConditionalExpr:
		condition, conditionChanged := rewriteWithRule(node.Condition, rule)
		thenBranch, thenChanged := rewriteWithRule(node.ThenBranch, rule)
		elseBranch, elseChanged := rewriteWithRule(node.ElseBranch, rule)
		if !conditionChanged && !thenChanged && !elseChanged {
			return expr, false
		}

		return &ast.ConditionalExpr{
			SourceSpan: node.SourceSpan,
			Condition:  condition,
			ThenBranch: thenBranch,
			ElseBranch: elseBranch,
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
	switch node := expr.(type) {
	case *ast.OperatorLiteral,
		*ast.NumberLiteral,
		*ast.StringLiteral,
		*ast.BooleanLiteral,
		*ast.NilLiteral,
		*ast.Identifier:
		return nil
	case *ast.GroupExpr:
		return validateMutationExpr(node.Inner)
	case *ast.ListLiteral:
		for _, element := range node.Elements {
			if err := validateMutationExpr(element); err != nil {
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

func isOperatorLiteral(expr ast.Expr) bool {
	_, ok := expr.(*ast.OperatorLiteral)
	return ok
}

func cloneExprs(items []ast.Expr) []ast.Expr {
	cloned := make([]ast.Expr, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, CloneExpr(item))
	}

	return cloned
}

func cloneIdentifier(identifier *ast.Identifier) *ast.Identifier {
	if identifier == nil {
		return nil
	}

	return &ast.Identifier{SourceSpan: identifier.SourceSpan, Name: identifier.Name}
}

func cloneIdentifiers(items []*ast.Identifier) []*ast.Identifier {
	cloned := make([]*ast.Identifier, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, cloneIdentifier(item))
	}

	return cloned
}

func equalExprSlices(left, right []ast.Expr) bool {
	if len(left) != len(right) {
		return false
	}

	for i := range left {
		if !EqualExpr(left[i], right[i]) {
			return false
		}
	}

	return true
}

func equalIdentifiers(left, right []*ast.Identifier) bool {
	if len(left) != len(right) {
		return false
	}

	for i := range left {
		if !EqualExpr(left[i], right[i]) {
			return false
		}
	}

	return true
}

func equalRules(left, right []*ast.MutationRule) bool {
	if len(left) != len(right) {
		return false
	}

	for i := range left {
		if !EqualExpr(left[i].Pattern, right[i].Pattern) || !EqualExpr(left[i].Replacement, right[i].Replacement) {
			return false
		}
	}

	return true
}
