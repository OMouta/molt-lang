package evaluator

import (
	"bytes"
	"errors"
	"path/filepath"
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

	recordValue := mustEval(t, env, "record.molt", `record { answer: x + 1, nested: record { ok: true } }`)
	record := expectValue[*runtime.RecordValue](t, recordValue)
	if len(record.Fields) != 2 {
		t.Fatalf("record field count = %d, want 2", len(record.Fields))
	}

	answer := expectValue[*runtime.NumberValue](t, record.Fields[0].Value)
	if answer.Value != 42 {
		t.Fatalf("answer field = %v, want 42", answer.Value)
	}

	nested := expectValue[*runtime.RecordValue](t, record.Fields[1].Value)
	okValue, exists := nested.GetField("ok")
	if !exists {
		t.Fatalf("nested ok field lookup failed")
	}

	boolean := expectValue[*runtime.BooleanValue](t, okValue)
	if !boolean.Value {
		t.Fatalf("nested ok field = false, want true")
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
		"colors = record { space: \"\\n\" }\n"+
		"[xs[1], colors[\"space\"]]",
	)

	values := expectValue[*runtime.ListValue](t, result)
	if len(values.Elements) != 2 {
		t.Fatalf("result length = %d, want 2", len(values.Elements))
	}

	number := expectValue[*runtime.NumberValue](t, values.Elements[0])
	if number.Value != 20 {
		t.Fatalf("first value = %v, want 20", number.Value)
	}

	separator := expectValue[*runtime.StringValue](t, values.Elements[1])
	if separator.Value != "\n" {
		t.Fatalf("second value = %q, want %q", separator.Value, "\\n")
	}
}

func TestEvaluateRecordFieldAccess(t *testing.T) {
	result := mustEval(t, runtime.NewEnvironment(nil), "field_access.molt", ""+
		"profile = record { name: \"molt\", stats: record { runs: 3 } }\n"+
		"[profile.name, profile.stats.runs]",
	)

	values := expectValue[*runtime.ListValue](t, result)
	if len(values.Elements) != 2 {
		t.Fatalf("result length = %d, want 2", len(values.Elements))
	}

	name := expectValue[*runtime.StringValue](t, values.Elements[0])
	if name.Value != "molt" {
		t.Fatalf("name = %q, want %q", name.Value, "molt")
	}

	runs := expectValue[*runtime.NumberValue](t, values.Elements[1])
	if runs.Value != 3 {
		t.Fatalf("runs = %v, want 3", runs.Value)
	}
}

func TestEvaluateWhileLoopsReturnNilAndUseIterationLocalScope(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	result := mustEval(t, env, "while_loop.molt", ""+
		"x = 0\n"+
		"loop = while x < 3 -> {\n"+
		"  temp = x\n"+
		"  x = x + 1\n"+
		"}\n"+
		"[loop, x]",
	)

	values := expectValue[*runtime.ListValue](t, result)
	if len(values.Elements) != 2 {
		t.Fatalf("result length = %d, want 2", len(values.Elements))
	}

	if _, ok := values.Elements[0].(runtime.NilValue); !ok {
		t.Fatalf("while result type = %T, want runtime.NilValue", values.Elements[0])
	}

	finalX := expectValue[*runtime.NumberValue](t, values.Elements[1])
	if finalX.Value != 3 {
		t.Fatalf("x = %v, want 3", finalX.Value)
	}

	if _, ok := env.Get("temp"); ok {
		t.Fatalf("iteration-local binding leaked into outer scope")
	}
}

func TestEvaluateWhileLoopCanUseBareBodyExpression(t *testing.T) {
	result := mustEval(t, runtime.NewEnvironment(nil), "while_bare_body.molt", ""+
		"x = 0\n"+
		"while x < 2 -> x = x + 1\n"+
		"x",
	)

	number := expectValue[*runtime.NumberValue](t, result)
	if number.Value != 2 {
		t.Fatalf("result = %v, want 2", number.Value)
	}
}

