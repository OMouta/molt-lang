package evaluator

import (
	"bytes"
	"errors"
	"testing"

	"molt/internal/ast"
	"molt/internal/diagnostic"
	"molt/internal/parser"
	"molt/internal/runtime"
)

func TestEvaluateLiteralsListsAndIdentifierLookups(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	env.Define("x", &runtime.NumberValue{Value: 41})

	got := mustEval(t, env, "literals.molt", "x + 1")
	number := expectValue[*runtime.NumberValue](t, got)
	if number.Value != 42 {
		t.Fatalf("result = %v, want 42", number.Value)
	}

	listValue := mustEval(t, env, "list.molt", "[1, x, \"ok\", true, nil]")
	list := expectValue[*runtime.ListValue](t, listValue)
	if len(list.Elements) != 5 {
		t.Fatalf("list length = %d, want 5", len(list.Elements))
	}

	second := expectValue[*runtime.NumberValue](t, list.Elements[1])
	if second.Value != 41 {
		t.Fatalf("second element = %v, want 41", second.Value)
	}
}

func TestEvaluateBlocksAndAssignmentSemantics(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	result := mustEval(t, env, "blocks.molt", ""+
		"x = 1\n"+
		"{\n"+
		"  x = x + 1\n"+
		"  y = 10\n"+
		"  x\n"+
		"}\n"+
		"x",
	)

	number := expectValue[*runtime.NumberValue](t, result)
	if number.Value != 2 {
		t.Fatalf("result = %v, want 2", number.Value)
	}

	if _, ok := env.Get("y"); ok {
		t.Fatalf("block-local binding leaked into outer scope")
	}
}

func TestEvaluateIndexing(t *testing.T) {
	result := mustEval(t, runtime.NewEnvironment(nil), "index.molt", ""+
		"xs = [10, 20, 30]\n"+
		"xs[1]",
	)

	number := expectValue[*runtime.NumberValue](t, result)
	if number.Value != 20 {
		t.Fatalf("result = %v, want 20", number.Value)
	}
}

func TestEvaluateOperatorsConditionalsAndShortCircuiting(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	result := mustEval(t, env, "operators.molt", ""+
		"x = 0\n"+
		"false and (x = 1)\n"+
		"true or (x = 2)\n"+
		"if (1 + 2 * 3 == 7) and not false -> x else -> 99",
	)

	number := expectValue[*runtime.NumberValue](t, result)
	if number.Value != 0 {
		t.Fatalf("result = %v, want 0", number.Value)
	}

	current := expectValue[*runtime.NumberValue](t, env.MustGet("x"))
	if current.Value != 0 {
		t.Fatalf("short-circuiting failed, x = %v, want 0", current.Value)
	}
}

func TestEvaluateFunctionDefinitionsClosuresAndCalls(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	env.Define("doubleNative", &runtime.NativeFunctionValue{
		FunctionName: "doubleNative",
		Arity:        1,
		Impl: func(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
			value := expectValue[*runtime.NumberValue](t, args[0])
			return &runtime.NumberValue{Value: value.Value * 2}, nil
		},
	})

	result := mustEval(t, env, "functions.molt", ""+
		"x = 10\n"+
		"adder = fn(y) = x + y\n"+
		"fn choose(flag, n) = {\n"+
		"  if flag -> adder(n)\n"+
		"  else -> doubleNative(n)\n"+
		"}\n"+
		"x = 20\n"+
		"choose(true, 2)",
	)

	number := expectValue[*runtime.NumberValue](t, result)
	if number.Value != 22 {
		t.Fatalf("result = %v, want 22", number.Value)
	}

	bound := expectValue[*runtime.UserFunctionValue](t, env.MustGet("choose"))
	if bound.Name != "choose" {
		t.Fatalf("named function binding = %q, want %q", bound.Name, "choose")
	}
}

func TestEvaluateEmptyBlockReturnsNil(t *testing.T) {
	result := mustEval(t, runtime.NewEnvironment(nil), "empty_block.molt", "{\n}")
	if _, ok := result.(runtime.NilValue); !ok {
		t.Fatalf("result type = %T, want runtime.NilValue", result)
	}
}

func TestEvaluateRuntimeErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		message string
	}{
		{name: "invalid call target", input: "1(2)", message: `cannot call value of type "number"`},
		{name: "operator mismatch", input: "\"x\" + 1", message: `operator "+" requires number operands, got "string"`},
		{name: "index out of bounds", input: "xs = [1]\nxs[1]", message: "list index 1 out of bounds"},
		{name: "invalid index type", input: "xs = [1]\nxs[true]", message: `list index must be a number, got "boolean"`},
		{name: "fractional index", input: "xs = [1]\nxs[1.5]", message: "list index must be a non-negative integer, got 1.5"},
		{name: "undefined identifier", input: "missing", message: `undefined identifier "missing"`},
		{name: "condition type", input: "if 1 -> 2 else -> 3", message: `if condition must be boolean, got "number"`},
		{name: "user arity", input: "f = fn(x) = x\nf()", message: "expected 1 arguments but got 0"},
		{name: "invalid eval target", input: "eval(10)", message: `eval expects code value, got "number"`},
		{name: "invalid mutation rule", input: "~{ + -> 1 }", message: `invalid mutation rule 1: operator replacement rules must replace one operator with another`},
		{name: "invalid mutation operand", input: "code = @{ 1 }\ncode ~ 1", message: `expected mutation value, got "number"`},
		{name: "invalid len target", input: "len(1)", message: `len expects list or string, got "number"`},
		{name: "invalid push target", input: "push(1, 2)", message: `push expects list as first argument, got "number"`},
		{name: "invalid read_file target", input: "read_file(1)", message: `read_file expects string path, got "number"`},
		{name: "empty read_file path", input: `read_file("")`, message: "read_file path cannot be empty"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := evalStringWithEvaluator(nil, runtime.NewEnvironment(nil), tc.name+".molt", tc.input)
			runtimeErr := expectRuntimeError(t, err)
			if runtimeErr.Diagnostic().Message != tc.message {
				t.Fatalf("message = %q, want %q", runtimeErr.Diagnostic().Message, tc.message)
			}
		})
	}
}

func TestEvaluateStdinReadFailure(t *testing.T) {
	evaluator := NewWithIO(errReader{err: errors.New("boom")}, nil)

	_, err := evalStringWithEvaluator(evaluator, runtime.NewEnvironment(nil), "stdin_failure.molt", "stdin()")
	runtimeErr := expectRuntimeError(t, err)
	if runtimeErr.Diagnostic().Message != "stdin failed: boom" {
		t.Fatalf("message = %q, want %q", runtimeErr.Diagnostic().Message, "stdin failed: boom")
	}
}

func TestEvaluateReadFileFailure(t *testing.T) {
	evaluator := NewWithRuntime(nil, nil, nil, func(path string) ([]byte, error) {
		return nil, errors.New("boom")
	})

	_, err := evalStringWithEvaluator(evaluator, runtime.NewEnvironment(nil), "read_file_failure.molt", `read_file("missing.txt")`)
	runtimeErr := expectRuntimeError(t, err)
	if runtimeErr.Diagnostic().Message != `read_file failed for "missing.txt": boom` {
		t.Fatalf("message = %q, want %q", runtimeErr.Diagnostic().Message, `read_file failed for "missing.txt": boom`)
	}
}

func TestEvaluateNativeFunctionArityErrors(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	env.Define("one", &runtime.NativeFunctionValue{
		FunctionName: "one",
		Arity:        1,
		Impl: func(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
			return runtime.Nil, nil
		},
	})

	_, err := evalString(env, "native_arity.molt", "one()")
	runtimeErr := expectRuntimeError(t, err)
	if runtimeErr.Diagnostic().Message != "expected 1 arguments but got 0" {
		t.Fatalf("message = %q, want %q", runtimeErr.Diagnostic().Message, "expected 1 arguments but got 0")
	}
}

func TestEvaluateIntegrationProgram(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	result := mustEval(t, env, "integration.molt", ""+
		"base = 3\n"+
		"fn step(n) = {\n"+
		"  base = base + n\n"+
		"  if base > 10 -> base else -> step(base)\n"+
		"}\n"+
		"step(2)",
	)

	number := expectValue[*runtime.NumberValue](t, result)
	if number.Value != 20 {
		t.Fatalf("result = %v, want 20", number.Value)
	}

	base := expectValue[*runtime.NumberValue](t, env.MustGet("base"))
	if base.Value != 20 {
		t.Fatalf("base = %v, want 20", base.Value)
	}
}

