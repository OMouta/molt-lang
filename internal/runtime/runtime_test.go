package runtime

import (
	"errors"
	"testing"

	"molt/internal/ast"
	"molt/internal/source"
)

func TestValueTypesCoverEveryRuntimeValueKind(t *testing.T) {
	body := sampleIdentifierExpr()
	env := NewEnvironment(nil)
	native := &NativeFunctionValue{
		FunctionName: "len",
		Arity:        1,
		Impl: func(ctx *CallContext, args []Value) (Value, error) {
			return &NumberValue{Value: float64(len(args))}, nil
		},
	}

	values := []Value{
		&NumberValue{Value: 1},
		&StringValue{Value: "hello"},
		&BooleanValue{Value: true},
		Nil,
		&ListValue{Elements: []Value{&NumberValue{Value: 1}}},
		NewRecordValue([]RecordField{{Name: "answer", Value: &NumberValue{Value: 42}}}),
		&UserFunctionValue{Name: "double", Parameters: []string{"x"}, Body: body, Env: env},
		native,
		&CodeValue{Body: body, Env: env},
		&MutationValue{Rules: []*ast.MutationRule{
			{
				SourceSpan:  body.Span(),
				Pattern:     &ast.Identifier{SourceSpan: body.Span(), Name: "x"},
				Replacement: &ast.Identifier{SourceSpan: body.Span(), Name: "y"},
			},
		}},
	}

	want := []string{
		"number",
		"string",
		"boolean",
		"nil",
		"list",
		"record",
		"function",
		"native-function",
		"code",
		"mutation",
	}

	if len(values) != len(want) {
		t.Fatalf("value count = %d, want %d", len(values), len(want))
	}

	for i, value := range values {
		if got := value.TypeName(); got != want[i] {
			t.Fatalf("values[%d].TypeName() = %q, want %q", i, got, want[i])
		}
	}

	result, err := native.Call(&CallContext{FunctionName: "len", Environment: env}, []Value{
		&StringValue{Value: "a"},
		&StringValue{Value: "b"},
	})
	if err != nil {
		t.Fatalf("native call returned error: %v", err)
	}

	count := expectValue[*NumberValue](t, result)
	if count.Value != 2 {
		t.Fatalf("native call result = %v, want 2", count.Value)
	}
}

func TestRecordValuePreservesFieldOrderAndLookup(t *testing.T) {
	record := NewRecordValue([]RecordField{
		{Name: "first", Value: &NumberValue{Value: 1}},
		{Name: "second", Value: &NumberValue{Value: 2}},
	})

	if len(record.Fields) != 2 {
		t.Fatalf("field count = %d, want 2", len(record.Fields))
	}

	if record.Fields[0].Name != "first" || record.Fields[1].Name != "second" {
		t.Fatalf("record field order was not preserved")
	}

	value, ok := record.GetField("second")
	if !ok {
		t.Fatalf("expected second field lookup to succeed")
	}

	number := expectValue[*NumberValue](t, value)
	if number.Value != 2 {
		t.Fatalf("lookup value = %v, want 2", number.Value)
	}

	if _, ok := record.GetField("missing"); ok {
		t.Fatalf("missing field lookup should fail")
	}

	if got := record.Keys(); len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("record keys = %v, want [first second]", got)
	}

	values := record.Values()
	if len(values) != 2 {
		t.Fatalf("record values length = %d, want 2", len(values))
	}

	first := expectValue[*NumberValue](t, values[0])
	secondValue := expectValue[*NumberValue](t, values[1])
	if first.Value != 1 || secondValue.Value != 2 {
		t.Fatalf("record values = [%v, %v], want [1, 2]", first.Value, secondValue.Value)
	}
}

