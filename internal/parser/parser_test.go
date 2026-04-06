package parser

import (
	"testing"

	"molt/internal/ast"
	"molt/internal/diagnostic"
)

func TestParsePrimaryFormsAndSequences(t *testing.T) {
	program := mustParse(t, "primary.molt", "{\n  [1, 2]\n  (\"ok\")\n  nil\n  export value\n  import \"./lib.molt\"\n}")

	if len(program.Expressions) != 1 {
		t.Fatalf("program expression count = %d, want 1", len(program.Expressions))
	}

	block := expectExpr[*ast.BlockExpr](t, program.Expressions[0])
	if len(block.Expressions) != 5 {
		t.Fatalf("block expression count = %d, want 5", len(block.Expressions))
	}

	list := expectExpr[*ast.ListLiteral](t, block.Expressions[0])
	if len(list.Elements) != 2 {
		t.Fatalf("list element count = %d, want 2", len(list.Elements))
	}

	group := expectExpr[*ast.GroupExpr](t, block.Expressions[1])
	stringExpr := expectExpr[*ast.StringLiteral](t, group.Inner)
	if stringExpr.Value != "ok" {
		t.Fatalf("string literal = %q, want %q", stringExpr.Value, "ok")
	}

	if _, ok := block.Expressions[2].(*ast.NilLiteral); !ok {
		t.Fatalf("expected nil literal, got %T", block.Expressions[2])
	}

	exportExpr := expectExpr[*ast.ExportExpr](t, block.Expressions[3])
	if exportExpr.Name.Name != "value" {
		t.Fatalf("export name = %q, want %q", exportExpr.Name.Name, "value")
	}

	importExpr := expectExpr[*ast.ImportExpr](t, block.Expressions[4])
	if importExpr.Path.Value != "./lib.molt" {
		t.Fatalf("import path = %q, want %q", importExpr.Path.Value, "./lib.molt")
	}
}

func TestParseFunctionSyntax(t *testing.T) {
	program := mustParse(t, "functions.molt", "fn add(a, b) = a + b\nfn mul = add ~{ + -> * }\nfn(x) = {\n  x\n  x\n}")

	if len(program.Expressions) != 3 {
		t.Fatalf("program expression count = %d, want 3", len(program.Expressions))
	}

	named := expectExpr[*ast.NamedFunctionExpr](t, program.Expressions[0])
	if named.Name.Name != "add" {
		t.Fatalf("named function name = %q, want %q", named.Name.Name, "add")
	}

	if len(named.Parameters) != 2 {
		t.Fatalf("named function parameter count = %d, want 2", len(named.Parameters))
	}

	if _, ok := named.Body.(*ast.BinaryExpr); !ok {
		t.Fatalf("expected binary body, got %T", named.Body)
	}

	alias := expectExpr[*ast.AssignmentExpr](t, program.Expressions[1])
	if alias.Target.Name != "mul" {
		t.Fatalf("alias target = %q, want %q", alias.Target.Name, "mul")
	}

	if _, ok := alias.Value.(*ast.ApplyMutationExpr); !ok {
		t.Fatalf("alias value = %T, want *ast.ApplyMutationExpr", alias.Value)
	}

	anon := expectExpr[*ast.FunctionLiteralExpr](t, program.Expressions[2])
	if len(anon.Parameters) != 1 || anon.Parameters[0].Name != "x" {
		t.Fatalf("anonymous function parameter mismatch")
	}

	block := expectExpr[*ast.BlockExpr](t, anon.Body)
	if len(block.Expressions) != 2 {
		t.Fatalf("anonymous function block expression count = %d, want 2", len(block.Expressions))
	}
}