func TestEvaluateQuoteCreatesCodeValueWithoutRunningIt(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	result := mustEval(t, env, "quote_create.molt", ""+
		"x = 0\n"+
		"code = @{ x = x + 1 }\n"+
		"code",
	)

	code := expectValue[*runtime.CodeValue](t, result)
	if code.TypeName() != "code" {
		t.Fatalf("code type = %q, want %q", code.TypeName(), "code")
	}

	if value := expectValue[*runtime.NumberValue](t, env.MustGet("x")); value.Value != 0 {
		t.Fatalf("quote executed eagerly, x = %v, want 0", value.Value)
	}
}

func TestEvaluateMutationLiteralCreatesMutationValueInOrder(t *testing.T) {
	result := mustEval(t, runtime.NewEnvironment(nil), "mutation_value.molt", "~{ x -> y\n1 -> 2\n+ -> * }")
	mutation := expectValue[*runtime.MutationValue](t, result)
	if mutation.TypeName() != "mutation" {
		t.Fatalf("mutation type = %q, want %q", mutation.TypeName(), "mutation")
	}

	if len(mutation.Rules) != 3 {
		t.Fatalf("rule count = %d, want 3", len(mutation.Rules))
	}

	first := expectMutationExpr[*ast.Identifier](t, mutation.Rules[0].Pattern)
	second := expectMutationExpr[*ast.NumberLiteral](t, mutation.Rules[1].Pattern)
	third := expectMutationExpr[*ast.OperatorLiteral](t, mutation.Rules[2].Pattern)
	if first.Name != "x" || second.Value != 1 || third.Symbol != "+" {
		t.Fatalf("mutation rules were not preserved in source order")
	}
}

func TestEvaluateCodeMutationPreservesCapturedEnvironmentAndOriginal(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	result := mustEval(t, env, "code_mutation.molt", ""+
		"x = 2\n"+
		"code = @{ x + 3 }\n"+
		"mut = ~{ + -> * }\n"+
		"mutated = code ~ mut\n"+
		"x = 4\n"+
		"[eval(code), eval(mutated), eval(code)]",
	)

	values := expectValue[*runtime.ListValue](t, result)
	if len(values.Elements) != 3 {
		t.Fatalf("result length = %d, want 3", len(values.Elements))
	}

	first := expectValue[*runtime.NumberValue](t, values.Elements[0])
	second := expectValue[*runtime.NumberValue](t, values.Elements[1])
	third := expectValue[*runtime.NumberValue](t, values.Elements[2])
	if first.Value != 7 || second.Value != 12 || third.Value != 7 {
		t.Fatalf("results = [%v, %v, %v], want [7, 12, 7]", first.Value, second.Value, third.Value)
	}

	original := expectValue[*runtime.CodeValue](t, env.MustGet("code"))
	mutated := expectValue[*runtime.CodeValue](t, env.MustGet("mutated"))
	if original == mutated {
		t.Fatalf("mutation should return a new code value")
	}

	originalBody := expectMutationExpr[*ast.BinaryExpr](t, original.Body)
	mutatedBody := expectMutationExpr[*ast.BinaryExpr](t, mutated.Body)
	if originalBody.Operator != ast.BinaryAdd {
		t.Fatalf("original operator = %q, want %q", originalBody.Operator, ast.BinaryAdd)
	}

	if mutatedBody.Operator != ast.BinaryMultiply {
		t.Fatalf("mutated operator = %q, want %q", mutatedBody.Operator, ast.BinaryMultiply)
	}

	if original.Env != mutated.Env {
		t.Fatalf("mutated code did not preserve captured environment")
	}
}

