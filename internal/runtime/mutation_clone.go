package runtime

import (
	"fmt"

	"molt/internal/ast"
)

func CloneExpr(expr ast.Expr) ast.Expr {
	if expr == nil {
		return nil
	}

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
	case *ast.BreakExpr:
		return &ast.BreakExpr{SourceSpan: node.SourceSpan}
	case *ast.ContinueExpr:
		return &ast.ContinueExpr{SourceSpan: node.SourceSpan}
	case *ast.Identifier:
		return &ast.Identifier{SourceSpan: node.SourceSpan, Name: node.Name}
	case *ast.ExportExpr:
		return &ast.ExportExpr{
			SourceSpan: node.SourceSpan,
			Name:       cloneIdentifier(node.Name),
		}
	case *ast.ImportExpr:
		return &ast.ImportExpr{
			SourceSpan: node.SourceSpan,
			Path:       CloneExpr(node.Path).(*ast.StringLiteral),
		}
	case *ast.GroupExpr:
		return &ast.GroupExpr{SourceSpan: node.SourceSpan, Inner: CloneExpr(node.Inner)}
	case *ast.ListLiteral:
		return &ast.ListLiteral{SourceSpan: node.SourceSpan, Elements: cloneExprs(node.Elements)}
	case *ast.ListBindingPattern:
		return &ast.ListBindingPattern{SourceSpan: node.SourceSpan, Elements: cloneBindingPatterns(node.Elements)}
	case *ast.RecordLiteral:
		return &ast.RecordLiteral{SourceSpan: node.SourceSpan, Fields: cloneRecordFields(node.Fields)}
	case *ast.RecordBindingPattern:
		return &ast.RecordBindingPattern{SourceSpan: node.SourceSpan, Fields: cloneRecordBindingFields(node.Fields)}
	case *ast.BlockExpr:
		return &ast.BlockExpr{SourceSpan: node.SourceSpan, Expressions: cloneExprs(node.Expressions)}
	case *ast.AssignmentExpr:
		return &ast.AssignmentExpr{
			SourceSpan: node.SourceSpan,
			Target:     CloneExpr(node.Target).(ast.AssignmentTarget),
			Value:      CloneExpr(node.Value),
		}
	case *ast.IndexExpr:
		return &ast.IndexExpr{
			SourceSpan: node.SourceSpan,
			Target:     CloneExpr(node.Target),
			Index:      CloneExpr(node.Index),
		}
	case *ast.FieldAccessExpr:
		return &ast.FieldAccessExpr{
			SourceSpan: node.SourceSpan,
			Target:     CloneExpr(node.Target),
			Name:       cloneIdentifier(node.Name),
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
		var elseBranch ast.Expr
		if node.ElseBranch != nil {
			elseBranch = CloneExpr(node.ElseBranch)
		}

		return &ast.ConditionalExpr{
			SourceSpan: node.SourceSpan,
			Condition:  CloneExpr(node.Condition),
			ThenBranch: CloneExpr(node.ThenBranch),
			ElseBranch: elseBranch,
		}
	case *ast.WhileExpr:
		return &ast.WhileExpr{
			SourceSpan: node.SourceSpan,
			Condition:  CloneExpr(node.Condition),
			Body:       CloneExpr(node.Body),
		}
	case *ast.MatchExpr:
		return &ast.MatchExpr{
			SourceSpan: node.SourceSpan,
			Subject:    CloneExpr(node.Subject),
			Cases:      cloneMatchCases(node.Cases),
		}
	case *ast.ForInExpr:
		return &ast.ForInExpr{
			SourceSpan: node.SourceSpan,
			Binding:    CloneExpr(node.Binding).(ast.BindingPattern),
			Iterable:   CloneExpr(node.Iterable),
			Body:       CloneExpr(node.Body),
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

func cloneMatchCases(items []*ast.MatchCase) []*ast.MatchCase {
	cloned := make([]*ast.MatchCase, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, &ast.MatchCase{
			SourceSpan: item.SourceSpan,
			Pattern:    CloneExpr(item.Pattern),
			Branch:     CloneExpr(item.Branch),
		})
	}

	return cloned
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

func cloneRecordFields(items []*ast.RecordField) []*ast.RecordField {
	cloned := make([]*ast.RecordField, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, &ast.RecordField{
			SourceSpan: item.SourceSpan,
			Name:       cloneIdentifier(item.Name),
			Value:      CloneExpr(item.Value),
		})
	}

	return cloned
}

func cloneBindingPatterns(items []ast.BindingPattern) []ast.BindingPattern {
	cloned := make([]ast.BindingPattern, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, CloneExpr(item).(ast.BindingPattern))
	}

	return cloned
}

func cloneRecordBindingFields(items []*ast.RecordBindingField) []*ast.RecordBindingField {
	cloned := make([]*ast.RecordBindingField, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, &ast.RecordBindingField{
			SourceSpan: item.SourceSpan,
			Name:       cloneIdentifier(item.Name),
			Value:      CloneExpr(item.Value).(ast.BindingPattern),
		})
	}

	return cloned
}
