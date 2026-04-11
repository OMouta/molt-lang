package integration_test

import (
	"os"
	"testing"

	"molt/internal/formatter"
	"molt/internal/parser"
)

// formatSource is a test helper that parses Molt source and formats it.
func formatSource(t *testing.T, src string) string {
	t.Helper()
	prog, err := parser.Parse("<test>", src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return formatter.Format(prog)
}

// assertFormat checks that src formats to want.
func assertFormat(t *testing.T, src, want string) {
	t.Helper()
	got := formatSource(t, src)
	if got != want {
		t.Errorf("Format mismatch\n--- got ---\n%s--- want ---\n%s", got, want)
	}
}

// assertIdempotent checks that formatting the already-formatted output produces
// the same result (format(format(src)) == format(src)).
func assertIdempotent(t *testing.T, src string) {
	t.Helper()
	once := formatSource(t, src)
	twice := formatSource(t, once)
	if once != twice {
		t.Errorf("Format is not idempotent\n--- first pass ---\n%s--- second pass ---\n%s", once, twice)
	}
}

// TestFormatterLiterals covers all primitive literal types.
func TestFormatterLiterals(t *testing.T) {
	assertFormat(t, "42", "42\n")
	assertFormat(t, "3.14", "3.14\n")
	assertFormat(t, `"hello"`, `"hello"`+"\n")
	assertFormat(t, "true", "true\n")
	assertFormat(t, "false", "false\n")
	assertFormat(t, "nil", "nil\n")
}

// TestFormatterBinding covers variable assignments.
func TestFormatterBinding(t *testing.T) {
	assertFormat(t, "x = 42", "x = 42\n")
	assertFormat(t, `name = "molt"`, `name = "molt"`+"\n")
}

// TestFormatterOperators covers unary and binary expressions.
// Note: BinaryExpr formats WITHOUT wrapping parens; GroupExpr adds parens.
func TestFormatterOperators(t *testing.T) {
	assertFormat(t, "-x", "-x\n")
	assertFormat(t, "not true", "not true\n")
	// Bare binary expr – no parens added by formatter
	assertFormat(t, "x + y", "x + y\n")
	assertFormat(t, "x == y", "x == y\n")
	assertFormat(t, "a and b", "a and b\n")
	assertFormat(t, "a or b", "a or b\n")
	// Explicit parens in source are preserved through GroupExpr
	assertFormat(t, "(x + y)", "(x + y)\n")
	assertIdempotent(t, "(x + y)")
}

// TestFormatterFieldAccess covers dot-notation and index access.
func TestFormatterFieldAccess(t *testing.T) {
	assertFormat(t, "obj.field", "obj.field\n")
	assertFormat(t, "a.b.c", "a.b.c\n")
	assertFormat(t, "list[0]", "list[0]\n")
}

// TestFormatterLists covers list literals – inline and multiline.
func TestFormatterLists(t *testing.T) {
	// Empty list
	assertFormat(t, "[]", "[]\n")

	// Short list – stays inline
	assertFormat(t, "[1, 2, 3]", "[1, 2, 3]\n")

	// List that fits inline stays inline (74 chars ≤ 80)
	assertFormat(t, `["a long first element", "a long second element", "a long third element"]`,
		`["a long first element", "a long second element", "a long third element"]`+"\n")

	// Long list that exceeds 80 chars expands with trailing comma
	long := `["a very long first element value", "a very long second element value", "a very long third"]`
	wantLong := `[
  "a very long first element value",
  "a very long second element value",
  "a very long third",
]
`
	assertFormat(t, long, wantLong)
	assertIdempotent(t, long)
}

// TestFormatterRecords covers record literals – inline and multiline.
func TestFormatterRecords(t *testing.T) {
	// Empty record
	assertFormat(t, "record {}", "record {}\n")

	// Short record – inline
	assertFormat(t, `record { x: 1, y: 2 }`, "record { x: 1, y: 2 }\n")

	// Record that exceeds line limit expands with trailing comma
	// (81 chars > 80-char limit)
	long := `record { first_name: "Alexander", last_name: "Moldovanu", description: "testing" }`
	wantLong := `record {
  first_name: "Alexander",
  last_name: "Moldovanu",
  description: "testing",
}
`
	assertFormat(t, long, wantLong)
	assertIdempotent(t, long)

	// Nested record – inner stays inline if it fits
	nested := `record { name: "molt", stats: record { runs: 3, wins: 1 } }`
	assertFormat(t, nested, `record { name: "molt", stats: record { runs: 3, wins: 1 } }`+"\n")
	assertIdempotent(t, nested)
}

// TestFormatterTrailingCommas verifies that the parser now accepts trailing
// commas and the formatter canonically emits them in multiline mode.
func TestFormatterTrailingCommas(t *testing.T) {
	// Parser accepts trailing commas in lists
	assertFormat(t, "[1, 2, 3,]", "[1, 2, 3]\n")
	assertFormat(t, "[1, 2, 3]", "[1, 2, 3]\n")

	// Parser accepts trailing commas in records
	assertFormat(t, `record { x: 1, y: 2, }`, "record { x: 1, y: 2 }\n")
}

// TestFormatterBlocks covers block expressions with different statement counts.
func TestFormatterBlocks(t *testing.T) {
	// Multi-expression block always expands
	multi := "{\nx = 1\ny = 2\nx\n}"
	wantMulti := "{\n  x = 1\n  y = 2\n  x\n}\n"
	assertFormat(t, multi, wantMulti)
	assertIdempotent(t, multi)

	// Empty block
	assertFormat(t, "{}", "{}\n")
}

// TestFormatterFunctions covers named and anonymous functions.
func TestFormatterFunctions(t *testing.T) {
	// Short named function – inline
	assertFormat(t, "fn add(a, b) = a + b", "fn add(a, b) = a + b\n")

	// Named function with a short if-else body – still inline
	assertFormat(t, "fn abs(x) = if x < 0 -> -x else -> x",
		"fn abs(x) = if x < 0 -> -x else -> x\n")
	assertIdempotent(t, "fn abs(x) = if x < 0 -> -x else -> x")

	// Named function with block body
	src := "fn greet(name) = {\nmsg = name\nio.print(msg)\nmsg\n}"
	want := "fn greet(name) = {\n  msg = name\n  io.print(msg)\n  msg\n}\n"
	assertFormat(t, src, want)
	assertIdempotent(t, src)

	// Anonymous function – short
	assertFormat(t, "fn(x) = x", "fn(x) = x\n")

	// Zero-param function
	assertFormat(t, "fn() = 42", "fn() = 42\n")

	// Function call
	assertFormat(t, "add(1, 2)", "add(1, 2)\n")
	assertFormat(t, `io.print("hi")`, `io.print("hi")`+"\n")
}

// TestFormatterTopLevelGrouping covers blank-line insertion around named functions.
func TestFormatterTopLevelGrouping(t *testing.T) {
	// Bindings with a named function in between get blank lines.
	src := "x = 1\nfn add(a, b) = a + b\ny = 2"
	want := "x = 1\n\nfn add(a, b) = a + b\n\ny = 2\n"
	assertFormat(t, src, want)
	assertIdempotent(t, src)

	// Two consecutive functions get a blank line between them.
	twoFuncs := "fn a() = 1\nfn b() = 2"
	wantTwo := "fn a() = 1\n\nfn b() = 2\n"
	assertFormat(t, twoFuncs, wantTwo)
	assertIdempotent(t, twoFuncs)

	// Plain bindings stay grouped (no blank lines between them).
	bindings := "x = 1\ny = 2\nz = 3"
	assertFormat(t, bindings, "x = 1\ny = 2\nz = 3\n")
}

// TestFormatterImportGrouping covers import-block placement and blank line.
func TestFormatterImportGrouping(t *testing.T) {
	src := "import \"std:io\"\nimport \"std:collections\" as c\nx = 1"
	want := "import \"std:io\"\nimport \"std:collections\" as c\n\nx = 1\n"
	assertFormat(t, src, want)
	assertIdempotent(t, src)

	// Imports-only file: no trailing blank line
	assertFormat(t, `import "std:io"`, `import "std:io"`+"\n")
}

// TestFormatterImportForms covers all import syntaxes.
func TestFormatterImportForms(t *testing.T) {
	// Module import
	assertFormat(t, `import "std:io"`, `import "std:io"`+"\n")

	// Module import with alias
	assertFormat(t, `import "std:collections" as c`, `import "std:collections" as c`+"\n")

	// Named import – single name (canonical: no braces)
	assertFormat(t, `import foo from "./mod.molt"`, `import foo from "./mod.molt"`+"\n")

	// Named import – brace form with single name normalizes to no-brace form
	assertFormat(t, `import {foo} from "./mod.molt"`, `import foo from "./mod.molt"`+"\n")

	// Named import – multiple names keep braces
	assertFormat(t, `import {a, b} from "./mod.molt"`, `import {a, b} from "./mod.molt"`+"\n")
}

// TestFormatterExport covers export statements.
func TestFormatterExport(t *testing.T) {
	assertFormat(t, "export foo", "export foo\n")
}

// TestFormatterConditional covers if/else expressions.
func TestFormatterConditional(t *testing.T) {
	// Short if-else – inline
	assertFormat(t, `if x -> "yes" else -> "no"`, `if x -> "yes" else -> "no"`+"\n")
	assertIdempotent(t, `if x -> "yes" else -> "no"`)

	// if without else
	assertFormat(t, `if x -> "yes"`, `if x -> "yes"`+"\n")

	// if/else with multi-statement block bodies (2+ stmts → always BlockExpr)
	src := "if x -> {\na = 1\nb = 2\na\n} else -> {\nc = 3\nc\n}"
	want := "if x -> {\n  a = 1\n  b = 2\n  a\n} else -> {\n  c = 3\n  c\n}\n"
	assertFormat(t, src, want)
	assertIdempotent(t, src)
}

// TestFormatterWhile covers while loops.
func TestFormatterWhile(t *testing.T) {
	// Single-statement body – wrapped as inline block
	assertFormat(t, "while x < 3 -> x = x",
		"while x < 3 -> { x = x }\n")
	assertIdempotent(t, "while x < 3 -> x = x")

	// Single-statement body with explicit parens – preserves them
	assertFormat(t, "while (x < 3) -> x = x",
		"while (x < 3) -> { x = x }\n")
	assertIdempotent(t, "while (x < 3) -> x = x")

	// Multi-statement body – expanded block
	src := "while x < 3 -> {\nx = x + 1\ny = y + 1\n}"
	want := "while x < 3 -> {\n  x = x + 1\n  y = y + 1\n}\n"
	assertFormat(t, src, want)
	assertIdempotent(t, src)
}

// TestFormatterForIn covers for-in loops.
func TestFormatterForIn(t *testing.T) {
	// Single-statement body – wrapped as inline block
	assertFormat(t, `for item in xs -> io.print(item)`,
		`for item in xs -> { io.print(item) }`+"\n")
	assertIdempotent(t, `for item in xs -> io.print(item)`)

	// Multi-statement body – expanded block
	src := "for item in xs -> {\nio.print(item)\ntotal = total + item\n}"
	want := "for item in xs -> {\n  io.print(item)\n  total = total + item\n}\n"
	assertFormat(t, src, want)
	assertIdempotent(t, src)

	// Destructuring binding in for
	assertFormat(t, `for [a, b] in pairs -> io.print(a)`,
		`for [a, b] in pairs -> { io.print(a) }`+"\n")
	assertIdempotent(t, `for [a, b] in pairs -> io.print(a)`)
}

// TestFormatterBreakContinue covers break and continue.
func TestFormatterBreakContinue(t *testing.T) {
	assertFormat(t, "for x in xs -> break", "for x in xs -> { break }\n")
	assertFormat(t, "for x in xs -> continue", "for x in xs -> { continue }\n")
}

// TestFormatterMatch covers match expressions – always multiline.
func TestFormatterMatch(t *testing.T) {
	// Match cases must be on separate lines in source
	src := "match value {\n1 -> \"one\"\n2 -> \"two\"\n_ -> \"other\"\n}"
	want := "match value {\n  1 -> \"one\"\n  2 -> \"two\"\n  _ -> \"other\"\n}\n"
	assertFormat(t, src, want)
	assertIdempotent(t, src)

	// Empty match
	assertFormat(t, "match x {}", "match x {}\n")

	// Match inside a function
	funcSrc := "fn describe(v) = match v {\n1 -> \"one\"\n_ -> \"other\"\n}"
	wantFunc := "fn describe(v) = match v {\n  1 -> \"one\"\n  _ -> \"other\"\n}\n"
	assertFormat(t, funcSrc, wantFunc)
	assertIdempotent(t, funcSrc)
}

// TestFormatterTryCatch covers try/catch expressions.
func TestFormatterTryCatch(t *testing.T) {
	// Short – inline
	src := `try c.len(1) catch err -> err.message`
	assertFormat(t, src, src+"\n")
	assertIdempotent(t, src)

	// Multi-statement try block
	trySrc := "try {\nval = riskyOp()\nval + 1\n} catch err -> {\nio.print(err.message)\nnil\n}"
	wantTry := "try {\n  val = riskyOp()\n  val + 1\n} catch err -> {\n  io.print(err.message)\n  nil\n}\n"
	assertFormat(t, trySrc, wantTry)
	assertIdempotent(t, trySrc)
}

// TestFormatterDestructuring covers list and record destructuring in bindings.
func TestFormatterDestructuring(t *testing.T) {
	// List destructuring
	assertFormat(t, "[a, b] = pair", "[a, b] = pair\n")
	assertIdempotent(t, "[a, b] = pair")

	// Nested list destructuring
	assertFormat(t, "[a, [b, c]] = nested", "[a, [b, c]] = nested\n")
	assertIdempotent(t, "[a, [b, c]] = nested")

	// Record destructuring
	assertFormat(t, `record { name: who } = p`, "record { name: who } = p\n")
	assertFormat(t, `record { name: who, age: n } = p`, "record { name: who, age: n } = p\n")
	assertIdempotent(t, `record { name: who, age: n } = p`)
}

// TestFormatterQuote covers @{} quote expressions.
func TestFormatterQuote(t *testing.T) {
	// Short quote – inline
	assertFormat(t, "@{ x + 1 }", "@{ x + 1 }\n")
	assertIdempotent(t, "@{ x + 1 }")

	// Multi-expression quote body – expanded
	src := "@{ x = 1\nx + y }"
	want := "@{\n  x = 1\n  x + y\n}\n"
	assertFormat(t, src, want)
	assertIdempotent(t, src)
}

// TestFormatterUnquoteSplice covers ~() and ~[] forms (valid inside quotes).
func TestFormatterUnquoteSplice(t *testing.T) {
	assertFormat(t, "@{ ~(expr) }", "@{ ~(expr) }\n")
	assertFormat(t, "@{ ~[list] }", "@{ ~[list] }\n")
	assertIdempotent(t, "@{ ~(expr) }")
	assertIdempotent(t, "@{ ~[list] }")
}

// TestFormatterMutation covers ~{} mutation rule literals.
func TestFormatterMutation(t *testing.T) {
	// Single short rule – inline  (the ~{ is part of apply, not a separate ~)
	assertFormat(t, "add ~{ + -> * }", "add ~{ + -> * }\n")
	assertIdempotent(t, "add ~{ + -> * }")

	// Standalone mutation literal with multiple rules (rules need newlines)
	src := "~{\n($x + 0) -> $x\n(0 + $x) -> $x\n}"
	want := "~{\n  ($x + 0) -> $x\n  (0 + $x) -> $x\n}\n"
	assertFormat(t, src, want)
	assertIdempotent(t, src)

	// Wildcard pattern in mutation
	src2 := "~{\n[1, ...$tail, 4] -> [0, ...$tail]\n_ -> nil\n}"
	want2 := "~{\n  [1, ...$tail, 4] -> [0, ...$tail]\n  _ -> nil\n}\n"
	assertFormat(t, src2, want2)
	assertIdempotent(t, src2)
}

// TestFormatterApplyMutation covers the ~ apply operator with named mutations.
func TestFormatterApplyMutation(t *testing.T) {
	// Apply with a named mutation variable uses " ~ " infix
	assertFormat(t, "@{ x + 0 } ~ simplify", "@{ x + 0 } ~ simplify\n")
	assertIdempotent(t, "@{ x + 0 } ~ simplify")
}

// TestFormatterMutationCapture covers $capture and _ wildcard patterns.
// Single short rules stay inline ("~{ pattern -> replacement }").
func TestFormatterMutationCapture(t *testing.T) {
	assertFormat(t, "~{\n$x -> $x\n}", "~{ $x -> $x }\n")
	assertIdempotent(t, "~{\n$x -> $x\n}")
	assertFormat(t, "~{\n_ -> nil\n}", "~{ _ -> nil }\n")
	assertIdempotent(t, "~{\n_ -> nil\n}")
	assertFormat(t, "~{\n...$rest -> nil\n}", "~{ ...$rest -> nil }\n")
	assertIdempotent(t, "~{\n...$rest -> nil\n}")
}

// TestFormatterIdempotencyExamples runs idempotency checks against programs
// that exercise multiple features at once.
func TestFormatterIdempotencyExamples(t *testing.T) {
	programs := []string{
		// Capture mutation program
		`import "std:io"
import "std:meta"

value = 7

simplify = ~{
  ($x + 0) -> $x
  (0 + $x) -> $x
}

io.print(meta.eval(@{ 0 + (value + 0) } ~ simplify))
`,
		// Destructuring program
		`import "std:io"

pair = [1, [2, 3]]
[left, [middle, right]] = pair
io.print([left, middle, right])

profile = record { name: "molt", stats: record { runs: 4 } }
record { name: who, stats: record { runs: count } } = profile
io.print([who, count])
`,
		// Error handling
		`import "std:errors"
import "std:io"

result = try errors.throw(errors.error("bad", record { code: 1 })) catch err -> err.message
io.print(result)
`,
	}

	for _, prog := range programs {
		assertIdempotent(t, prog)
	}
}

// TestFormatterRealExamples formats actual example files and checks idempotency.
func TestFormatterRealExamples(t *testing.T) {
	examples := []string{
		"examples/basic/capture_mutation.molt",
		"examples/basic/destructuring.molt",
		"examples/errors/try_catch.molt",
	}

	for _, rel := range examples {
		t.Run(rel, func(t *testing.T) {
			path := repoPath(t, rel)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", rel, err)
			}
			assertIdempotent(t, string(data))
		})
	}
}
