package runtime

import (
	"testing"

	"molt/internal/ast"
	"molt/internal/source"
)

func TestValidateMutationRuleRejectsUnsupportedForms(t *testing.T) {
	span := helperSpan()

	tests := []struct {
		name string
		rule *ast.MutationRule
	}{
		{
			name: "operator to non-operator",
			rule: &ast.MutationRule{
				SourceSpan:  span,
				Pattern:     &ast.OperatorLiteral{SourceSpan: span, Symbol: "+"},
				Replacement: &ast.NumberLiteral{SourceSpan: span, Value: 1},
			},
		},
		{
			name: "non-operator to operator",
			rule: &ast.MutationRule{
				SourceSpan:  span,
				Pattern:     &ast.NumberLiteral{SourceSpan: span, Value: 1},
				Replacement: &ast.OperatorLiteral{SourceSpan: span, Symbol: "+"},
			},
		},
		{
			name: "nested mutation literal",
			rule: &ast.MutationRule{
				SourceSpan: span,
				Pattern: &ast.MutationLiteralExpr{
					SourceSpan: span,
					Rules: []*ast.MutationRule{
						{
							SourceSpan:  span,
							Pattern:     &ast.OperatorLiteral{SourceSpan: span, Symbol: "+"},
							Replacement: &ast.OperatorLiteral{SourceSpan: span, Symbol: "*"},
						},
					},
				},
				Replacement: &ast.NumberLiteral{SourceSpan: span, Value: 1},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateMutationRule(tc.rule); err == nil {
				t.Fatalf("expected validation error, got nil")
			}
		})
	}
}

func TestApplyRuleSupportsOperatorIdentifierLiteralAndExactSubtreeReplacement(t *testing.T) {
	expr := binary(
		binary(identifier("x"), "+", number(1)),
		"+",
		group(binary(identifier("a"), "*", identifier("b"))),
	)

	mutation := &MutationValue{
		Rules: []*ast.MutationRule{
			rule(operator("+"), operator("*")),
			rule(identifier("x"), identifier("y")),
			rule(number(1), number(2)),
			rule(group(binary(identifier("a"), "*", identifier("b"))), group(binary(identifier("a"), "-", identifier("b")))),
		},
	}

	rewritten, err := Rewrite(expr, mutation)
	if err != nil {
		t.Fatalf("Rewrite returned error: %v", err)
	}

	root := expectExpr[*ast.BinaryExpr](t, rewritten)
	if root.Operator != ast.BinaryMultiply {
		t.Fatalf("root operator = %q, want %q", root.Operator, ast.BinaryMultiply)
	}

	left := expectExpr[*ast.BinaryExpr](t, root.Left)
	if left.Operator != ast.BinaryMultiply {
		t.Fatalf("left operator = %q, want %q", left.Operator, ast.BinaryMultiply)
	}

	leftIdent := expectExpr[*ast.Identifier](t, left.Left)
	if leftIdent.Name != "y" {
		t.Fatalf("left identifier = %q, want %q", leftIdent.Name, "y")
	}

	leftNumber := expectExpr[*ast.NumberLiteral](t, left.Right)
	if leftNumber.Value != 2 {
		t.Fatalf("left literal = %v, want 2", leftNumber.Value)
	}

	right := expectExpr[*ast.GroupExpr](t, root.Right)
	rightInner := expectExpr[*ast.BinaryExpr](t, right.Inner)
	if rightInner.Operator != ast.BinarySubtract {
		t.Fatalf("right operator = %q, want %q", rightInner.Operator, ast.BinarySubtract)
	}
}

func TestApplyRuleDoesNotRematchReplacementNodesInSamePass(t *testing.T) {
	expr := identifier("x")
	rewritten, err := ApplyRule(expr, rule(identifier("x"), binary(identifier("x"), "+", number(1))))
	if err != nil {
		t.Fatalf("ApplyRule returned error: %v", err)
	}

	bin := expectExpr[*ast.BinaryExpr](t, rewritten)
	left := expectExpr[*ast.Identifier](t, bin.Left)
	if left.Name != "x" {
		t.Fatalf("left identifier = %q, want %q", left.Name, "x")
	}
}

func TestRewriteAppliesRulesInOrder(t *testing.T) {
	expr := identifier("x")
	mutation := &MutationValue{
		Rules: []*ast.MutationRule{
			rule(identifier("x"), number(1)),
			rule(number(1), number(2)),
		},
	}

	rewritten, err := Rewrite(expr, mutation)
	if err != nil {
		t.Fatalf("Rewrite returned error: %v", err)
	}

	number := expectExpr[*ast.NumberLiteral](t, rewritten)
	if number.Value != 2 {
		t.Fatalf("result = %v, want 2", number.Value)
	}
}

