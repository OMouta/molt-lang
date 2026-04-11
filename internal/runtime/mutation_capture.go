package runtime

import (
	"fmt"

	"molt/internal/ast"
)

type mutationCaptureKind string

const (
	mutationCaptureSingle mutationCaptureKind = "single"
	mutationCaptureRest   mutationCaptureKind = "rest"
)

type mutationCaptureSet map[string]mutationCaptureKind

type mutationCaptures struct {
	singles map[string]ast.Expr
	rests   map[string][]ast.Expr
}

func newMutationCaptures() mutationCaptures {
	return mutationCaptures{
		singles: make(map[string]ast.Expr),
		rests:   make(map[string][]ast.Expr),
	}
}

func collectMutationCaptures(expr ast.Expr, captures mutationCaptureSet) error {
	if expr == nil {
		return nil
	}

	switch node := expr.(type) {
	case *ast.CommentExpr:
		return nil
	case *ast.MutationCaptureExpr:
		return recordMutationCapture(captures, mutationCaptureSingle, node.Name, "capture")
	case *ast.MutationWildcardExpr:
		return nil
	case *ast.MutationRestCaptureExpr:
		return recordMutationCapture(captures, mutationCaptureRest, node.Name, "rest capture")
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
		return collectMutationCaptureSlice(node.Elements, captures)
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
		return collectMutationCaptureSlice(node.Expressions, captures)
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
		return collectMutationCaptureSlice(node.Arguments, captures)
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

func recordMutationCapture(captures mutationCaptureSet, kind mutationCaptureKind, name *ast.Identifier, label string) error {
	if name == nil {
		return fmt.Errorf("%s name cannot be nil", label)
	}
	if name.Name == "_" {
		return fmt.Errorf("%s name cannot be '_'", label)
	}
	if existing, ok := captures[name.Name]; ok && existing != kind {
		return fmt.Errorf("capture %q cannot be used as both single and rest capture", name.Name)
	}
	captures[name.Name] = kind
	return nil
}

func collectMutationCaptureSlice(items []ast.Expr, captures mutationCaptureSet) error {
	for _, item := range items {
		if err := collectMutationCaptures(item, captures); err != nil {
			return err
		}
	}

	return nil
}

func matchMutationExpr(pattern, target ast.Expr, captures mutationCaptures) bool {
	if pattern == nil || target == nil {
		return pattern == nil && target == nil
	}

	switch p := pattern.(type) {
	case *ast.CommentExpr:
		t, ok := target.(*ast.CommentExpr)
		return ok && p.Text == t.Text
	case *ast.MutationCaptureExpr:
		if p.Name == nil {
			return false
		}
		return captures.bindSingle(p.Name.Name, target)
	case *ast.MutationWildcardExpr:
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

func (c mutationCaptures) bindSingle(name string, expr ast.Expr) bool {
	if _, seen := c.rests[name]; seen {
		return false
	}
	if existing, seen := c.singles[name]; seen {
		return EqualExpr(existing, expr)
	}
	c.singles[name] = expr
	return true
}

func (c mutationCaptures) bindRest(name string, exprs []ast.Expr) bool {
	if _, seen := c.singles[name]; seen {
		return false
	}
	if existing, seen := c.rests[name]; seen {
		return equalExprSlices(existing, exprs)
	}
	c.rests[name] = cloneExprs(exprs)
	return true
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
		captured, exists := captures.singles[node.Name.Name]
		if !exists {
			panic(fmt.Sprintf("missing mutation capture %q during replacement instantiation", node.Name.Name))
		}
		return CloneExpr(captured)
	case *ast.MutationWildcardExpr:
		panic("mutation wildcard is not valid in replacement expressions")
	case *ast.MutationRestCaptureExpr:
		panic("mutation rest capture must be instantiated in a sequence position")
	case *ast.CommentExpr,
		*ast.OperatorLiteral,
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
		return &ast.ListLiteral{SourceSpan: node.SourceSpan, Elements: instantiateMutationExprSlice(node.Elements, captures)}
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
		return &ast.BlockExpr{SourceSpan: node.SourceSpan, Expressions: instantiateMutationExprSlice(node.Expressions, captures)}
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
		args := instantiateMutationExprSlice(node.Arguments, captures)
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

func instantiateMutationExprSlice(items []ast.Expr, captures mutationCaptures) []ast.Expr {
	instantiated := make([]ast.Expr, 0, len(items))
	for _, item := range items {
		if rest, ok := item.(*ast.MutationRestCaptureExpr); ok {
			if rest.Name == nil {
				panic("missing mutation rest capture name during replacement instantiation")
			}
			captured, exists := captures.rests[rest.Name.Name]
			if !exists {
				panic(fmt.Sprintf("missing mutation rest capture %q during replacement instantiation", rest.Name.Name))
			}
			instantiated = append(instantiated, cloneExprs(captured)...)
			continue
		}

		instantiated = append(instantiated, instantiateMutationExpr(item, captures))
	}

	return instantiated
}

func matchMutationExprSlice(patterns, targets []ast.Expr, captures mutationCaptures) bool {
	restIndex := -1
	var rest *ast.MutationRestCaptureExpr
	for i, pattern := range patterns {
		if current, ok := pattern.(*ast.MutationRestCaptureExpr); ok {
			restIndex = i
			rest = current
			break
		}
	}

	if restIndex < 0 {
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

	required := len(patterns) - 1
	if len(targets) < required {
		return false
	}

	for i := 0; i < restIndex; i++ {
		if !matchMutationExpr(patterns[i], targets[i], captures) {
			return false
		}
	}

	suffixCount := len(patterns) - restIndex - 1
	suffixStart := len(targets) - suffixCount
	for i := 0; i < suffixCount; i++ {
		if !matchMutationExpr(patterns[restIndex+1+i], targets[suffixStart+i], captures) {
			return false
		}
	}

	if rest == nil || rest.Name == nil {
		return false
	}

	return captures.bindRest(rest.Name.Name, targets[restIndex:suffixStart])
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