func TestParseQuoteMutationAndPostfixForms(t *testing.T) {
	program := mustParse(t, "postfix.molt", "warp @{ 1 + 2 }\ncode ~{ + -> * }\nm1 ~ m2\nxs[0]\nf(1, 2, 3)")

	if len(program.Expressions) != 5 {
		t.Fatalf("program expression count = %d, want 5", len(program.Expressions))
	}

	sugar := expectExpr[*ast.CallExpr](t, program.Expressions[0])
	if len(sugar.Arguments) != 1 {
		t.Fatalf("quoted sugar argument count = %d, want 1", len(sugar.Arguments))
	}

	quote := expectExpr[*ast.QuoteExpr](t, sugar.Arguments[0])
	if _, ok := quote.Body.(*ast.BinaryExpr); !ok {
		t.Fatalf("quote body = %T, want *ast.BinaryExpr", quote.Body)
	}

	applied := expectExpr[*ast.ApplyMutationExpr](t, program.Expressions[1])
	mutation := expectExpr[*ast.MutationLiteralExpr](t, applied.Mutation)
	if len(mutation.Rules) != 1 {
		t.Fatalf("mutation rule count = %d, want 1", len(mutation.Rules))
	}

	pattern := expectExpr[*ast.OperatorLiteral](t, mutation.Rules[0].Pattern)
	replacement := expectExpr[*ast.OperatorLiteral](t, mutation.Rules[0].Replacement)
	if pattern.Symbol != "+" || replacement.Symbol != "*" {
		t.Fatalf("operator rule mismatch: %q -> %q", pattern.Symbol, replacement.Symbol)
	}

	composed := expectExpr[*ast.ApplyMutationExpr](t, program.Expressions[2])
	if _, ok := composed.Target.(*ast.Identifier); !ok {
		t.Fatalf("composition target = %T, want identifier", composed.Target)
	}

	if _, ok := composed.Mutation.(*ast.Identifier); !ok {
		t.Fatalf("composition mutation = %T, want identifier", composed.Mutation)
	}

	if _, ok := program.Expressions[3].(*ast.IndexExpr); !ok {
		t.Fatalf("expected index expression, got %T", program.Expressions[3])
	}

	call := expectExpr[*ast.CallExpr](t, program.Expressions[4])
	if len(call.Arguments) != 3 {
		t.Fatalf("call argument count = %d, want 3", len(call.Arguments))
	}
}

func TestParseMutationApplicationIsLeftAssociative(t *testing.T) {
	program := mustParse(t, "mutation_chain.molt", "code ~ m1 ~ m2")
	if len(program.Expressions) != 1 {
		t.Fatalf("program expression count = %d, want 1", len(program.Expressions))
	}

	outer := expectExpr[*ast.ApplyMutationExpr](t, program.Expressions[0])
	inner := expectExpr[*ast.ApplyMutationExpr](t, outer.Target)
	if _, ok := inner.Target.(*ast.Identifier); !ok {
		t.Fatalf("inner target = %T, want identifier", inner.Target)
	}

	if mutation := expectExpr[*ast.Identifier](t, inner.Mutation); mutation.Name != "m1" {
		t.Fatalf("inner mutation = %q, want %q", mutation.Name, "m1")
	}

	if mutation := expectExpr[*ast.Identifier](t, outer.Mutation); mutation.Name != "m2" {
		t.Fatalf("outer mutation = %q, want %q", mutation.Name, "m2")
	}
}

func TestParsePrecedenceAndAssociativity(t *testing.T) {
	program := mustParse(t, "precedence.molt", "x = a or b and c == d + e * -f\nif cond -> left = 1 else -> right = 2")

	if len(program.Expressions) != 2 {
		t.Fatalf("program expression count = %d, want 2", len(program.Expressions))
	}

	assign := expectExpr[*ast.AssignmentExpr](t, program.Expressions[0])
	if assign.Target.Name != "x" {
		t.Fatalf("assignment target = %q, want %q", assign.Target.Name, "x")
	}

	orExpr := expectExpr[*ast.BinaryExpr](t, assign.Value)
	if orExpr.Operator != ast.BinaryOr {
		t.Fatalf("top operator = %q, want %q", orExpr.Operator, ast.BinaryOr)
	}

	andExpr := expectExpr[*ast.BinaryExpr](t, orExpr.Right)
	if andExpr.Operator != ast.BinaryAnd {
		t.Fatalf("right operator = %q, want %q", andExpr.Operator, ast.BinaryAnd)
	}

	equality := expectExpr[*ast.BinaryExpr](t, andExpr.Right)
	if equality.Operator != ast.BinaryEqual {
		t.Fatalf("equality operator = %q, want %q", equality.Operator, ast.BinaryEqual)
	}

	add := expectExpr[*ast.BinaryExpr](t, equality.Right)
	if add.Operator != ast.BinaryAdd {
		t.Fatalf("add operator = %q, want %q", add.Operator, ast.BinaryAdd)
	}

	multiply := expectExpr[*ast.BinaryExpr](t, add.Right)
	if multiply.Operator != ast.BinaryMultiply {
		t.Fatalf("multiply operator = %q, want %q", multiply.Operator, ast.BinaryMultiply)
	}

	unary := expectExpr[*ast.UnaryExpr](t, multiply.Right)
	if unary.Operator != ast.UnaryNegate {
		t.Fatalf("unary operator = %q, want %q", unary.Operator, ast.UnaryNegate)
	}

	conditional := expectExpr[*ast.ConditionalExpr](t, program.Expressions[1])
	if _, ok := conditional.ThenBranch.(*ast.AssignmentExpr); !ok {
		t.Fatalf("then branch = %T, want assignment", conditional.ThenBranch)
	}

	if _, ok := conditional.ElseBranch.(*ast.AssignmentExpr); !ok {
		t.Fatalf("else branch = %T, want assignment", conditional.ElseBranch)
	}
}

