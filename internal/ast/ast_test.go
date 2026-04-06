package ast

import (
	"testing"

	"molt/internal/source"
)

var (
	_ Expr = (*NumberLiteral)(nil)
	_ Expr = (*StringLiteral)(nil)
	_ Expr = (*BooleanLiteral)(nil)
	_ Expr = (*NilLiteral)(nil)
	_ Expr = (*OperatorLiteral)(nil)
	_ Expr = (*Identifier)(nil)
	_ Expr = (*GroupExpr)(nil)
	_ Expr = (*ListLiteral)(nil)
	_ Expr = (*BlockExpr)(nil)
	_ Expr = (*AssignmentExpr)(nil)
	_ Expr = (*IndexExpr)(nil)
	_ Expr = (*UnaryExpr)(nil)
	_ Expr = (*BinaryExpr)(nil)
	_ Expr = (*ConditionalExpr)(nil)
	_ Expr = (*ExportExpr)(nil)
	_ Expr = (*ImportExpr)(nil)
	_ Expr = (*CallExpr)(nil)
	_ Expr = (*NamedFunctionExpr)(nil)
	_ Expr = (*FunctionLiteralExpr)(nil)
	_ Expr = (*QuoteExpr)(nil)
	_ Expr = (*MutationLiteralExpr)(nil)
	_ Expr = (*ApplyMutationExpr)(nil)
	_ Node = (*MutationRule)(nil)
)

func TestLiteralIdentifierAndListNodesPreserveSpansAndPayloads(t *testing.T) {
	file := source.NewFile("ast.molt", "dummy")
	span := file.MustSpan(0, 5)
	number := &NumberLiteral{SourceSpan: span, Value: 3.14}
	str := &StringLiteral{SourceSpan: span, Value: "hello"}
	boolean := &BooleanLiteral{SourceSpan: span, Value: true}
	nilValue := &NilLiteral{SourceSpan: span}
	operator := &OperatorLiteral{SourceSpan: span, Symbol: "+"}
	ident := &Identifier{SourceSpan: span, Name: "value"}
	group := &GroupExpr{SourceSpan: span, Inner: ident}
	list := &ListLiteral{SourceSpan: span, Elements: []Expr{number, str, boolean, nilValue, operator, ident, group}}

	assertSpan(t, number, span)
	assertSpan(t, str, span)
	assertSpan(t, boolean, span)
	assertSpan(t, nilValue, span)
	assertSpan(t, operator, span)
	assertSpan(t, ident, span)
	assertSpan(t, group, span)
	assertSpan(t, list, span)

	if number.Value != 3.14 {
		t.Fatalf("number value = %v, want 3.14", number.Value)
	}

	if str.Value != "hello" {
		t.Fatalf("string value = %q, want %q", str.Value, "hello")
	}

	if !boolean.Value {
		t.Fatalf("boolean value = false, want true")
	}

	if ident.Name != "value" {
		t.Fatalf("identifier name = %q, want %q", ident.Name, "value")
	}

	if operator.Symbol != "+" {
		t.Fatalf("operator symbol = %q, want %q", operator.Symbol, "+")
	}

	if group.Inner != ident {
		t.Fatalf("group inner was not preserved")
	}

	if len(list.Elements) != 7 {
		t.Fatalf("list element count = %d, want 7", len(list.Elements))
	}
}

func TestStructuredExpressionNodesPreserveChildrenAndOperators(t *testing.T) {
	file := source.NewFile("ast.molt", "dummy")
	span := file.MustSpan(0, 5)
	name := &Identifier{SourceSpan: span, Name: "x"}
	value := &NumberLiteral{SourceSpan: span, Value: 1}
	other := &Identifier{SourceSpan: span, Name: "xs"}

	block := &BlockExpr{SourceSpan: span, Expressions: []Expr{name, value}}
	assign := &AssignmentExpr{SourceSpan: span, Target: name, Value: value}
	index := &IndexExpr{SourceSpan: span, Target: other, Index: value}
	unary := &UnaryExpr{SourceSpan: span, Operator: UnaryNegate, Operand: value}
	binary := &BinaryExpr{SourceSpan: span, Left: name, Operator: BinaryAdd, Right: value}
	conditional := &ConditionalExpr{
		SourceSpan: span,
		Condition:  &BooleanLiteral{SourceSpan: span, Value: true},
		ThenBranch: name,
		ElseBranch: value,
	}

	assertSpan(t, block, span)
	assertSpan(t, assign, span)
	assertSpan(t, index, span)
	assertSpan(t, unary, span)
	assertSpan(t, binary, span)
	assertSpan(t, conditional, span)

	if len(block.Expressions) != 2 {
		t.Fatalf("block expression count = %d, want 2", len(block.Expressions))
	}

	if assign.Target != name {
		t.Fatalf("assignment target was not preserved")
	}

	if index.Target != other || index.Index != value {
		t.Fatalf("index children were not preserved")
	}

	if unary.Operator != UnaryNegate {
		t.Fatalf("unary operator = %q, want %q", unary.Operator, UnaryNegate)
	}

	if binary.Operator != BinaryAdd {
		t.Fatalf("binary operator = %q, want %q", binary.Operator, BinaryAdd)
	}

	if conditional.ThenBranch != name || conditional.ElseBranch != value {
		t.Fatalf("conditional branches were not preserved")
	}
}

