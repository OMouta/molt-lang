package runtime

import (
	"fmt"

	"molt/internal/ast"
)

type mutationCaptures map[string]ast.Expr

func collectMutationCaptures(expr ast.Expr, captures map[string]struct{}) error {
	if expr == nil {
		return nil
	}

	switch node := expr.(type) {
	case *ast.MutationCaptureExpr:
		if node.Name == nil {
			return fmt.Errorf("capture name cannot be nil")
		}
		if node.Name.Name == "_" {
			return fmt.Errorf("capture name cannot be '_'")
		}
		captures[node.Name.Name] = struct{}{}
		return nil
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
		return collectMutationCaptures(node.Name, captures)
	case *ast.ImportExpr:
		return collectMutationCaptures(node.Path, captures)
	case *ast.GroupExpr:
		return collectMutationCaptures(node.Inner, captures)
	case *ast.ListLiteral:
		for _, element := range node.Elements {
			if err := collectMutationCaptures(element, captures); err != nil {
				return err
			}
		}
		return nil
	case *ast.ListBindingPattern:
		for _, element := range node.Elements {
			if err := collectMutationCaptures(element, captures); err != nil {
				return err
			}
		}
		return nil
	case *ast.RecordLiteral:
		for _, field := range node.Fields {
			if err := collectMutationCaptures(field.Name, captures); err != nil {
				return err
			}
			if err := collectMutationCaptures(field.Value, captures); err != nil {
				return err
			}
		}
		return nil
	case *ast.RecordBindingPattern:
		for _, field := range node.Fields {
			if err := collectMutationCaptures(field.Name, captures); err != nil {
				return err
			}
			if err := collectMutationCaptures(field.Value, captures); err != nil {
				return err
			}
		}
		return nil
	case *ast.BlockExpr:
		for _, inner := range node.Expressions {
			if err := collectMutationCaptures(inner, captures); err != nil {
				return err
			}
		}
		return nil
	case *ast.AssignmentExpr:
		if err := collectMutationCaptures(node.Target, captures); err != nil {
			return err
		}
		return collectMutationCaptures(node.Value, captures)
	case *ast.IndexExpr:
		if err := collectMutationCaptures(node.Target, captures); err != nil {
			return err
		}
		return collectMutationCaptures(node.Index, captures)
	case *ast.FieldAccessExpr:
		if err := collectMutationCaptures(node.Target, captures); err != nil {
			return err
		}
		return collectMutationCaptures(node.Name, captures)
	case *ast.UnaryExpr:
		return collectMutationCaptures(node.Operand, captures)
	case *ast.BinaryExpr:
		if err := collectMutationCaptures(node.Left, captures); err != nil {
			return err
		}
		return collectMutationCaptures(node.Right, captures)
	case *ast.ConditionalExpr:
		if err := collectMutationCaptures(node.Condition, captures); err != nil {
			return err
		}
		if err := collectMutationCaptures(node.ThenBranch, captures); err != nil {
			return err
		}
		return collectMutationCaptures(node.ElseBranch, captures)
	case *ast.WhileExpr:
		if err := collectMutationCaptures(node.Condition, captures); err != nil {
			return err
		}
		return collectMutationCaptures(node.Body, captures)
	case *ast.MatchExpr:
		if err := collectMutationCaptures(node.Subject, captures); err != nil {
			return err
		}
		for _, matchCase := range node.Cases {
			if err := collectMutationCaptures(matchCase.Pattern, captures); err != nil {
				return err
			}
			if err := collectMutationCaptures(matchCase.Branch, captures); err != nil {
				return err
			}
		}
		return nil
	case *ast.ForInExpr:
		if err := collectMutationCaptures(node.Binding, captures); err != nil {
			return err
		}
		if err := collectMutationCaptures(node.Iterable, captures); err != nil {
			return err
		}
		return collectMutationCaptures(node.Body, captures)
	case *ast.CallExpr:
		if err := collectMutationCaptures(node.Callee, captures); err != nil {
			return err
		}
		for _, argument := range node.Arguments {
			if err := collectMutationCaptures(argument, captures); err != nil {
				return err
			}
		}
		return nil
	case *ast.NamedFunctionExpr:
		if err := collectMutationCaptures(node.Name, captures); err != nil {
			return err
		}
		for _, parameter := range node.Parameters {
			if err := collectMutationCaptures(parameter, captures); err != nil {
				return err
			}
		}
		return collectMutationCaptures(node.Body, captures)
	case *ast.FunctionLiteralExpr:
		for _, parameter := range node.Parameters {
			if err := collectMutationCaptures(parameter, captures); err != nil {
				return err
			}
		}
		return collectMutationCaptures(node.Body, captures)
	case *ast.QuoteExpr:
		return collectMutationCaptures(node.Body, captures)
	case *ast.UnquoteExpr:
		return collectMutationCaptures(node.Expression, captures)
	case *ast.SpliceExpr:
		return collectMutationCaptures(node.Expression, captures)
	case *ast.MutationLiteralExpr:
		return fmt.Errorf("nested mutation literals are not supported in mutation rules")
	case *ast.ApplyMutationExpr:
		return fmt.Errorf("mutation applications are not supported in mutation rules")
	default:
		return fmt.Errorf("unsupported mutation expression type %T", expr)
	}
}