func TestEvaluateFunctionMutationPreservesParametersEnvironmentAndOriginal(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	result := mustEval(t, env, "function_mutation.molt", ""+
		"x = 10\n"+
		"fn add(y) = x + y\n"+
		"mut = ~{ + -> * }\n"+
		"mul = add ~ mut\n"+
		"x = 20\n"+
		"[add(2), mul(2), add(2)]",
	)

	values := expectValue[*runtime.ListValue](t, result)
	if len(values.Elements) != 3 {
		t.Fatalf("result length = %d, want 3", len(values.Elements))
	}

	first := expectValue[*runtime.NumberValue](t, values.Elements[0])
	second := expectValue[*runtime.NumberValue](t, values.Elements[1])
	third := expectValue[*runtime.NumberValue](t, values.Elements[2])
	if first.Value != 22 || second.Value != 40 || third.Value != 22 {
		t.Fatalf("results = [%v, %v, %v], want [22, 40, 22]", first.Value, second.Value, third.Value)
	}

	original := expectValue[*runtime.UserFunctionValue](t, env.MustGet("add"))
	mutated := expectValue[*runtime.UserFunctionValue](t, env.MustGet("mul"))
	if original == mutated {
		t.Fatalf("mutation should return a new function value")
	}

	if len(original.Parameters) != 1 || original.Parameters[0] != "y" {
		t.Fatalf("original parameters = %v, want [y]", original.Parameters)
	}

	if len(mutated.Parameters) != 1 || mutated.Parameters[0] != "y" {
		t.Fatalf("mutated parameters = %v, want [y]", mutated.Parameters)
	}

	originalBody := expectMutationExpr[*ast.BinaryExpr](t, original.Body)
	mutatedBody := expectMutationExpr[*ast.BinaryExpr](t, mutated.Body)
	if originalBody.Operator != ast.BinaryAdd {
		t.Fatalf("original operator = %q, want %q", originalBody.Operator, ast.BinaryAdd)
	}

	if mutatedBody.Operator != ast.BinaryMultiply {
		t.Fatalf("mutated operator = %q, want %q", mutatedBody.Operator, ast.BinaryMultiply)
	}

	if original.Env != mutated.Env {
		t.Fatalf("mutated function did not preserve closure environment")
	}
}

func TestEvaluateMutationCompositionConcatenatesRulesWithoutChangingInputs(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	result := mustEval(t, env, "mutation_composition.molt", ""+
		"m1 = ~{ 1 -> 2 }\n"+
		"m2 = ~{ 2 -> 3 }\n"+
		"m3 = m1 ~ m2\n"+
		"code = @{ 1 }\n"+
		"eval(code ~ m3)",
	)

	number := expectValue[*runtime.NumberValue](t, result)
	if number.Value != 3 {
		t.Fatalf("result = %v, want 3", number.Value)
	}

	m1 := expectValue[*runtime.MutationValue](t, env.MustGet("m1"))
	m2 := expectValue[*runtime.MutationValue](t, env.MustGet("m2"))
	m3 := expectValue[*runtime.MutationValue](t, env.MustGet("m3"))
	if len(m1.Rules) != 1 || len(m2.Rules) != 1 || len(m3.Rules) != 2 {
		t.Fatalf("rule counts = [%d, %d, %d], want [1, 1, 2]", len(m1.Rules), len(m2.Rules), len(m3.Rules))
	}

	first := expectMutationExpr[*ast.NumberLiteral](t, m3.Rules[0].Pattern)
	second := expectMutationExpr[*ast.NumberLiteral](t, m3.Rules[1].Pattern)
	if first.Value != 1 || second.Value != 2 {
		t.Fatalf("composed rule order = [%v, %v], want [1, 2]", first.Value, second.Value)
	}
}

func TestEvalUsesCapturedEnvironmentByReference(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	result := mustEval(t, env, "captured_reference.molt", ""+
		"x = 10\n"+
		"code = @{ x + 1 }\n"+
		"x = 20\n"+
		"eval(code)",
	)

	number := expectValue[*runtime.NumberValue](t, result)
	if number.Value != 21 {
		t.Fatalf("result = %v, want 21", number.Value)
	}
}

func TestEvalIgnoresCallerLocalScopeOutsideCapturedChain(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	result := mustEval(t, env, "caller_isolation.molt", ""+
		"x = 10\n"+
		"code = @{ x + 1 }\n"+
		"fn run(x) = eval(code)\n"+
		"run(20)",
	)

	number := expectValue[*runtime.NumberValue](t, result)
	if number.Value != 11 {
		t.Fatalf("result = %v, want 11", number.Value)
	}
}