func TestEvaluateForInLoopsReturnNilAndUseIterationLocalScope(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	result := mustEval(t, env, "for_in_loop.molt", ""+
		"total = 0\n"+
		"loop = for item in [1, 2, 3] -> {\n"+
		"  temp = item\n"+
		"  total = total + item\n"+
		"}\n"+
		"[loop, total]",
	)

	values := expectValue[*runtime.ListValue](t, result)
	if len(values.Elements) != 2 {
		t.Fatalf("result length = %d, want 2", len(values.Elements))
	}

	if _, ok := values.Elements[0].(runtime.NilValue); !ok {
		t.Fatalf("for result type = %T, want runtime.NilValue", values.Elements[0])
	}

	total := expectValue[*runtime.NumberValue](t, values.Elements[1])
	if total.Value != 6 {
		t.Fatalf("total = %v, want 6", total.Value)
	}

	if _, ok := env.Get("item"); ok {
		t.Fatalf("loop binding leaked into outer scope")
	}

	if _, ok := env.Get("temp"); ok {
		t.Fatalf("iteration-local binding leaked into outer scope")
	}
}

func TestEvaluateForInLoopIteratesStringsByUnicodeCodePoint(t *testing.T) {
	result := mustEval(t, runtime.NewEnvironment(nil), "for_in_string.molt", ""+
		"chars = []\n"+
		"for ch in \"aé\" -> push(chars, ch)\n"+
		"chars",
	)

	if got := runtime.ShowValue(result); got != `["a", "é"]` {
		t.Fatalf("string iteration = %q, want %q", got, `["a", "é"]`)
	}
}