func TestParseMultiRuleMutationAndQuotedBlock(t *testing.T) {
	program := mustParse(t, "meta.molt", "@{ x = 1\nx + 2 }\n~{ x -> y\n1 -> 2\n(a + b) -> (a * b) }")

	if len(program.Expressions) != 2 {
		t.Fatalf("program expression count = %d, want 2", len(program.Expressions))
	}

	quote := expectExpr[*ast.QuoteExpr](t, program.Expressions[0])
	block := expectExpr[*ast.BlockExpr](t, quote.Body)
	if len(block.Expressions) != 2 {
		t.Fatalf("quoted block expression count = %d, want 2", len(block.Expressions))
	}

	mutation := expectExpr[*ast.MutationLiteralExpr](t, program.Expressions[1])
	if len(mutation.Rules) != 3 {
		t.Fatalf("mutation rule count = %d, want 3", len(mutation.Rules))
	}

	if _, ok := mutation.Rules[0].Pattern.(*ast.Identifier); !ok {
		t.Fatalf("first pattern = %T, want identifier", mutation.Rules[0].Pattern)
	}

	if _, ok := mutation.Rules[1].Pattern.(*ast.NumberLiteral); !ok {
		t.Fatalf("second pattern = %T, want number literal", mutation.Rules[1].Pattern)
	}

	group := expectExpr[*ast.GroupExpr](t, mutation.Rules[2].Pattern)
	if _, ok := group.Inner.(*ast.BinaryExpr); !ok {
		t.Fatalf("third pattern inner = %T, want binary expression", group.Inner)
	}
}

func TestParseSpecExamples(t *testing.T) {
	input := "" +
		"fn add(a, b) = a + b\n" +
		"fn mul = add ~{ + -> * }\n" +
		"code = @{ 2 + 3 }\n" +
		"print(eval(code ~{ + -> * }))\n" +
		"fn warp(code) = eval(code ~{ + -> * })\n" +
		"warp @{ 2 + 3 }\n" +
		"fn compare(code) = {\n" +
		"  print(eval(code))\n" +
		"  print(eval(code ~{ + -> * }))\n" +
		"}"

	program := mustParse(t, "spec_examples.molt", input)
	if len(program.Expressions) != 7 {
		t.Fatalf("program expression count = %d, want 7", len(program.Expressions))
	}
}

func TestParseRejectsMalformedPrograms(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		message string
	}{
		{name: "missing closing paren", input: "f(1, 2", message: "expected ')' after list"},
		{name: "invalid assignment target", input: "(x) = 1", message: "invalid assignment target; expected identifier"},
		{name: "chained relational", input: "a < b < c", message: "chained relational operators are not allowed"},
		{name: "chained equality", input: "a == b != c", message: "chained equality operators are not allowed"},
		{name: "missing else", input: "if x -> y", message: "expected 'else' after then branch"},
		{name: "same-line block sequence", input: "{ a b }", message: "expected line break or '}' after expression"},
		{name: "export missing name", input: "export 1", message: "expected identifier after 'export'"},
		{name: "import missing path", input: "import x", message: "expected string literal after 'import'"},
		{name: "missing mutation arrow", input: "~{ x y }", message: "expected '->' in mutation rule"},
		{name: "missing mutation operand", input: "code ~\nnext", message: "expected mutation after '~'"},
		{name: "trailing comma", input: "[1,]", message: "expected expression after ','"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.name+".molt", tc.input)
			parseErr := expectParseError(t, err)

			if parseErr.Diagnostic().Message != tc.message {
				t.Fatalf("message = %q, want %q", parseErr.Diagnostic().Message, tc.message)
			}
		})
	}
}

func TestParseProducesStableProgramSpan(t *testing.T) {
	program := mustParse(t, "span.molt", "a\nb")
	if len(program.Expressions) != 2 {
		t.Fatalf("program expression count = %d, want 2", len(program.Expressions))
	}

	if program.Span().Start.Line != 1 || program.Span().End.Line != 2 {
		t.Fatalf("program span = %d:%d -> %d:%d, want lines 1 -> 2",
			program.Span().Start.Line,
			program.Span().Start.Column,
			program.Span().End.Line,
			program.Span().End.Column,
		)
	}
}

func mustParse(t *testing.T, path, input string) *ast.Program {
	t.Helper()

	program, err := Parse(path, input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	return program
}

func expectExpr[T any](t *testing.T, expr ast.Expr) T {
	t.Helper()

	value, ok := any(expr).(T)
	if !ok {
		t.Fatalf("expression type = %T, want %T", expr, value)
	}

	return value
}

func expectParseError(t *testing.T, err error) diagnostic.ParseError {
	t.Helper()

	if err == nil {
		t.Fatalf("expected parse error, got nil")
	}

	parseErr, ok := err.(diagnostic.ParseError)
	if !ok {
		t.Fatalf("expected diagnostic.ParseError, got %T", err)
	}

	return parseErr
}
