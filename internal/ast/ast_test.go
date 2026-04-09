package ast

import (
	"testing"

	"molt/internal/source"
)

var (
	_ Expr             = (*NumberLiteral)(nil)
	_ Expr             = (*StringLiteral)(nil)
	_ Expr             = (*BooleanLiteral)(nil)
	_ Expr             = (*NilLiteral)(nil)
	_ Expr             = (*OperatorLiteral)(nil)
	_ Expr             = (*Identifier)(nil)
	_ Expr             = (*GroupExpr)(nil)
	_ Expr             = (*ListLiteral)(nil)
	_ Expr             = (*RecordLiteral)(nil)
	_ Expr             = (*FieldAccessExpr)(nil)
	_ Expr             = (*BlockExpr)(nil)
	_ Expr             = (*AssignmentExpr)(nil)
	_ Expr             = (*IndexExpr)(nil)
	_ Expr             = (*UnaryExpr)(nil)
	_ Expr             = (*BinaryExpr)(nil)
	_ Expr             = (*ConditionalExpr)(nil)
	_ Expr             = (*WhileExpr)(nil)
	_ Expr             = (*TryCatchExpr)(nil)
	_ Expr             = (*MatchExpr)(nil)
	_ Expr             = (*ForInExpr)(nil)
	_ Expr             = (*BreakExpr)(nil)
	_ Expr             = (*ContinueExpr)(nil)
	_ Expr             = (*ExportExpr)(nil)
	_ Expr             = (*ImportExpr)(nil)
	_ Expr             = (*CallExpr)(nil)
	_ Expr             = (*NamedFunctionExpr)(nil)
	_ Expr             = (*FunctionLiteralExpr)(nil)
	_ Expr             = (*QuoteExpr)(nil)
	_ Expr             = (*UnquoteExpr)(nil)
	_ Expr             = (*SpliceExpr)(nil)
	_ Expr             = (*MutationLiteralExpr)(nil)
	_ Expr             = (*ApplyMutationExpr)(nil)
	_ Expr             = (*ListBindingPattern)(nil)
	_ Expr             = (*RecordBindingPattern)(nil)
	_ BindingPattern   = (*Identifier)(nil)
	_ BindingPattern   = (*ListBindingPattern)(nil)
	_ BindingPattern   = (*RecordBindingPattern)(nil)
	_ AssignmentTarget = (*Identifier)(nil)
	_ AssignmentTarget = (*ListBindingPattern)(nil)
	_ AssignmentTarget = (*RecordBindingPattern)(nil)
	_ AssignmentTarget = (*FieldAccessExpr)(nil)
	_ Node             = (*MatchCase)(nil)
	_ Node             = (*RecordBindingField)(nil)
	_ Node             = (*MutationRule)(nil)
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
	recordField := &RecordField{SourceSpan: span, Name: ident, Value: number}
	record := &RecordLiteral{SourceSpan: span, Fields: []*RecordField{recordField}}
	fieldAccess := &FieldAccessExpr{SourceSpan: span, Target: ident, Name: &Identifier{SourceSpan: span, Name: "name"}}
	listBinding := &ListBindingPattern{SourceSpan: span, Elements: []BindingPattern{ident}}
	recordBindingField := &RecordBindingField{SourceSpan: span, Name: ident, Value: listBinding}
	recordBinding := &RecordBindingPattern{SourceSpan: span, Fields: []*RecordBindingField{recordBindingField}}

	assertSpan(t, number, span)
	assertSpan(t, str, span)
	assertSpan(t, boolean, span)
	assertSpan(t, nilValue, span)
	assertSpan(t, operator, span)
	assertSpan(t, ident, span)
	assertSpan(t, group, span)
	assertSpan(t, list, span)
	assertSpan(t, recordField, span)
	assertSpan(t, record, span)
	assertSpan(t, fieldAccess, span)
	assertSpan(t, listBinding, span)
	assertSpan(t, recordBindingField, span)
	assertSpan(t, recordBinding, span)

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

	if len(record.Fields) != 1 || record.Fields[0] != recordField {
		t.Fatalf("record field was not preserved")
	}

	if fieldAccess.Target != ident || fieldAccess.Name.Name != "name" {
		t.Fatalf("field access shape was not preserved")
	}

	if len(listBinding.Elements) != 1 || listBinding.Elements[0] != ident {
		t.Fatalf("list binding shape was not preserved")
	}

	if len(recordBinding.Fields) != 1 || recordBinding.Fields[0] != recordBindingField {
		t.Fatalf("record binding field was not preserved")
	}
}