func matchMutationExpr(pattern, target ast.Expr, captures mutationCaptures) bool {
	if pattern == nil || target == nil {
		return pattern == nil && target == nil
	}

	switch p := pattern.(type) {
	case *ast.MutationCaptureExpr:
		if p.Name == nil {
			return false
		}
		if existing, seen := captures[p.Name.Name]; seen {
			return EqualExpr(existing, target)
		}
		captures[p.Name.Name] = target
		return true
	case *ast.OperatorLiteral:
		t, ok := target.(*ast.OperatorLiteral)
		return ok && p.Symbol == t.Symbol
	case *ast.NumberLiteral:
		t, ok := target.(*ast.NumberLiteral)
		return ok && p.Value == t.Value
	case *ast.StringLiteral:
		t, ok := target.(*ast.StringLiteral)
		return ok && p.Value == t.Value
	case *ast.BooleanLiteral:
		t, ok := target.(*ast.BooleanLiteral)
		return ok && p.Value == t.Value
	case *ast.NilLiteral:
		_, ok := target.(*ast.NilLiteral)
		return ok
	case *ast.BreakExpr:
		_, ok := target.(*ast.BreakExpr)
		return ok
	case *ast.ContinueExpr:
		_, ok := target.(*ast.ContinueExpr)
		return ok
	case *ast.Identifier:
		t, ok := target.(*ast.Identifier)
		return ok && p.Name == t.Name
	case *ast.ExportExpr:
		t, ok := target.(*ast.ExportExpr)
		return ok && matchMutationExpr(p.Name, t.Name, captures)
	case *ast.ImportExpr:
		t, ok := target.(*ast.ImportExpr)
		return ok && matchMutationExpr(p.Path, t.Path, captures)
	case *ast.GroupExpr:
		t, ok := target.(*ast.GroupExpr)
		return ok && matchMutationExpr(p.Inner, t.Inner, captures)
	case *ast.ListLiteral:
		t, ok := target.(*ast.ListLiteral)
		return ok && matchMutationExprSlice(p.Elements, t.Elements, captures)
	case *ast.ListBindingPattern:
		t, ok := target.(*ast.ListBindingPattern)
		return ok && matchMutationBindingPatterns(p.Elements, t.Elements, captures)
	case *ast.RecordLiteral:
		t, ok := target.(*ast.RecordLiteral)
		return ok && matchMutationRecordFields(p.Fields, t.Fields, captures)
	case *ast.RecordBindingPattern:
		t, ok := target.(*ast.RecordBindingPattern)
		return ok && matchMutationRecordBindingFields(p.Fields, t.Fields, captures)
	case *ast.BlockExpr:
		t, ok := target.(*ast.BlockExpr)
		return ok && matchMutationExprSlice(p.Expressions, t.Expressions, captures)
	case *ast.AssignmentExpr:
		t, ok := target.(*ast.AssignmentExpr)
		return ok &&
			matchMutationExpr(p.Target, t.Target, captures) &&
			matchMutationExpr(p.Value, t.Value, captures)
	case *ast.IndexExpr:
		t, ok := target.(*ast.IndexExpr)
		return ok &&
			matchMutationExpr(p.Target, t.Target, captures) &&
			matchMutationExpr(p.Index, t.Index, captures)
	case *ast.FieldAccessExpr:
		t, ok := target.(*ast.FieldAccessExpr)
		return ok &&
			matchMutationExpr(p.Target, t.Target, captures) &&
			matchMutationExpr(p.Name, t.Name, captures)
	case *ast.UnaryExpr:
		t, ok := target.(*ast.UnaryExpr)
		return ok && p.Operator == t.Operator && matchMutationExpr(p.Operand, t.Operand, captures)
	case *ast.BinaryExpr:
		t, ok := target.(*ast.BinaryExpr)
		return ok &&
			p.Operator == t.Operator &&
			matchMutationExpr(p.Left, t.Left, captures) &&
			matchMutationExpr(p.Right, t.Right, captures)
	case *ast.ConditionalExpr:
		t, ok := target.(*ast.ConditionalExpr)
		return ok &&
			matchMutationExpr(p.Condition, t.Condition, captures) &&
			matchMutationExpr(p.ThenBranch, t.ThenBranch, captures) &&
			matchMutationExpr(p.ElseBranch, t.ElseBranch, captures)
	case *ast.WhileExpr:
		t, ok := target.(*ast.WhileExpr)
		return ok &&
			matchMutationExpr(p.Condition, t.Condition, captures) &&
			matchMutationExpr(p.Body, t.Body, captures)
	case *ast.MatchExpr:
		t, ok := target.(*ast.MatchExpr)
		return ok &&
			matchMutationExpr(p.Subject, t.Subject, captures) &&
			matchMutationCases(p.Cases, t.Cases, captures)
	case *ast.ForInExpr:
		t, ok := target.(*ast.ForInExpr)
		return ok &&
			matchMutationExpr(p.Binding, t.Binding, captures) &&
			matchMutationExpr(p.Iterable, t.Iterable, captures) &&
			matchMutationExpr(p.Body, t.Body, captures)
	case *ast.CallExpr:
		t, ok := target.(*ast.CallExpr)
		return ok &&
			matchMutationExpr(p.Callee, t.Callee, captures) &&
			matchMutationExprSlice(p.Arguments, t.Arguments, captures)
	case *ast.NamedFunctionExpr:
		t, ok := target.(*ast.NamedFunctionExpr)
		return ok &&
			matchMutationExpr(p.Name, t.Name, captures) &&
			matchMutationIdentifiers(p.Parameters, t.Parameters, captures) &&
			matchMutationExpr(p.Body, t.Body, captures)
	case *ast.FunctionLiteralExpr:
		t, ok := target.(*ast.FunctionLiteralExpr)
		return ok &&
			matchMutationIdentifiers(p.Parameters, t.Parameters, captures) &&
			matchMutationExpr(p.Body, t.Body, captures)
	case *ast.QuoteExpr:
		t, ok := target.(*ast.QuoteExpr)
		return ok && matchMutationExpr(p.Body, t.Body, captures)
	case *ast.UnquoteExpr:
		t, ok := target.(*ast.UnquoteExpr)
		return ok && matchMutationExpr(p.Expression, t.Expression, captures)
	case *ast.SpliceExpr:
		t, ok := target.(*ast.SpliceExpr)
		return ok && matchMutationExpr(p.Expression, t.Expression, captures)
	case *ast.MutationLiteralExpr:
		t, ok := target.(*ast.MutationLiteralExpr)
		return ok && matchMutationRules(p.Rules, t.Rules, captures)
	case *ast.ApplyMutationExpr:
		t, ok := target.(*ast.ApplyMutationExpr)
		return ok &&
			matchMutationExpr(p.Target, t.Target, captures) &&
			matchMutationExpr(p.Mutation, t.Mutation, captures)
	default:
		return false
	}
}