func TestEvaluateLoopControlSupportsBreakContinueAndNestedLoops(t *testing.T) {
	result := mustEval(t, runtime.NewEnvironment(nil), "loop_control.molt", ""+
		"pairs = []\n"+
		"for outer in [1, 2, 3] -> {\n"+
		"  if outer == 2 -> continue else -> nil\n"+
		"  for inner in [10, 20, 30] -> {\n"+
		"    if inner == 30 -> break else -> nil\n"+
		"    push(pairs, outer + inner)\n"+
		"  }\n"+
		"}\n"+
		"pairs",
	)

	if got := runtime.ShowValue(result); got != `[11, 21, 13, 23]` {
		t.Fatalf("pairs = %q, want %q", got, `[11, 21, 13, 23]`)
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

func TestEvaluateConditionalWithoutElseReturnsNilWhenFalse(t *testing.T) {
	result := mustEval(t, runtime.NewEnvironment(nil), "if_without_else.molt", ""+
		"steps = []\n"+
		"hit = if true -> push(steps, 1)\n"+
		"miss = if false -> push(steps, 2)\n"+
		"[hit, miss, steps]",
	)

	if got := runtime.ShowValue(result); got != `[[1], nil, [1]]` {
		t.Fatalf("result = %q, want %q", got, `[[1], nil, [1]]`)
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

func TestEvaluateImportLoadsRelativeModuleBindings(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.molt")
	libPath := filepath.Join(dir, "lib.molt")

	files := map[string]string{
		libPath: "" +
			"answer = 41\n" +
			"fn bump(x) = x + 1\n" +
			"export answer\n" +
			"export bump",
	}

	evaluator := NewWithRuntime(nil, nil, nil, func(path string) ([]byte, error) {
		value, ok := files[path]
		if !ok {
			return nil, errors.New("missing file")
		}

		return []byte(value), nil
	}, nil)

	result, err := evalStringWithEvaluator(evaluator, runtime.NewEnvironment(nil), mainPath, ""+
		"import \"./lib.molt\"\n"+
		"bump(answer)",
	)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	number := expectValue[*runtime.NumberValue](t, result)
	if number.Value != 42 {
		t.Fatalf("result = %v, want 42", number.Value)
	}
}

func TestEvaluateImportCachesModulesWithinOneRun(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.molt")
	libPath := filepath.Join(dir, "lib.molt")

	files := map[string]string{
		libPath: "" +
			"xs = []\n" +
			"fn tick() = {\n" +
			"  push(xs, 1)\n" +
			"  len(xs)\n" +
			"}\n" +
			"export tick",
	}

	readCounts := make(map[string]int)
	evaluator := NewWithRuntime(nil, nil, nil, func(path string) ([]byte, error) {
		readCounts[path]++
		value, ok := files[path]
		if !ok {
			return nil, errors.New("missing file")
		}

		return []byte(value), nil
	}, nil)

	result, err := evalStringWithEvaluator(evaluator, runtime.NewEnvironment(nil), mainPath, ""+
		"import \"./lib.molt\"\n"+
		"a = tick()\n"+
		"import \"./lib.molt\"\n"+
		"b = tick()\n"+
		"[a, b]",
	)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	if got := runtime.ShowValue(result); got != "[1, 2]" {
		t.Fatalf("result = %q, want %q", got, "[1, 2]")
	}

	if got := readCounts[libPath]; got != 1 {
		t.Fatalf("read count = %d, want 1", got)
	}
}

func TestEvaluateImportUsesExportedFunctionsWithoutLeakingPrivateBindings(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.molt")
	libPath := filepath.Join(dir, "lib.molt")

	files := map[string]string{
		libPath: "" +
			"helper = 40\n" +
			"fn add2(x) = helper + x\n" +
			"export add2",
	}

	evaluator := NewWithRuntime(nil, nil, nil, func(path string) ([]byte, error) {
		value, ok := files[path]
		if !ok {
			return nil, errors.New("missing file")
		}

		return []byte(value), nil
	}, nil)

	result, err := evalStringWithEvaluator(evaluator, runtime.NewEnvironment(nil), mainPath, ""+
		"import \"./lib.molt\"\n"+
		"add2(2)",
	)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	number := expectValue[*runtime.NumberValue](t, result)
	if number.Value != 42 {
		t.Fatalf("result = %v, want 42", number.Value)
	}

	_, err = evalStringWithEvaluator(evaluator, runtime.NewEnvironment(nil), mainPath, ""+
		"import \"./lib.molt\"\n"+
		"helper",
	)
	runtimeErr := expectRuntimeError(t, err)
	if runtimeErr.Diagnostic().Message != `undefined identifier "helper"` {
		t.Fatalf("message = %q, want %q", runtimeErr.Diagnostic().Message, `undefined identifier "helper"`)
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
		{name: "record invalid index type", input: "r = record { x: 1 }\nr[1]", message: `record index must be a string, got "number"`},
		{name: "record missing field", input: `record { answer: 42 }["name"]`, message: `record has no field "name"`},
		{name: "field access invalid target", input: `(1).name`, message: `cannot access field "name" on value of type "number"`},
		{name: "field access missing field", input: `record { answer: 42 }.name`, message: `record has no field "name"`},
		{name: "undefined identifier", input: "missing", message: `undefined identifier "missing"`},
		{name: "condition type", input: "if 1 -> 2 else -> 3", message: `if condition must be boolean, got "number"`},
		{name: "while condition type", input: "while 1 -> 2", message: `while condition must be boolean, got "number"`},
		{name: "for iterable type", input: "for x in 1 -> x", message: `for loop expects list or string, got "number"`},
		{name: "top level break", input: "break", message: "break is only allowed inside loops"},
		{name: "top level continue", input: "continue", message: "continue is only allowed inside loops"},
		{name: "user arity", input: "f = fn(x) = x\nf()", message: "expected 1 arguments but got 0"},
		{name: "invalid eval target", input: "eval(10)", message: `eval expects code value, got "number"`},
		{name: "invalid mutation rule", input: "~{ + -> 1 }", message: `invalid mutation rule 1: operator replacement rules must replace one operator with another`},
		{name: "invalid mutation operand", input: "code = @{ 1 }\ncode ~ 1", message: `expected mutation value, got "number"`},
		{name: "invalid len target", input: "len(1)", message: `len expects list, string, or record, got "number"`},
		{name: "invalid push target", input: "push(1, 2)", message: `push expects list as first argument, got "number"`},
		{name: "invalid split target", input: `split(1, ",")`, message: `split expects string as first argument, got "number"`},
		{name: "invalid join element", input: `join([1], ",")`, message: `join expects list of strings, but element 0 has type "number"`},
		{name: "invalid trim target", input: `trim(1)`, message: `trim expects string, got "number"`},
		{name: "invalid lines target", input: `lines(1)`, message: `lines expects string, got "number"`},
		{name: "invalid replace text", input: `replace(1, "a", "b")`, message: `replace expects string as first argument, got "number"`},
		{name: "invalid replace old", input: `replace("abc", 1, "b")`, message: `replace expects string as second argument, got "number"`},
		{name: "invalid replace new", input: `replace("abc", "a", 1)`, message: `replace expects string as third argument, got "number"`},
		{name: "invalid contains text", input: `contains(1, "a")`, message: `contains expects string or record as first argument, got "number"`},
		{name: "invalid contains needle", input: `contains("abc", 1)`, message: `contains expects string as second argument, got "number"`},
		{name: "invalid contains record key", input: `contains(record { answer: 42 }, 1)`, message: `contains expects string key as second argument for records, got "number"`},
		{name: "invalid keys target", input: `keys(1)`, message: `keys expects record, got "number"`},
		{name: "invalid values target", input: `values(1)`, message: `values expects record, got "number"`},
		{name: "invalid range arity", input: `range(1, 2, 3)`, message: "range expects 1 or 2 arguments but got 3"},
		{name: "invalid range integer", input: `range(1.5)`, message: "range expects integer at argument 1, got 1.5"},
		{name: "invalid map callback", input: `map([1], 1)`, message: `map expects function as second argument, got "number"`},
		{name: "invalid map callback arity", input: `map([1], fn(a, b, c) = a)`, message: "map callback must accept 1 or 2 arguments, got 3"},
		{name: "invalid filter callback result", input: `filter([1], fn(x) = x)`, message: `filter callback must return boolean, got "number"`},
		{name: "invalid to_number parse", input: `to_number("abc")`, message: `to_number could not parse "abc"`},
		{name: "invalid read_file target", input: "read_file(1)", message: `read_file expects string path, got "number"`},
		{name: "empty read_file path", input: `read_file("")`, message: "read_file path cannot be empty"},
		{name: "invalid write_file path", input: `write_file(1, "x")`, message: `write_file expects string path as first argument, got "number"`},
		{name: "invalid write_file text", input: `write_file("out.txt", 1)`, message: `write_file expects string text as second argument, got "number"`},
		{name: "empty write_file path", input: `write_file("", "x")`, message: "write_file path cannot be empty"},
		{name: "empty import path", input: `import ""`, message: "import path cannot be empty"},
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

func TestEvaluateLoopControlDoesNotCrossFunctionOrEvalBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		message string
	}{
		{
			name: "function break",
			input: "" +
				"fn stop() = break\n" +
				"while true -> stop()",
			message: "break is only allowed inside loops",
		},
		{
			name: "eval continue",
			input: "" +
				"code = @{ continue }\n" +
				"while true -> eval(code)",
			message: "continue is only allowed inside loops",
		},
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

func TestEvaluateImportReadFailure(t *testing.T) {
	evaluator := NewWithRuntime(nil, nil, nil, func(path string) ([]byte, error) {
		return nil, errors.New("boom")
	}, nil)

	_, err := evalStringWithEvaluator(evaluator, runtime.NewEnvironment(nil), "main.molt", `import "./missing.molt"`)
	runtimeErr := expectRuntimeError(t, err)
	if runtimeErr.Diagnostic().Message != `import failed for "./missing.molt": boom` {
		t.Fatalf("message = %q, want %q", runtimeErr.Diagnostic().Message, `import failed for "./missing.molt": boom`)
	}
}

func TestEvaluateImportDirectCycleFailure(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.molt")
	loopPath := filepath.Join(dir, "loop.molt")

	files := map[string]string{
		loopPath: `import "./loop.molt"`,
	}

	evaluator := NewWithRuntime(nil, nil, nil, func(path string) ([]byte, error) {
		value, ok := files[path]
		if !ok {
			return nil, errors.New("missing file")
		}

		return []byte(value), nil
	}, nil)

	_, err := evalStringWithEvaluator(evaluator, runtime.NewEnvironment(nil), mainPath, `import "./loop.molt"`)
	runtimeErr := expectRuntimeError(t, err)
	want := "import cycle detected: " + filepath.ToSlash(loopPath) + " -> " + filepath.ToSlash(loopPath)
	if runtimeErr.Diagnostic().Message != want {
		t.Fatalf("message = %q, want %q", runtimeErr.Diagnostic().Message, want)
	}
}

func TestEvaluateImportIndirectCycleFailure(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.molt")
	aPath := filepath.Join(dir, "a.molt")
	bPath := filepath.Join(dir, "b.molt")

	files := map[string]string{
		aPath: `import "./b.molt"`,
		bPath: `import "./a.molt"`,
	}

	evaluator := NewWithRuntime(nil, nil, nil, func(path string) ([]byte, error) {
		value, ok := files[path]
		if !ok {
			return nil, errors.New("missing file")
		}

		return []byte(value), nil
	}, nil)

	_, err := evalStringWithEvaluator(evaluator, runtime.NewEnvironment(nil), mainPath, `import "./a.molt"`)
	runtimeErr := expectRuntimeError(t, err)
	want := "import cycle detected: " + filepath.ToSlash(aPath) + " -> " + filepath.ToSlash(bPath) + " -> " + filepath.ToSlash(aPath)
	if runtimeErr.Diagnostic().Message != want {
		t.Fatalf("message = %q, want %q", runtimeErr.Diagnostic().Message, want)
	}
}

func TestEvaluateImportUndefinedExportFailure(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.molt")
	libPath := filepath.Join(dir, "lib.molt")

	files := map[string]string{
		libPath: `export missing`,
	}

	evaluator := NewWithRuntime(nil, nil, nil, func(path string) ([]byte, error) {
		value, ok := files[path]
		if !ok {
			return nil, errors.New("missing file")
		}

		return []byte(value), nil
	}, nil)

	_, err := evalStringWithEvaluator(evaluator, runtime.NewEnvironment(nil), mainPath, `import "./lib.molt"`)
	runtimeErr := expectRuntimeError(t, err)
	if runtimeErr.Diagnostic().Message != `exported name "missing" is not defined at module top level` {
		t.Fatalf("message = %q, want %q", runtimeErr.Diagnostic().Message, `exported name "missing" is not defined at module top level`)
	}
}

func TestEvaluateImportDuplicateExportFailure(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.molt")
	libPath := filepath.Join(dir, "lib.molt")

	files := map[string]string{
		libPath: "" +
			"value = 1\n" +
			"export value\n" +
			"export value",
	}

	evaluator := NewWithRuntime(nil, nil, nil, func(path string) ([]byte, error) {
		value, ok := files[path]
		if !ok {
			return nil, errors.New("missing file")
		}

		return []byte(value), nil
	}, nil)

	_, err := evalStringWithEvaluator(evaluator, runtime.NewEnvironment(nil), mainPath, `import "./lib.molt"`)
	runtimeErr := expectRuntimeError(t, err)
	if runtimeErr.Diagnostic().Message != `duplicate export "value"` {
		t.Fatalf("message = %q, want %q", runtimeErr.Diagnostic().Message, `duplicate export "value"`)
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

func TestEvaluateInputReadFailure(t *testing.T) {
	evaluator := NewWithIO(errReader{err: errors.New("boom")}, nil)

	_, err := evalStringWithEvaluator(evaluator, runtime.NewEnvironment(nil), "input_failure.molt", "input()")
	runtimeErr := expectRuntimeError(t, err)
	if runtimeErr.Diagnostic().Message != "input failed: boom" {
		t.Fatalf("message = %q, want %q", runtimeErr.Diagnostic().Message, "input failed: boom")
	}
}

func TestEvaluateReadFileFailure(t *testing.T) {
	evaluator := NewWithRuntime(nil, nil, nil, func(path string) ([]byte, error) {
		return nil, errors.New("boom")
	}, nil)

	_, err := evalStringWithEvaluator(evaluator, runtime.NewEnvironment(nil), "read_file_failure.molt", `read_file("missing.txt")`)
	runtimeErr := expectRuntimeError(t, err)
	if runtimeErr.Diagnostic().Message != `read_file failed for "missing.txt": boom` {
		t.Fatalf("message = %q, want %q", runtimeErr.Diagnostic().Message, `read_file failed for "missing.txt": boom`)
	}
}

func TestEvaluateWriteFileFailure(t *testing.T) {
	evaluator := NewWithRuntime(nil, nil, nil, nil, func(path string, data []byte) error {
		return errors.New("boom")
	})

	_, err := evalStringWithEvaluator(evaluator, runtime.NewEnvironment(nil), "write_file_failure.molt", `write_file("missing.txt", "hello")`)
	runtimeErr := expectRuntimeError(t, err)
	if runtimeErr.Diagnostic().Message != `write_file failed for "missing.txt": boom` {
		t.Fatalf("message = %q, want %q", runtimeErr.Diagnostic().Message, `write_file failed for "missing.txt": boom`)
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
	files := map[string]string{
		"note.txt": "file contents",
	}

	result, err := evalStringWithEvaluator(NewWithRuntime(bytes.NewBufferString("hello\nworld"), nil, []string{"alpha", "beta"}, func(path string) ([]byte, error) {
		value, ok := files[path]
		if !ok {
			return nil, errors.New("unexpected path")
		}

		return []byte(value), nil
	}, func(path string, data []byte) error {
		files[path] = string(data)
		return nil
	}), env, "builtins.molt", ""+
		"xs = [1]\n"+
		"same = push(xs, 2)\n"+
		"rec = record { name: \"molt\", size: len(xs) }\n"+
		"fn add(a, b) = a + b\n"+
		"code = @{ 1 + 2 }\n"+
		"mut = ~{ x -> y\n1 -> 2 }\n"+
		"cli1 = args()\n"+
		"cli2 = args()\n"+
		"push(cli1, \"extra\")\n"+
		"file1 = read_file(\"note.txt\")\n"+
		"file2 = read_file(\"note.txt\")\n"+
		"write_file(\"written.txt\", \"saved\")\n"+
		"write_file(\"written.txt\", \"updated\")\n"+
		"written = read_file(\"written.txt\")\n"+
		"line1 = input()\n"+
		"line2 = input()\n"+
		"line3 = input()\n"+
		"input1 = stdin()\n"+
		"input2 = stdin()\n"+
		"[\n"+
		"  type(1),\n"+
		"  type(\"x\"),\n"+
		"  type(true),\n"+
		"  type(nil),\n"+
		"  type(xs),\n"+
		"  type(rec),\n"+
		"  type(add),\n"+
		"  type(eval),\n"+
		"  type(code),\n"+
		"  type(mut),\n"+
		"  len(xs),\n"+
		"  len(\"aé\"),\n"+
		"  same == xs,\n"+
		"  show(xs),\n"+
		"  show(rec),\n"+
		"  show(code),\n"+
		"  show(mut),\n"+
		"  show(add),\n"+
		"  cli1,\n"+
		"  cli2,\n"+
		"  file1,\n"+
		"  file2,\n"+
		"  written,\n"+
		"  line1,\n"+
		"  line2,\n"+
		"  line3,\n"+
		"  input1,\n"+
		"  input2\n"+
		"]",
	)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	values := expectValue[*runtime.ListValue](t, result)
	if len(values.Elements) != 28 {
		t.Fatalf("result length = %d, want 28", len(values.Elements))
	}

	wantTypes := []string{
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

	for i, want := range wantTypes {
		got := expectValue[*runtime.StringValue](t, values.Elements[i])
		if got.Value != want {
			t.Fatalf("type result %d = %q, want %q", i, got.Value, want)
		}
	}

	if got := expectValue[*runtime.NumberValue](t, values.Elements[10]); got.Value != 2 {
		t.Fatalf("len(xs) = %v, want 2", got.Value)
	}

	if got := expectValue[*runtime.NumberValue](t, values.Elements[11]); got.Value != 2 {
		t.Fatalf("len(\"aé\") = %v, want 2", got.Value)
	}

	if got := expectValue[*runtime.BooleanValue](t, values.Elements[12]); !got.Value {
		t.Fatalf("push did not return the mutated list")
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[13]); got.Value != `[1, 2]` {
		t.Fatalf("show(xs) = %q, want %q", got.Value, `[1, 2]`)
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[14]); got.Value != `record { name: "molt", size: 2 }` {
		t.Fatalf("show(rec) = %q, want %q", got.Value, `record { name: "molt", size: 2 }`)
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[15]); got.Value != "@{ (1 + 2) }" {
		t.Fatalf("show(code) = %q, want %q", got.Value, "@{ (1 + 2) }")
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[16]); got.Value != "~{\n  x -> y\n  1 -> 2\n}" {
		t.Fatalf("show(mut) = %q, want multiline mutation", got.Value)
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[17]); got.Value != "fn add(a, b) = (a + b)" {
		t.Fatalf("show(add) = %q, want %q", got.Value, "fn add(a, b) = (a + b)")
	}

	if got := expectValue[*runtime.ListValue](t, values.Elements[18]); runtime.ShowValue(got) != `["alpha", "beta", "extra"]` {
		t.Fatalf("mutated cli1 = %q, want %q", runtime.ShowValue(got), `["alpha", "beta", "extra"]`)
	}

	if got := expectValue[*runtime.ListValue](t, values.Elements[19]); runtime.ShowValue(got) != `["alpha", "beta"]` {
		t.Fatalf("fresh cli2 = %q, want %q", runtime.ShowValue(got), `["alpha", "beta"]`)
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[20]); got.Value != "file contents" {
		t.Fatalf("read_file first = %q, want %q", got.Value, "file contents")
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[21]); got.Value != "file contents" {
		t.Fatalf("read_file second = %q, want %q", got.Value, "file contents")
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[22]); got.Value != "updated" {
		t.Fatalf("written file = %q, want %q", got.Value, "updated")
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[23]); got.Value != "hello" {
		t.Fatalf("input first = %q, want %q", got.Value, "hello")
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[24]); got.Value != "world" {
		t.Fatalf("input second = %q, want %q", got.Value, "world")
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[25]); got.Value != "" {
		t.Fatalf("input third = %q, want empty string", got.Value)
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[26]); got.Value != "" {
		t.Fatalf("stdin() first read after input() = %q, want empty string", got.Value)
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[27]); got.Value != "" {
		t.Fatalf("stdin() second read = %q, want empty string", got.Value)
	}
}

func TestEvaluateStdlibHelpers(t *testing.T) {
	result := mustEval(t, runtime.NewEnvironment(nil), "stdlib_helpers.molt", ""+
		"parts = split(\"a,b,c\", \",\")\n"+
		"joined = join(parts, \"-\")\n"+
		"trimmed = trim(\"  hello\\n\")\n"+
		"lineParts = lines(\"one\\r\\ntwo\\nthree\\n\")\n"+
		"lineParts2 = lines(\"\")\n"+
		"replaced = replace(\"molt molt\", \"molt\", \"bolt\")\n"+
		"hasNeedle = contains(\"mutation\", \"tat\")\n"+
		"hasMissing = contains(\"mutation\", \"zzz\")\n"+
		"xs = range(5)\n"+
		"ys = range(2, 5)\n"+
		"doubled = map(xs, fn(x) = x * 2)\n"+
		"indexed = map(xs, fn(x, i) = x + i)\n"+
		"evens = filter(xs, fn(x) = x % 2 == 0)\n"+
		"prefix = filter(xs, fn(x, i) = i < 2)\n"+
		"typed = map([1, \"x\", true], type)\n"+
		"[\n"+
		"  parts,\n"+
		"  joined,\n"+
		"  trimmed,\n"+
		"  lineParts,\n"+
		"  lineParts2,\n"+
		"  replaced,\n"+
		"  hasNeedle,\n"+
		"  hasMissing,\n"+
		"  xs,\n"+
		"  ys,\n"+
		"  doubled,\n"+
		"  indexed,\n"+
		"  evens,\n"+
		"  prefix,\n"+
		"  typed,\n"+
		"  to_string([1, 2]),\n"+
		"  to_string(nil),\n"+
		"  to_number(\" 12.5 \"),\n"+
		"  to_number(7)\n"+
		"]",
	)

	values := expectValue[*runtime.ListValue](t, result)
	if len(values.Elements) != 19 {
		t.Fatalf("result length = %d, want 19", len(values.Elements))
	}

	if got := runtime.ShowValue(values.Elements[0]); got != `["a", "b", "c"]` {
		t.Fatalf("split result = %q, want %q", got, `["a", "b", "c"]`)
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[1]); got.Value != "a-b-c" {
		t.Fatalf("join result = %q, want %q", got.Value, "a-b-c")
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[2]); got.Value != "hello" {
		t.Fatalf("trim result = %q, want %q", got.Value, "hello")
	}

	if got := runtime.ShowValue(values.Elements[3]); got != `["one", "two", "three"]` {
		t.Fatalf("lines result = %q, want %q", got, `["one", "two", "three"]`)
	}

	if got := runtime.ShowValue(values.Elements[4]); got != `[]` {
		t.Fatalf("empty lines result = %q, want %q", got, `[]`)
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[5]); got.Value != "bolt bolt" {
		t.Fatalf("replace result = %q, want %q", got.Value, "bolt bolt")
	}

	if got := expectValue[*runtime.BooleanValue](t, values.Elements[6]); !got.Value {
		t.Fatalf("contains should have found substring")
	}

	if got := expectValue[*runtime.BooleanValue](t, values.Elements[7]); got.Value {
		t.Fatalf("contains should not have found missing substring")
	}

	if got := runtime.ShowValue(values.Elements[8]); got != `[0, 1, 2, 3, 4]` {
		t.Fatalf("range(5) = %q, want %q", got, `[0, 1, 2, 3, 4]`)
	}

	if got := runtime.ShowValue(values.Elements[9]); got != `[2, 3, 4]` {
		t.Fatalf("range(2, 5) = %q, want %q", got, `[2, 3, 4]`)
	}

	if got := runtime.ShowValue(values.Elements[10]); got != `[0, 2, 4, 6, 8]` {
		t.Fatalf("mapped result = %q, want %q", got, `[0, 2, 4, 6, 8]`)
	}

	if got := runtime.ShowValue(values.Elements[11]); got != `[0, 2, 4, 6, 8]` {
		t.Fatalf("indexed map result = %q, want %q", got, `[0, 2, 4, 6, 8]`)
	}

	if got := runtime.ShowValue(values.Elements[12]); got != `[0, 2, 4]` {
		t.Fatalf("filter result = %q, want %q", got, `[0, 2, 4]`)
	}

	if got := runtime.ShowValue(values.Elements[13]); got != `[0, 1]` {
		t.Fatalf("indexed filter result = %q, want %q", got, `[0, 1]`)
	}

	if got := runtime.ShowValue(values.Elements[14]); got != `["number", "string", "boolean"]` {
		t.Fatalf("native map result = %q, want %q", got, `["number", "string", "boolean"]`)
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[15]); got.Value != "[1, 2]" {
		t.Fatalf("to_string(list) = %q, want %q", got.Value, "[1, 2]")
	}

	if got := expectValue[*runtime.StringValue](t, values.Elements[16]); got.Value != "nil" {
		t.Fatalf("to_string(nil) = %q, want %q", got.Value, "nil")
	}

	if got := expectValue[*runtime.NumberValue](t, values.Elements[17]); got.Value != 12.5 {
		t.Fatalf("to_number(string) = %v, want 12.5", got.Value)
	}

	if got := expectValue[*runtime.NumberValue](t, values.Elements[18]); got.Value != 7 {
		t.Fatalf("to_number(number) = %v, want 7", got.Value)
	}
}

func TestEvaluateRecordHelpers(t *testing.T) {
	result := mustEval(t, runtime.NewEnvironment(nil), "record_helpers.molt", ""+
		"item = record { name: \"molt\", nested: record { ok: true }, count: 2 }\n"+
		"[\n"+
		"  len(item),\n"+
		"  contains(item, \"name\"),\n"+
		"  contains(item, \"missing\"),\n"+
		"  keys(item),\n"+
		"  values(item)\n"+
		"]",
	)

	values := expectValue[*runtime.ListValue](t, result)
	if len(values.Elements) != 5 {
		t.Fatalf("result length = %d, want 5", len(values.Elements))
	}

	if got := expectValue[*runtime.NumberValue](t, values.Elements[0]); got.Value != 3 {
		t.Fatalf("len(record) = %v, want 3", got.Value)
	}

	if got := expectValue[*runtime.BooleanValue](t, values.Elements[1]); !got.Value {
		t.Fatalf("contains(record, name) should be true")
	}

	if got := expectValue[*runtime.BooleanValue](t, values.Elements[2]); got.Value {
		t.Fatalf("contains(record, missing) should be false")
	}

	if got := runtime.ShowValue(values.Elements[3]); got != `["name", "nested", "count"]` {
		t.Fatalf("keys(record) = %q, want %q", got, `["name", "nested", "count"]`)
	}

	if got := runtime.ShowValue(values.Elements[4]); got != `["molt", record { ok: true }, 2]` {
		t.Fatalf("values(record) = %q, want ordered values list", got)
	}
}

func TestEvaluatePrintWritesDisplayOutputAndReturnsNil(t *testing.T) {
	env := runtime.NewEnvironment(nil)
	var output bytes.Buffer
	evaluator := &Evaluator{output: &output}

	value, err := evalStringWithEvaluator(evaluator, env, "print.molt", ""+
		"xs = [1, 2]\n"+
		"print(\"hello\")\n"+
		"print(xs)",
	)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	if _, ok := value.(runtime.NilValue); !ok {
		t.Fatalf("print result type = %T, want runtime.NilValue", value)
	}

	if output.String() != "hello\n[1, 2]\n" {
		t.Fatalf("print output = %q, want %q", output.String(), "hello\n[1, 2]\n")
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
