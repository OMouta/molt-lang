package runtime

import "molt/internal/ast"

func EqualExpr(left, right ast.Expr) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

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
	case *ast.BreakExpr:
		_, ok := right.(*ast.BreakExpr)
		return ok
	case *ast.ContinueExpr:
		_, ok := right.(*ast.ContinueExpr)
		return ok
	case *ast.Identifier:
		r, ok := right.(*ast.Identifier)
		return ok && l.Name == r.Name
	case *ast.ExportExpr:
		r, ok := right.(*ast.ExportExpr)
		return ok && EqualExpr(l.Name, r.Name)
	case *ast.ImportExpr:
		r, ok := right.(*ast.ImportExpr)
		return ok && EqualExpr(l.Path, r.Path)
	case *ast.GroupExpr:
		r, ok := right.(*ast.GroupExpr)
		return ok && EqualExpr(l.Inner, r.Inner)
	case *ast.ListLiteral:
		r, ok := right.(*ast.ListLiteral)
		return ok && equalExprSlices(l.Elements, r.Elements)
	case *ast.ListBindingPattern:
		r, ok := right.(*ast.ListBindingPattern)
		return ok && equalBindingPatterns(l.Elements, r.Elements)
	case *ast.RecordLiteral:
		r, ok := right.(*ast.RecordLiteral)
		return ok && equalRecordFields(l.Fields, r.Fields)
	case *ast.RecordBindingPattern:
		r, ok := right.(*ast.RecordBindingPattern)
		return ok && equalRecordBindingFields(l.Fields, r.Fields)
	case *ast.BlockExpr:
		r, ok := right.(*ast.BlockExpr)
		return ok && equalExprSlices(l.Expressions, r.Expressions)
	case *ast.AssignmentExpr:
		r, ok := right.(*ast.AssignmentExpr)
		return ok && EqualExpr(l.Target, r.Target) && EqualExpr(l.Value, r.Value)
	case *ast.IndexExpr:
		r, ok := right.(*ast.IndexExpr)
		return ok && EqualExpr(l.Target, r.Target) && EqualExpr(l.Index, r.Index)
	case *ast.FieldAccessExpr:
		r, ok := right.(*ast.FieldAccessExpr)
		return ok && EqualExpr(l.Target, r.Target) && EqualExpr(l.Name, r.Name)
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
	case *ast.WhileExpr:
		r, ok := right.(*ast.WhileExpr)
		return ok && EqualExpr(l.Condition, r.Condition) && EqualExpr(l.Body, r.Body)
	case *ast.MatchExpr:
		r, ok := right.(*ast.MatchExpr)
		return ok && EqualExpr(l.Subject, r.Subject) && equalMatchCases(l.Cases, r.Cases)
	case *ast.ForInExpr:
		r, ok := right.(*ast.ForInExpr)
		return ok &&
			EqualExpr(l.Binding, r.Binding) &&
			EqualExpr(l.Iterable, r.Iterable) &&
			EqualExpr(l.Body, r.Body)
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
	case *ast.UnquoteExpr:
		r, ok := right.(*ast.UnquoteExpr)
		return ok && EqualExpr(l.Expression, r.Expression)
	case *ast.SpliceExpr:
		r, ok := right.(*ast.SpliceExpr)
		return ok && EqualExpr(l.Expression, r.Expression)
	case *ast.MutationLiteralExpr:
		r, ok := right.(*ast.MutationLiteralExpr)
		return ok && equalRules(l.Rules, r.Rules)
	case *ast.MutationCaptureExpr:
		r, ok := right.(*ast.MutationCaptureExpr)
		return ok && EqualExpr(l.Name, r.Name)
	case *ast.ApplyMutationExpr:
		r, ok := right.(*ast.ApplyMutationExpr)
		return ok && EqualExpr(l.Target, r.Target) && EqualExpr(l.Mutation, r.Mutation)
	default:
		return false
	}
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

func equalRecordFields(left, right []*ast.RecordField) bool {
	if len(left) != len(right) {
		return false
	}

	for i := range left {
		if !EqualExpr(left[i].Name, right[i].Name) || !EqualExpr(left[i].Value, right[i].Value) {
			return false
		}
	}

	return true
}

func equalBindingPatterns(left, right []ast.BindingPattern) bool {
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

func equalRecordBindingFields(left, right []*ast.RecordBindingField) bool {
	if len(left) != len(right) {
		return false
	}

	for i := range left {
		if !EqualExpr(left[i].Name, right[i].Name) || !EqualExpr(left[i].Value, right[i].Value) {
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

func equalMatchCases(left, right []*ast.MatchCase) bool {
	if len(left) != len(right) {
		return false
	}

	for i := range left {
		if !EqualExpr(left[i].Pattern, right[i].Pattern) || !EqualExpr(left[i].Branch, right[i].Branch) {
			return false
		}
	}

	return true
}