func TestEvalReexecutesFreshlyOnEachCall(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	result := mustEval(t, env, "repeated_eval.molt", ""+
		"x = 0\n"+
		"code = @{ x = x + 1 }\n"+
		"eval(code)\n"+
		"eval(code)\n"+
		"x",
	)

	number := expectValue[*runtime.NumberValue](t, result)
	if number.Value != 2 {
		t.Fatalf("result = %v, want 2", number.Value)
	}
}

func TestEvaluateBuiltins(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	result, err := evalStringWithEvaluator(NewWithRuntime(bytes.NewBufferString("hello\nworld"), nil, []string{"alpha", "beta"}, func(path string) ([]byte, error) {
		if path != "note.txt" {
			return nil, errors.New("unexpected path")
		}

		return []byte("file contents"), nil
	}), env, "builtins.molt", ""+
		"xs = [1]\n"+
		"same = push(xs, 2)\n"+
		"fn add(a, b) = a + b\n"+
		"code = @{ 1 + 2 }\n"+
		"mut = ~{ x -> y\n1 -> 2 }\n"+
		"cli1 = args()\n"+
		"cli2 = args()\n"+
		"push(cli1, \"extra\")\n"+
		"file1 = read_file(\"note.txt\")\n"+
		"file2 = read_file(\"note.txt\")\n"+
		"input1 = stdin()\n"+
		"input2 = stdin()\n"+
		"[\n"+
		"  type(1),\n"+
		"  type(\"x\"),\n"+
		"  type(true),\n"+
		"  type(nil),\n"+
		"  type(xs),\n"+
		"  type(add),\n"+
		"  type(eval),\n"+
		"  type(code),\n"+
		"  type(mut),\n"+
		"  len(xs),\n"+
		"  len(\"aé\"),\n"+
		"  same == xs,\n"+
		"  show(xs),\n"+
		"  show(code),\n"+
		"  show(mut),\n"+
		"  show(add),\n"+
		"  cli1,\n"+
		"  cli2,\n"+
		"  file1,\n"+
		"  file2,\n"+
		"  input1,\n"+
		"  input2\n"+
		"]",
	)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	values := expectValue[*runtime.ListValue](t, result)
	if len(values.Elements) != 22 {
		t.Fatalf("result length = %d, want 22", len(values.Elements))
	}

	wantTypes := []string{
		"number",
		"string",
		"boolean",
		"nil",
		"list",
		"function",
		"native-function",
		"code",
		"mutation",
	}

	for i, want := range wantTypes {
		got := expectValue[*runtime.StringValue](t, values.Elements[i])
		if got.Value != want {
			t.Fatalf("type result %d = %q, want %q", i, got.Value, want)
		}
	}

	if got := expectValue[*runtime.NumberValue](t, values.Elements[9]); got.Value != 2 {
		t.Fatalf("len(xs) = %v, want 2", got.Value)
	}

	if got := expectValue[*runtime.NumberValue](t, values.Elements[10]); got.Value != 2 {
		t.Fatalf("len(\"aé\") = %v, want 2", got.Value)
	}

	if got := expectValue[*runtime.BooleanValue](t, values.Elements[11]); !got.Value {
		t.Fatalf("push did not return the mutated list")
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[12]); got.Value != `[1, 2]` {
		t.Fatalf("show(xs) = %q, want %q", got.Value, `[1, 2]`)
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[13]); got.Value != "@{ (1 + 2) }" {
		t.Fatalf("show(code) = %q, want %q", got.Value, "@{ (1 + 2) }")
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[14]); got.Value != "~{\n  x -> y\n  1 -> 2\n}" {
		t.Fatalf("show(mut) = %q, want multiline mutation", got.Value)
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[15]); got.Value != "fn add(a, b) = (a + b)" {
		t.Fatalf("show(add) = %q, want %q", got.Value, "fn add(a, b) = (a + b)")
	}

	if got := expectValue[*runtime.ListValue](t, values.Elements[16]); runtime.ShowValue(got) != `["alpha", "beta", "extra"]` {
		t.Fatalf("mutated cli1 = %q, want %q", runtime.ShowValue(got), `["alpha", "beta", "extra"]`)
	}

	if got := expectValue[*runtime.ListValue](t, values.Elements[17]); runtime.ShowValue(got) != `["alpha", "beta"]` {
		t.Fatalf("fresh cli2 = %q, want %q", runtime.ShowValue(got), `["alpha", "beta"]`)
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[18]); got.Value != "file contents" {
		t.Fatalf("read_file first = %q, want %q", got.Value, "file contents")
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[19]); got.Value != "file contents" {
		t.Fatalf("read_file second = %q, want %q", got.Value, "file contents")
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[20]); got.Value != "hello\nworld" {
		t.Fatalf("stdin() first read = %q, want %q", got.Value, "hello\nworld")
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[21]); got.Value != "" {
		t.Fatalf("stdin() second read = %q, want empty string", got.Value)
	}
}