func TestEnvironmentLookupFindsBindingsAcrossOuterScopes(t *testing.T) {
	global := NewEnvironment(nil)
	global.Define("x", &NumberValue{Value: 1})

	middle := NewEnvironment(global)
	middle.Define("y", &StringValue{Value: "middle"})

	inner := NewEnvironment(middle)
	inner.Define("z", &BooleanValue{Value: true})

	if got := expectValue[*NumberValue](t, inner.MustGet("x")); got.Value != 1 {
		t.Fatalf("lookup x = %v, want 1", got.Value)
	}

	if got := expectValue[*StringValue](t, inner.MustGet("y")); got.Value != "middle" {
		t.Fatalf("lookup y = %q, want %q", got.Value, "middle")
	}

	if got := expectValue[*BooleanValue](t, inner.MustGet("z")); !got.Value {
		t.Fatalf("lookup z = false, want true")
	}
}

func TestAssignUpdatesOuterBindingOtherwiseCreatesLocalBinding(t *testing.T) {
	global := NewEnvironment(nil)
	global.Define("shared", &NumberValue{Value: 1})

	inner := NewEnvironment(global)
	inner.Assign("shared", &NumberValue{Value: 2})

	updated := expectValue[*NumberValue](t, global.MustGet("shared"))
	if updated.Value != 2 {
		t.Fatalf("outer binding value = %v, want 2", updated.Value)
	}

	if inner.HasLocal("shared") {
		t.Fatalf("assigning an outer binding should not create a local shadow")
	}

	inner.Assign("local", &StringValue{Value: "created"})
	created := expectValue[*StringValue](t, inner.MustGet("local"))
	if created.Value != "created" {
		t.Fatalf("local binding value = %q, want %q", created.Value, "created")
	}

	if _, ok := global.Get("local"); ok {
		t.Fatalf("new binding should be created in current scope, not outer scope")
	}
}

func TestNestedScopesCanShadowWithoutTouchingOuterBindings(t *testing.T) {
	global := NewEnvironment(nil)
	global.Define("name", &StringValue{Value: "outer"})

	inner := NewEnvironment(global)
	inner.Define("name", &StringValue{Value: "inner"})

	outer := expectValue[*StringValue](t, global.MustGet("name"))
	if outer.Value != "outer" {
		t.Fatalf("outer name = %q, want %q", outer.Value, "outer")
	}

	shadow := expectValue[*StringValue](t, inner.MustGet("name"))
	if shadow.Value != "inner" {
		t.Fatalf("inner name = %q, want %q", shadow.Value, "inner")
	}
}

func TestUserFunctionClosuresCaptureEnvironmentByReference(t *testing.T) {
	global := NewEnvironment(nil)
	global.Define("factor", &NumberValue{Value: 2})

	function := &UserFunctionValue{
		Name:       "scale",
		Parameters: []string{"x"},
		Body:       sampleIdentifierExpr(),
		Env:        global,
	}

	captured := expectValue[*NumberValue](t, function.Env.MustGet("factor"))
	if captured.Value != 2 {
		t.Fatalf("captured factor = %v, want 2", captured.Value)
	}

	global.Assign("factor", &NumberValue{Value: 3})
	updated := expectValue[*NumberValue](t, function.Env.MustGet("factor"))
	if updated.Value != 3 {
		t.Fatalf("updated captured factor = %v, want 3", updated.Value)
	}

	callEnv := NewEnvironment(function.Env)
	callEnv.Define("x", &NumberValue{Value: 10})

	if callEnv.Parent() != function.Env {
		t.Fatalf("call environment parent was not preserved")
	}

	param := expectValue[*NumberValue](t, callEnv.MustGet("x"))
	if param.Value != 10 {
		t.Fatalf("parameter binding = %v, want 10", param.Value)
	}
}

func TestNativeFunctionRequiresImplementation(t *testing.T) {
	native := &NativeFunctionValue{FunctionName: "broken"}
	_, err := native.Call(&CallContext{FunctionName: "broken"}, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, err) {
		t.Fatalf("expected a concrete error value")
	}
}

func sampleIdentifierExpr() ast.Expr {
	file := source.NewFile("runtime.molt", "x")
	span := file.MustSpan(0, 1)
	return &ast.Identifier{SourceSpan: span, Name: "x"}
}

func expectValue[T any](t *testing.T, value Value) T {
	t.Helper()

	cast, ok := any(value).(T)
	if !ok {
		t.Fatalf("value type = %T, want %T", value, cast)
	}

	return cast
}
