package integration_test

import "testing"

func TestSpecExamples(t *testing.T) {
	t.Run("function definition", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_fn_definition.molt", ""+
			"fn add(a, b) = a + b\n"+
			"add(2, 3)",
		)
		expectShownValue(t, value, "5")
	})

	t.Run("function block body", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_fn_block.molt", ""+
			"fn abs(x) = {\n"+
			"  if x < 0 -> -x\n"+
			"  else -> x\n"+
			"}\n"+
			"[abs(-2), abs(2)]",
		)
		expectShownValue(t, value, "[2, 2]")
	})

	t.Run("anonymous function", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_fn_anon.molt", ""+
			"f = fn(x) = x + 1\n"+
			"f(2)",
		)
		expectShownValue(t, value, "3")
	})

	t.Run("quoted argument sugar", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_quote_sugar.molt", ""+
			"fn warp(code) = eval(code)\n"+
			"warp @{ 1 + 2 }",
		)
		expectShownValue(t, value, "3")
	})

	t.Run("block value", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_block.molt", "{\n1\n2\n3\n}")
		expectShownValue(t, value, "3")
	})

	t.Run("list indexing", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_list_index.molt", ""+
			"xs = [1, 2, 3]\n"+
			"xs[0]",
		)
		expectShownValue(t, value, "1")
	})

	t.Run("record literal", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_record_literal.molt", ""+
			"item = record { name: \"molt\", nested: record { ok: true } }\n"+
			"[show(item), type(item)]",
		)
		expectShownValue(t, value, "[\n  \"record { name: \\\"molt\\\", nested: record { ok: true } }\",\n  \"record\"\n]")
	})

	t.Run("record field access", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_record_field_access.molt", ""+
			"item = record { name: \"molt\", nested: record { ok: true } }\n"+
			"[item.name, item.nested.ok]",
		)
		expectShownValue(t, value, "[\"molt\", true]")
	})

	t.Run("record field assignment", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_record_field_assignment.molt", ""+
			"item = record { name: \"molt\", nested: record { ok: true } }\n"+
			"item.nested.ok = false\n"+
			"item.count = 2\n"+
			"[item.nested.ok, item.count, keys(item)]",
		)
		expectShownValue(t, value, `[false, 2, ["name", "nested", "count"]]`)
	})

	t.Run("record helpers", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_record_helpers.molt", ""+
			"item = record { name: \"molt\", nested: record { ok: true } }\n"+
			"[len(item), contains(item, \"name\"), keys(item), values(item)]",
		)
		expectShownValue(t, value, `[2, true, ["name", "nested"], ["molt", record { ok: true }]]`)
	})

	t.Run("error values", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_error_values.molt", ""+
			"err = error(\"missing file\", record { path: \"note.txt\" })\n"+
			"[type(err), err.message, err.data.path]",
		)
		expectShownValue(t, value, `["error", "missing file", "note.txt"]`)
	})

	t.Run("try catch", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_try_catch.molt", ""+
			"try throw(error(\"boom\", record { code: 7 })) catch err -> [err.message, err.data.code]",
		)
		expectShownValue(t, value, `["boom", 7]`)
	})

	t.Run("match expression", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_match.molt", ""+
			"kind = match 2 {\n"+
			"  1 -> \"one\"\n"+
			"  2 -> \"two\"\n"+
			"  _ -> \"many\"\n"+
			"}\n"+
			"captured = match \"molt\" {\n"+
			"  name -> name\n"+
			"}\n"+
			"[kind, captured]",
		)
		expectShownValue(t, value, `["two", "molt"]`)
	})

	t.Run("while loop", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_while_loop.molt", ""+
			"x = 0\n"+
			"loop = while x < 2 -> x = x + 1\n"+
			"[loop, x]",
		)
		expectShownValue(t, value, "[nil, 2]")
	})

	t.Run("for loop", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_for_loop.molt", ""+
			"total = 0\n"+
			"for item in [1, 2, 3] -> total = total + item\n"+
			"chars = []\n"+
			"for ch in \"ok\" -> push(chars, ch)\n"+
			"[total, chars]",
		)
		expectShownValue(t, value, `[6, ["o", "k"]]`)
	})

	t.Run("loop control", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_loop_control.molt", ""+
			"xs = []\n"+
			"for item in [1, 2, 3, 4] -> {\n"+
			"  if item == 2 -> continue else -> nil\n"+
			"  if item == 4 -> break else -> nil\n"+
			"  push(xs, item)\n"+
			"}\n"+
			"xs",
		)
		expectShownValue(t, value, `[1, 3]`)
	})

	t.Run("conditional", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_conditional.molt", "if true -> 1 else -> 2")
		expectShownValue(t, value, "1")
	})

	t.Run("conditional without else", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_conditional_without_else.molt", ""+
			"seen = []\n"+
			"if true -> push(seen, 1)\n"+
			"if false -> push(seen, 2)\n"+
			"seen",
		)
		expectShownValue(t, value, `[1]`)
	})

	t.Run("quote example", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_quote_example.molt", ""+
			"x = 10\n"+
			"code = @{ x + 1 }\n"+
			"eval(code)",
		)
		expectShownValue(t, value, "11")
	})

	t.Run("quote reevaluation", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_quote_reeval.molt", ""+
			"x = 0\n"+
			"code = @{ x = x + 1 }\n"+
			"eval(code)\n"+
			"eval(code)\n"+
			"x",
		)
		expectShownValue(t, value, "2")
	})

	t.Run("mutation as value", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_mutation_value.molt", ""+
			"m = ~{ + -> * }\n"+
			"type(m)",
		)
		expectStringValue(t, value, "mutation")
	})

	t.Run("mutation application", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_mutation_apply.molt", ""+
			"code = @{ 2 + 3 }\n"+
			"code2 = code ~{ + -> * }\n"+
			"eval(code2)",
		)
		expectShownValue(t, value, "6")
	})

	t.Run("mutation composition", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_mutation_compose.molt", ""+
			"m1 = ~{ + -> * }\n"+
			"m2 = ~{ 1 -> 2 }\n"+
			"m3 = m1 ~ m2\n"+
			"eval(@{ 1 + 1 } ~ m3)",
		)
		expectShownValue(t, value, "4")
	})

	t.Run("function mutation", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_function_mutation.molt", ""+
			"fn add(a, b) = a + b\n"+
			"fn mul = add ~{ + -> * }\n"+
			"mul(2, 3)",
		)
		expectShownValue(t, value, "6")
	})

	t.Run("show example", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_show_example.molt", ""+
			"code = @{ 2 + 3 }\n"+
			"show(code)",
		)
		expectStringValue(t, value, "@{ (2 + 3) }")
	})

	t.Run("basic mutation example", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_basic_mutation.molt", ""+
			"fn add(a, b) = a + b\n"+
			"fn mul = add ~{ + -> * }\n"+
			"mul(2, 3)",
		)
		expectShownValue(t, value, "6")
	})

	t.Run("code mutation example", func(t *testing.T) {
		_, output := mustExecuteProgram(t, "spec_code_mutation.molt", ""+
			"code = @{ 2 + 3 }\n"+
			"print(eval(code ~{ + -> * }))",
		)
		if output != "6\n" {
			t.Fatalf("output = %q, want %q", output, "6\n")
		}
	})

	t.Run("dynamic mutation example", func(t *testing.T) {
		value, _ := mustExecuteProgram(t, "spec_dynamic_mutation.molt", ""+
			"fn warp(code) = eval(code ~{ + -> * })\n"+
			"warp @{ 2 + 3 }",
		)
		expectShownValue(t, value, "6")
	})

	t.Run("compare worlds example", func(t *testing.T) {
		_, output := mustExecuteProgram(t, "spec_compare_worlds.molt", ""+
			"fn compare(code) = {\n"+
			"  print(eval(code))\n"+
			"  print(eval(code ~{ + -> * }))\n"+
			"}\n"+
			"compare @{ 2 + 3 }",
		)
		if output != "5\n6\n" {
			t.Fatalf("output = %q, want %q", output, "5\n6\n")
		}
	})
}