func TestRewriteTraversesParentBeforeChildren(t *testing.T) {
	expr := binary(number(1), "+", number(2))
	rewritten, err := ApplyRule(expr, rule(binary(number(1), "+", number(2)), number(9)))
	if err != nil {
		t.Fatalf("ApplyRule returned error: %v", err)
	}

	number := expectExpr[*ast.NumberLiteral](t, rewritten)
	if number.Value != 9 {
		t.Fatalf("result = %v, want 9", number.Value)
	}
}

func TestRewritePreservesOriginalAstImmutability(t *testing.T) {
	original := binary(identifier("x"), "+", number(1))
	rewritten, err := Rewrite(original, &MutationValue{
		Rules: []*ast.MutationRule{
			rule(operator("+"), operator("*")),
			rule(identifier("x"), identifier("y")),
		},
	})
	if err != nil {
		t.Fatalf("Rewrite returned error: %v", err)
	}

	originalBinary := expectExpr[*ast.BinaryExpr](t, original)
	if originalBinary.Operator != ast.BinaryAdd {
		t.Fatalf("original operator mutated to %q", originalBinary.Operator)
	}

	originalLeft := expectExpr[*ast.Identifier](t, originalBinary.Left)
	if originalLeft.Name != "x" {
		t.Fatalf("original identifier mutated to %q", originalLeft.Name)
	}

	rewrittenBinary := expectExpr[*ast.BinaryExpr](t, rewritten)
	if rewrittenBinary.Operator != ast.BinaryMultiply {
		t.Fatalf("rewritten operator = %q, want %q", rewrittenBinary.Operator, ast.BinaryMultiply)
	}
}

func TestRewriteTraversesRecordLiteralsAndFieldAccess(t *testing.T) {
	span := helperSpan()
	expr := &ast.RecordLiteral{
		SourceSpan: span,
		Fields: []*ast.RecordField{
			{
				SourceSpan: span,
				Name:       identifier("name"),
				Value: &ast.FieldAccessExpr{
					SourceSpan: span,
					Target:     identifier("profile"),
					Name:       identifier("name"),
				},
			},
		},
	}

	rewritten, err := Rewrite(expr, &MutationValue{
		Rules: []*ast.MutationRule{
			rule(identifier("name"), identifier("label")),
			rule(identifier("profile"), identifier("user")),
		},
	})
	if err != nil {
		t.Fatalf("Rewrite returned error: %v", err)
	}

	record := expectExpr[*ast.RecordLiteral](t, rewritten)
	if len(record.Fields) != 1 {
		t.Fatalf("field count = %d, want 1", len(record.Fields))
	}

	if record.Fields[0].Name.Name != "label" {
		t.Fatalf("record field name = %q, want %q", record.Fields[0].Name.Name, "label")
	}

	access := expectExpr[*ast.FieldAccessExpr](t, record.Fields[0].Value)
	target := expectExpr[*ast.Identifier](t, access.Target)
	if target.Name != "user" || access.Name.Name != "label" {
		t.Fatalf("field access rewrite mismatch")
	}
}

func identifier(name string) *ast.Identifier {
	span := helperSpan()
	return &ast.Identifier{SourceSpan: span, Name: name}
}

func number(value float64) *ast.NumberLiteral {
	span := helperSpan()
	return &ast.NumberLiteral{SourceSpan: span, Value: value}
}

func operator(symbol string) *ast.OperatorLiteral {
	span := helperSpan()
	return &ast.OperatorLiteral{SourceSpan: span, Symbol: symbol}
}

func binary(left ast.Expr, operator string, right ast.Expr) *ast.BinaryExpr {
	span := helperSpan()
	var op ast.BinaryOperator
	switch operator {
	case "+":
		op = ast.BinaryAdd
	case "-":
		op = ast.BinarySubtract
	case "*":
		op = ast.BinaryMultiply
	default:
		panic("unsupported helper operator")
	}

	return &ast.BinaryExpr{SourceSpan: span, Left: left, Operator: op, Right: right}
}

func group(inner ast.Expr) *ast.GroupExpr {
	span := helperSpan()
	return &ast.GroupExpr{SourceSpan: span, Inner: inner}
}

func rule(pattern, replacement ast.Expr) *ast.MutationRule {
	span := helperSpan()
	return &ast.MutationRule{SourceSpan: span, Pattern: pattern, Replacement: replacement}
}

func helperSpan() source.Span {
	file := source.NewFile("rewrite.molt", "dummy")
	return file.MustSpan(0, 5)
}

func expectExpr[T any](t *testing.T, expr ast.Expr) T {
	t.Helper()

	cast, ok := any(expr).(T)
	if !ok {
		t.Fatalf("expr type = %T, want %T", expr, cast)
	}

	return cast
}
