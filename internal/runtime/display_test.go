package runtime

import (
	"testing"

	"molt/internal/ast"
)

func TestShowValueFormatsPrimitiveAndCompactListValues(t *testing.T) {
	list := &ListValue{
		Elements: []Value{
			&NumberValue{Value: 1},
			&StringValue{Value: "ok"},
			&BooleanValue{Value: true},
			Nil,
		},
	}

	if got := ShowValue(&NumberValue{Value: 3.5}); got != "3.5" {
		t.Fatalf("number = %q, want %q", got, "3.5")
	}

	if got := ShowValue(&StringValue{Value: "hello\nworld"}); got != "\"hello\\nworld\"" {
		t.Fatalf("string = %q, want %q", got, "\"hello\\nworld\"")
	}

	if got := ShowValue(list); got != `[1, "ok", true, nil]` {
		t.Fatalf("list = %q, want %q", got, `[1, "ok", true, nil]`)
	}

	record := NewRecordValue([]RecordField{
		{Name: "name", Value: &StringValue{Value: "molt"}},
		{Name: "ok", Value: &BooleanValue{Value: true}},
	})

	if got := ShowValue(record); got != `record { name: "molt", ok: true }` {
		t.Fatalf("record = %q, want %q", got, `record { name: "molt", ok: true }`)
	}
}

func TestShowValueFormatsFunctionsCodeMutationsAndNativeFunctions(t *testing.T) {
	function := &UserFunctionValue{
		Name:       "add",
		Parameters: []string{"a", "b"},
		Body: &ast.BinaryExpr{
			Left:     &ast.Identifier{Name: "a"},
			Operator: ast.BinaryAdd,
			Right:    &ast.Identifier{Name: "b"},
		},
	}

	code := &CodeValue{
		Body: &ast.BinaryExpr{
			Left:     &ast.NumberLiteral{Value: 2},
			Operator: ast.BinaryAdd,
			Right:    &ast.NumberLiteral{Value: 3},
		},
	}

	mutation := &MutationValue{
		Rules: []*ast.MutationRule{
			{
				Pattern:     &ast.Identifier{Name: "x"},
				Replacement: &ast.Identifier{Name: "y"},
			},
			{
				Pattern:     &ast.NumberLiteral{Value: 1},
				Replacement: &ast.NumberLiteral{Value: 2},
			},
		},
	}

	if got := ShowValue(function); got != "fn add(a, b) = (a + b)" {
		t.Fatalf("function = %q, want %q", got, "fn add(a, b) = (a + b)")
	}

	if got := ShowValue(code); got != "@{ (2 + 3) }" {
		t.Fatalf("code = %q, want %q", got, "@{ (2 + 3) }")
	}

	wantMutation := "~{\n  x -> y\n  1 -> 2\n}"
	if got := ShowValue(mutation); got != wantMutation {
		t.Fatalf("mutation = %q, want %q", got, wantMutation)
	}

	if got := ShowValue(&NativeFunctionValue{FunctionName: "eval"}); got != "<native fn>" {
		t.Fatalf("native function = %q, want %q", got, "<native fn>")
	}
}

func TestShowValueFormatsMultilineNestedStructures(t *testing.T) {
	value := &ListValue{
		Elements: []Value{
			&NumberValue{Value: 1},
			&CodeValue{
				Body: &ast.BlockExpr{
					Expressions: []ast.Expr{
						&ast.Identifier{Name: "x"},
						&ast.Identifier{Name: "y"},
					},
				},
			},
		},
	}

	want := "[\n  1,\n  @{\n    x\n    y\n  }\n]"
	if got := ShowValue(value); got != want {
		t.Fatalf("multiline list = %q, want %q", got, want)
	}
}

func TestShowValueFormatsMultilineRecords(t *testing.T) {
	value := NewRecordValue([]RecordField{
		{Name: "name", Value: &StringValue{Value: "molt"}},
		{Name: "data", Value: &ListValue{
			Elements: []Value{
				&NumberValue{Value: 1},
				&CodeValue{
					Body: &ast.BlockExpr{
						Expressions: []ast.Expr{
							&ast.Identifier{Name: "x"},
							&ast.Identifier{Name: "y"},
						},
					},
				},
			},
		}},
	})

	want := "record {\n  name: \"molt\",\n  data: [\n    1,\n    @{\n      x\n      y\n    }\n  ]\n}"
	if got := ShowValue(value); got != want {
		t.Fatalf("multiline record = %q, want %q", got, want)
	}
}

func TestShowValueFormatsCodeContainingFieldAccess(t *testing.T) {
	code := &CodeValue{
		Body: &ast.FieldAccessExpr{
			Target: &ast.Identifier{Name: "profile"},
			Name:   &ast.Identifier{Name: "name"},
		},
	}

	if got := ShowValue(code); got != "@{ profile.name }" {
		t.Fatalf("code = %q, want %q", got, "@{ profile.name }")
	}
}

func TestShowValueFormatsCodeContainingWhileLoop(t *testing.T) {
	code := &CodeValue{
		Body: &ast.WhileExpr{
			Condition: &ast.Identifier{Name: "keepGoing"},
			Body: &ast.AssignmentExpr{
				Target: &ast.Identifier{Name: "x"},
				Value: &ast.BinaryExpr{
					Left:     &ast.Identifier{Name: "x"},
					Operator: ast.BinaryAdd,
					Right:    &ast.NumberLiteral{Value: 1},
				},
			},
		},
	}

	if got := ShowValue(code); got != "@{ while keepGoing -> x = (x + 1) }" {
		t.Fatalf("code = %q, want %q", got, "@{ while keepGoing -> x = (x + 1) }")
	}
}

func TestShowValueFormatsCodeContainingForInLoop(t *testing.T) {
	code := &CodeValue{
		Body: &ast.ForInExpr{
			Binding:  &ast.Identifier{Name: "item"},
			Iterable: &ast.Identifier{Name: "items"},
			Body: &ast.AssignmentExpr{
				Target: &ast.Identifier{Name: "total"},
				Value: &ast.BinaryExpr{
					Left:     &ast.Identifier{Name: "total"},
					Operator: ast.BinaryAdd,
					Right:    &ast.Identifier{Name: "item"},
				},
			},
		},
	}

	if got := ShowValue(code); got != "@{ for item in items -> total = (total + item) }" {
		t.Fatalf("code = %q, want %q", got, "@{ for item in items -> total = (total + item) }")
	}
}