func instantiateMutationExpr(expr ast.Expr, captures mutationCaptures) ast.Expr {
	if expr == nil {
		return nil
	}

	switch node := expr.(type) {
	case *ast.MutationCaptureExpr:
		if node.Name == nil {
			panic("missing mutation capture name during replacement instantiation")
		}
		captured, exists := captures[node.Name.Name]
		if !exists {
			panic(fmt.Sprintf("missing mutation capture %q during replacement instantiation", node.Name.Name))
		}
		return CloneExpr(captured)
	case *ast.OperatorLiteral,
		*ast.NumberLiteral,
		*ast.StringLiteral,
		*ast.BooleanLiteral,
		*ast.NilLiteral,
		*ast.BreakExpr,
		*ast.ContinueExpr,
		*ast.Identifier:
		return CloneExpr(node)
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
		return &ast.GroupExpr{SourceSpan: node.SourceSpan, Inner: instantiateMutationExpr(node.Inner, captures)}
	case *ast.ListLiteral:
		elements := make([]ast.Expr, 0, len(node.Elements))
		for _, element := range node.Elements {
			elements = append(elements, instantiateMutationExpr(element, captures))
		}
		return &ast.ListLiteral{SourceSpan: node.SourceSpan, Elements: elements}
	case *ast.ListBindingPattern:
		elements := make([]ast.BindingPattern, 0, len(node.Elements))
		for _, element := range node.Elements {
			elements = append(elements, instantiateMutationExpr(element, captures).(ast.BindingPattern))
		}
		return &ast.ListBindingPattern{SourceSpan: node.SourceSpan, Elements: elements}
	case *ast.RecordLiteral:
		fields := make([]*ast.RecordField, 0, len(node.Fields))
		for _, field := range node.Fields {
			fields = append(fields, &ast.RecordField{
				SourceSpan: field.SourceSpan,
				Name:       cloneIdentifier(field.Name),
				Value:      instantiateMutationExpr(field.Value, captures),
			})
		}
		return &ast.RecordLiteral{SourceSpan: node.SourceSpan, Fields: fields}
	case *ast.RecordBindingPattern:
		fields := make([]*ast.RecordBindingField, 0, len(node.Fields))
		for _, field := range node.Fields {
			fields = append(fields, &ast.RecordBindingField{
				SourceSpan: field.SourceSpan,
				Name:       cloneIdentifier(field.Name),
				Value:      instantiateMutationExpr(field.Value, captures).(ast.BindingPattern),
			})
		}
		return &ast.RecordBindingPattern{SourceSpan: node.SourceSpan, Fields: fields}
	case *ast.BlockExpr:
		expressions := make([]ast.Expr, 0, len(node.Expressions))
		for _, item := range node.Expressions {
			expressions = append(expressions, instantiateMutationExpr(item, captures))
		}
		return &ast.BlockExpr{SourceSpan: node.SourceSpan, Expressions: expressions}
	case *ast.AssignmentExpr:
		return &ast.AssignmentExpr{
			SourceSpan: node.SourceSpan,
			Target:     instantiateMutationExpr(node.Target, captures).(ast.AssignmentTarget),
			Value:      instantiateMutationExpr(node.Value, captures),
		}
	case *ast.IndexExpr:
		return &ast.IndexExpr{
			SourceSpan: node.SourceSpan,
			Target:     instantiateMutationExpr(node.Target, captures),
			Index:      instantiateMutationExpr(node.Index, captures),
		}
	case *ast.FieldAccessExpr:
		return &ast.FieldAccessExpr{
			SourceSpan: node.SourceSpan,
			Target:     instantiateMutationExpr(node.Target, captures),
			Name:       cloneIdentifier(node.Name),
		}
	case *ast.UnaryExpr:
		return &ast.UnaryExpr{
			SourceSpan: node.SourceSpan,
			Operator:   node.Operator,
			Operand:    instantiateMutationExpr(node.Operand, captures),
		}
	case *ast.BinaryExpr:
		return &ast.BinaryExpr{
			SourceSpan: node.SourceSpan,
			Left:       instantiateMutationExpr(node.Left, captures),
			Operator:   node.Operator,
			Right:      instantiateMutationExpr(node.Right, captures),
		}
	case *ast.ConditionalExpr:
		return &ast.ConditionalExpr{
			SourceSpan: node.SourceSpan,
			Condition:  instantiateMutationExpr(node.Condition, captures),
			ThenBranch: instantiateMutationExpr(node.ThenBranch, captures),
			ElseBranch: instantiateMutationExpr(node.ElseBranch, captures),
		}
	case *ast.WhileExpr:
		return &ast.WhileExpr{
			SourceSpan: node.SourceSpan,
			Condition:  instantiateMutationExpr(node.Condition, captures),
			Body:       instantiateMutationExpr(node.Body, captures),
		}
	case *ast.MatchExpr:
		cases := make([]*ast.MatchCase, 0, len(node.Cases))
		for _, matchCase := range node.Cases {
			cases = append(cases, &ast.MatchCase{
				SourceSpan: matchCase.SourceSpan,
				Pattern:    instantiateMutationExpr(matchCase.Pattern, captures),
				Branch:     instantiateMutationExpr(matchCase.Branch, captures),
			})
		}
		return &ast.MatchExpr{
			SourceSpan: node.SourceSpan,
			Subject:    instantiateMutationExpr(node.Subject, captures),
			Cases:      cases,
		}
	case *ast.ForInExpr:
		return &ast.ForInExpr{
			SourceSpan: node.SourceSpan,
			Binding:    instantiateMutationExpr(node.Binding, captures).(ast.BindingPattern),
			Iterable:   instantiateMutationExpr(node.Iterable, captures),
			Body:       instantiateMutationExpr(node.Body, captures),
		}
	case *ast.CallExpr:
		args := make([]ast.Expr, 0, len(node.Arguments))
		for _, argument := range node.Arguments {
			args = append(args, instantiateMutationExpr(argument, captures))
		}
		return &ast.CallExpr{
			SourceSpan: node.SourceSpan,
			Callee:     instantiateMutationExpr(node.Callee, captures),
			Arguments:  args,
		}
	case *ast.NamedFunctionExpr:
		return &ast.NamedFunctionExpr{
			SourceSpan: node.SourceSpan,
			Name:       cloneIdentifier(node.Name),
			Parameters: cloneIdentifiers(node.Parameters),
			Body:       instantiateMutationExpr(node.Body, captures),
		}
	case *ast.FunctionLiteralExpr:
		return &ast.FunctionLiteralExpr{
			SourceSpan: node.SourceSpan,
			Parameters: cloneIdentifiers(node.Parameters),
			Body:       instantiateMutationExpr(node.Body, captures),
		}
	case *ast.QuoteExpr:
		return &ast.QuoteExpr{SourceSpan: node.SourceSpan, Body: instantiateMutationExpr(node.Body, captures)}
	case *ast.UnquoteExpr:
		return &ast.UnquoteExpr{SourceSpan: node.SourceSpan, Expression: instantiateMutationExpr(node.Expression, captures)}
	case *ast.SpliceExpr:
		return &ast.SpliceExpr{SourceSpan: node.SourceSpan, Expression: instantiateMutationExpr(node.Expression, captures)}
	default:
		panic(fmt.Sprintf("unsupported mutation replacement expression type %T", expr))
	}
}