func TestStructuredExpressionNodesPreserveChildrenAndOperators(t *testing.T) {
	file := source.NewFile("ast.molt", "dummy")
	span := file.MustSpan(0, 5)
	name := &Identifier{SourceSpan: span, Name: "x"}
	fieldName := &Identifier{SourceSpan: span, Name: "count"}
	value := &NumberLiteral{SourceSpan: span, Value: 1}
	other := &Identifier{SourceSpan: span, Name: "xs"}
	fieldTarget := &FieldAccessExpr{SourceSpan: span, Target: other, Name: fieldName}
	listTarget := &ListBindingPattern{SourceSpan: span, Elements: []BindingPattern{name, other}}
	recordTarget := &RecordBindingPattern{
		SourceSpan: span,
		Fields: []*RecordBindingField{
			{
				SourceSpan: span,
				Name:       fieldName,
				Value:      name,
			},
		},
	}

	block := &BlockExpr{SourceSpan: span, Expressions: []Expr{name, value}}
	assign := &AssignmentExpr{SourceSpan: span, Target: name, Value: value}
	fieldAssign := &AssignmentExpr{SourceSpan: span, Target: fieldTarget, Value: value}
	listAssign := &AssignmentExpr{SourceSpan: span, Target: listTarget, Value: value}
	recordAssign := &AssignmentExpr{SourceSpan: span, Target: recordTarget, Value: value}
	index := &IndexExpr{SourceSpan: span, Target: other, Index: value}
	unary := &UnaryExpr{SourceSpan: span, Operator: UnaryNegate, Operand: value}
	binary := &BinaryExpr{SourceSpan: span, Left: name, Operator: BinaryAdd, Right: value}
	conditional := &ConditionalExpr{
		SourceSpan: span,
		Condition:  &BooleanLiteral{SourceSpan: span, Value: true},
		ThenBranch: name,
		ElseBranch: value,
	}
	whileExpr := &WhileExpr{
		SourceSpan: span,
		Condition:  &BooleanLiteral{SourceSpan: span, Value: true},
		Body:       assign,
	}
	tryExpr := &TryCatchExpr{
		SourceSpan:   span,
		Body:         assign,
		CatchBinding: &Identifier{SourceSpan: span, Name: "err"},
		CatchBranch:  value,
	}
	matchCase := &MatchCase{
		SourceSpan: span,
		Pattern:    &Identifier{SourceSpan: span, Name: "_"},
		Branch:     value,
	}
	matchExpr := &MatchExpr{
		SourceSpan: span,
		Subject:    name,
		Cases:      []*MatchCase{matchCase},
	}
	forExpr := &ForInExpr{
		SourceSpan: span,
		Binding:    listTarget,
		Iterable:   other,
		Body:       assign,
	}
	breakExpr := &BreakExpr{SourceSpan: span}
	continueExpr := &ContinueExpr{SourceSpan: span}

	assertSpan(t, block, span)
	assertSpan(t, assign, span)
	assertSpan(t, fieldAssign, span)
	assertSpan(t, listAssign, span)
	assertSpan(t, recordAssign, span)
	assertSpan(t, index, span)
	assertSpan(t, unary, span)
	assertSpan(t, binary, span)
	assertSpan(t, conditional, span)
	assertSpan(t, whileExpr, span)
	assertSpan(t, tryExpr, span)
	assertSpan(t, matchCase, span)
	assertSpan(t, matchExpr, span)
	assertSpan(t, forExpr, span)
	assertSpan(t, breakExpr, span)
	assertSpan(t, continueExpr, span)

	if len(block.Expressions) != 2 {
		t.Fatalf("block expression count = %d, want 2", len(block.Expressions))
	}

	if assign.Target != name {
		t.Fatalf("assignment target was not preserved")
	}

	if fieldAssign.Target != fieldTarget {
		t.Fatalf("field assignment target was not preserved")
	}

	if listAssign.Target != listTarget {
		t.Fatalf("list destructuring target was not preserved")
	}

	if recordAssign.Target != recordTarget {
		t.Fatalf("record destructuring target was not preserved")
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

	if whileExpr.Condition == nil || whileExpr.Body != assign {
		t.Fatalf("while shape was not preserved")
	}

	if tryExpr.Body != assign || tryExpr.CatchBinding.Name != "err" || tryExpr.CatchBranch != value {
		t.Fatalf("try/catch shape was not preserved")
	}

	if matchExpr.Subject != name || len(matchExpr.Cases) != 1 || matchExpr.Cases[0] != matchCase {
		t.Fatalf("match shape was not preserved")
	}

	if forExpr.Binding != listTarget || forExpr.Iterable != other || forExpr.Body != assign {
		t.Fatalf("for-in shape was not preserved")
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
	unquote := &UnquoteExpr{
		SourceSpan: span,
		Expression: paramA,
	}
	splice := &SpliceExpr{
		SourceSpan: span,
		Expression: call,
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
	assertSpan(t, unquote, span)
	assertSpan(t, splice, span)
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

	if unquote.Expression != paramA {
		t.Fatalf("unquote expression was not preserved")
	}

	if splice.Expression != call {
		t.Fatalf("splice expression was not preserved")
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