func TestEvaluatePrintWritesDisplayOutputAndReturnsNil(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	var output bytes.Buffer
	evaluator := &Evaluator{output: &output}

	value, err := evalStringWithEvaluator(evaluator, env, "print.molt", ""+
		"xs = [1, 2]\n"+
		"print(xs)",
	)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	if _, ok := value.(runtime.NilValue); !ok {
		t.Fatalf("print result type = %T, want runtime.NilValue", value)
	}

	if output.String() != "[1, 2]\n" {
		t.Fatalf("print output = %q, want %q", output.String(), "[1, 2]\n")
	}
}

func TestEvaluateMutationApplicationRejectsInvalidTargets(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		env     *runtime.Environment
		message string
	}{
		{name: "number", input: "1 ~ ~{ 1 -> 2 }", env: runtime.NewEnvironment(nil), message: `cannot apply mutation to value of type "number"`},
		{name: "string", input: "\"x\" ~ ~{ \"x\" -> \"y\" }", env: runtime.NewEnvironment(nil), message: `cannot apply mutation to value of type "string"`},
		{name: "list", input: "[1] ~ ~{ 1 -> 2 }", env: runtime.NewEnvironment(nil), message: `cannot apply mutation to value of type "list"`},
		{
			name:  "native function",
			input: "native ~ ~{ 1 -> 2 }",
			env: func() *runtime.Environment {
				env := runtime.NewEnvironment(nil)
				env.Define("native", &runtime.NativeFunctionValue{
					FunctionName: "native",
					Arity:        0,
					Impl: func(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
						return runtime.Nil, nil
					},
				})
				return env
			}(),
			message: `cannot apply mutation to value of type "native-function"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := evalString(tc.env, tc.name+".molt", tc.input)
			runtimeErr := expectRuntimeError(t, err)
			if runtimeErr.Diagnostic().Message != tc.message {
				t.Fatalf("message = %q, want %q", runtimeErr.Diagnostic().Message, tc.message)
			}
		})
	}
}

func evalString(env *runtime.Environment, path, input string) (runtime.Value, error) {
	program, err := parser.Parse(path, input)
	if err != nil {
		return nil, err
	}

	return EvalProgram(program, env)
}

func evalStringWithEvaluator(evaluator *Evaluator, env *runtime.Environment, path, input string) (runtime.Value, error) {
	program, err := parser.Parse(path, input)
	if err != nil {
		return nil, err
	}

	if evaluator == nil {
		evaluator = &Evaluator{}
	}

	return evaluator.EvalProgram(program, env)
}

func mustEval(t *testing.T, env *runtime.Environment, path, input string) runtime.Value {
	t.Helper()

	value, err := evalString(env, path, input)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	return value
}

func expectValue[T any](t *testing.T, value runtime.Value) T {
	t.Helper()

	cast, ok := any(value).(T)
	if !ok {
		t.Fatalf("value type = %T, want %T", value, cast)
	}

	return cast
}

func expectRuntimeError(t *testing.T, err error) diagnostic.RuntimeError {
	t.Helper()

	if err == nil {
		t.Fatalf("expected runtime error, got nil")
	}

	runtimeErr, ok := err.(diagnostic.RuntimeError)
	if !ok {
		t.Fatalf("expected diagnostic.RuntimeError, got %T (%v)", err, err)
	}

	return runtimeErr
}

func expectMutationExpr[T any](t *testing.T, expr ast.Expr) T {
	t.Helper()

	cast, ok := any(expr).(T)
	if !ok {
		t.Fatalf("mutation expr type = %T, want %T", expr, cast)
	}

	return cast
}

type errReader struct {
	err error
}

func (r errReader) Read(p []byte) (int, error) {
	return 0, r.err
}