func matchMutationExprSlice(patterns, targets []ast.Expr, captures mutationCaptures) bool {
	if len(patterns) != len(targets) {
		return false
	}

	for i := range patterns {
		if !matchMutationExpr(patterns[i], targets[i], captures) {
			return false
		}
	}

	return true
}

func matchMutationIdentifiers(patterns, targets []*ast.Identifier, captures mutationCaptures) bool {
	if len(patterns) != len(targets) {
		return false
	}

	for i := range patterns {
		if !matchMutationExpr(patterns[i], targets[i], captures) {
			return false
		}
	}

	return true
}

func matchMutationBindingPatterns(patterns, targets []ast.BindingPattern, captures mutationCaptures) bool {
	if len(patterns) != len(targets) {
		return false
	}

	for i := range patterns {
		if !matchMutationExpr(patterns[i], targets[i], captures) {
			return false
		}
	}

	return true
}

func matchMutationRecordFields(patterns, targets []*ast.RecordField, captures mutationCaptures) bool {
	if len(patterns) != len(targets) {
		return false
	}

	for i := range patterns {
		if !matchMutationExpr(patterns[i].Name, targets[i].Name, captures) || !matchMutationExpr(patterns[i].Value, targets[i].Value, captures) {
			return false
		}
	}

	return true
}

func matchMutationRecordBindingFields(patterns, targets []*ast.RecordBindingField, captures mutationCaptures) bool {
	if len(patterns) != len(targets) {
		return false
	}

	for i := range patterns {
		if !matchMutationExpr(patterns[i].Name, targets[i].Name, captures) || !matchMutationExpr(patterns[i].Value, targets[i].Value, captures) {
			return false
		}
	}

	return true
}

func matchMutationCases(patterns, targets []*ast.MatchCase, captures mutationCaptures) bool {
	if len(patterns) != len(targets) {
		return false
	}

	for i := range patterns {
		if !matchMutationExpr(patterns[i].Pattern, targets[i].Pattern, captures) || !matchMutationExpr(patterns[i].Branch, targets[i].Branch, captures) {
			return false
		}
	}

	return true
}

func matchMutationRules(patterns, targets []*ast.MutationRule, captures mutationCaptures) bool {
	if len(patterns) != len(targets) {
		return false
	}

	for i := range patterns {
		if !matchMutationExpr(patterns[i].Pattern, targets[i].Pattern, captures) || !matchMutationExpr(patterns[i].Replacement, targets[i].Replacement, captures) {
			return false
		}
	}

	return true
}