func TestFunctionCallQuoteAndMutationNodesPreserveShape(t *testing.T) {
	file := source.NewFile("ast.molt", "dummy")
	span := file.MustSpan(0, 5)

	name := &Identifier{SourceSpan: span, Name: "warp"}
	exportName := &Identifier{SourceSpan: span, Name: "warp"}
	importPath := &StringLiteral{SourceSpan: span, Value: "./lib.molt"}
	paramA := &Identifier{SourceSpan: span, Name: "code"}
	paramB := &Identifier{SourceSpan: span, Name: "times"}
	body := &BinaryExpr{
		SourceSpan: span,
		Left:       paramB,
		Operator:   BinaryMultiply,
		Right:      &NumberLiteral{SourceSpan: span, Value: 2},
	}

	namedFn := &NamedFunctionExpr{
		SourceSpan: span,
		Name:       name,
		Parameters: []*Identifier{paramA, paramB},
		Body:       body,
	}

	anonFn := &FunctionLiteralExpr{
		SourceSpan: span,
		Parameters: []*Identifier{paramA},
		Body:       body,
	}

	call := &CallExpr{
		SourceSpan: span,
		Callee:     name,
		Arguments:  []Expr{paramA, body},
	}

	importExpr := &ImportExpr{
		SourceSpan: span,
		Path:       importPath,
	}

	exportExpr := &ExportExpr{
		SourceSpan: span,
		Name:       exportName,
	}

	quote := &QuoteExpr{
		SourceSpan: span,
		Body:       body,
	}

	ruleOne := &MutationRule{
		SourceSpan:  span,
		Pattern:     &Identifier{SourceSpan: span, Name: "x"},
		Replacement: &Identifier{SourceSpan: span, Name: "y"},
	}

	ruleTwo := &MutationRule{
		SourceSpan:  span,
		Pattern:     &BinaryExpr{SourceSpan: span, Left: paramA, Operator: BinaryAdd, Right: paramB},
		Replacement: &BinaryExpr{SourceSpan: span, Left: paramA, Operator: BinaryMultiply, Right: paramB},
	}

	mutation := &MutationLiteralExpr{
		SourceSpan: span,
		Rules:      []*MutationRule{ruleOne, ruleTwo},
	}

	applied := &ApplyMutationExpr{
		SourceSpan: span,
		Target:     quote,
		Mutation:   mutation,
	}

	assertSpan(t, namedFn, span)
	assertSpan(t, anonFn, span)
	assertSpan(t, exportExpr, span)
	assertSpan(t, importExpr, span)
	assertSpan(t, call, span)
	assertSpan(t, quote, span)
	assertSpan(t, ruleOne, span)
	assertSpan(t, mutation, span)
	assertSpan(t, applied, span)

	if namedFn.Name != name {
		t.Fatalf("named function name was not preserved")
	}

	if len(namedFn.Parameters) != 2 || namedFn.Body != body {
		t.Fatalf("named function shape was not preserved")
	}

	if len(anonFn.Parameters) != 1 || anonFn.Body != body {
		t.Fatalf("anonymous function shape was not preserved")
	}

	if exportExpr.Name != exportName {
		t.Fatalf("export name was not preserved")
	}

	if importExpr.Path != importPath {
		t.Fatalf("import path was not preserved")
	}

	if len(call.Arguments) != 2 || call.Callee != name {
		t.Fatalf("call shape was not preserved")
	}

	if quote.Body != body {
		t.Fatalf("quote body was not preserved")
	}

	if len(mutation.Rules) != 2 || mutation.Rules[0] != ruleOne || mutation.Rules[1] != ruleTwo {
		t.Fatalf("mutation rules were not preserved in order")
	}

	if applied.Target != quote || applied.Mutation != mutation {
		t.Fatalf("postfix mutation application shape was not preserved")
	}
}

func assertSpan(t *testing.T, node Node, want source.Span) {
	t.Helper()

	if got := node.Span(); got != want {
		t.Fatalf("Span() = %+v, want %+v", got, want)
	}
}
