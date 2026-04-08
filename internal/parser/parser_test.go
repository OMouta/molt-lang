package parser

import (
	"testing"

	"molt/internal/ast"
	"molt/internal/diagnostic"
)

func TestParsePrimaryFormsAndSequences(t *testing.T) {
	program := mustParse(t, "primary.molt", "{\n  [1, 2]\n  (\"ok\")\n  nil\n  break\n  continue\n  export value\n  import \"./lib.molt\"\n  record { answer: 42 }\n}")

	if len(program.Expressions) != 1 {
		t.Fatalf("program expression count = %d, want 1", len(program.Expressions))
	}

	block := expectExpr[*ast.BlockExpr](t, program.Expressions[0])
	if len(block.Expressions) != 8 {
		t.Fatalf("block expression count = %d, want 8", len(block.Expressions))
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

	if _, ok := block.Expressions[3].(*ast.BreakExpr); !ok {
		t.Fatalf("expected break expression, got %T", block.Expressions[3])
	}

	if _, ok := block.Expressions[4].(*ast.ContinueExpr); !ok {
		t.Fatalf("expected continue expression, got %T", block.Expressions[4])
	}

	exportExpr := expectExpr[*ast.ExportExpr](t, block.Expressions[5])
	if exportExpr.Name.Name != "value" {
		t.Fatalf("export name = %q, want %q", exportExpr.Name.Name, "value")
	}

	importExpr := expectExpr[*ast.ImportExpr](t, block.Expressions[6])
	if importExpr.Path.Value != "./lib.molt" {
		t.Fatalf("import path = %q, want %q", importExpr.Path.Value, "./lib.molt")
	}

	record := expectExpr[*ast.RecordLiteral](t, block.Expressions[7])
	if len(record.Fields) != 1 {
		t.Fatalf("record field count = %d, want 1", len(record.Fields))
	}

	if record.Fields[0].Name.Name != "answer" {
		t.Fatalf("record field name = %q, want %q", record.Fields[0].Name.Name, "answer")
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
	aliasTarget := expectExpr[*ast.Identifier](t, alias.Target)
	if aliasTarget.Name != "mul" {
		t.Fatalf("alias target = %q, want %q", aliasTarget.Name, "mul")
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
	program := mustParse(t, "postfix.molt", "warp @{ 1 + 2 }\ncode ~{ + -> * }\nm1 ~ m2\nxs[0]\nuser.name\nusers[0].profile.name\nf(1, 2, 3)")

	if len(program.Expressions) != 7 {
		t.Fatalf("program expression count = %d, want 7", len(program.Expressions))
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

	field := expectExpr[*ast.FieldAccessExpr](t, program.Expressions[4])
	if field.Name.Name != "name" {
		t.Fatalf("field name = %q, want %q", field.Name.Name, "name")
	}

	nestedField := expectExpr[*ast.FieldAccessExpr](t, program.Expressions[5])
	innerField := expectExpr[*ast.FieldAccessExpr](t, nestedField.Target)
	index := expectExpr[*ast.IndexExpr](t, innerField.Target)
	if _, ok := index.Target.(*ast.Identifier); !ok {
		t.Fatalf("nested field base target = %T, want identifier", index.Target)
	}

	if innerField.Name.Name != "profile" || nestedField.Name.Name != "name" {
		t.Fatalf("nested field chain mismatch")
	}

	call := expectExpr[*ast.CallExpr](t, program.Expressions[6])
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
	program := mustParse(t, "precedence.molt", "x = a or b and c == d + e * -f\nif cond -> left = 1 else -> right = 2\nwhile keepGoing -> step = step + 1\nfor item in xs -> total = total + item")

	if len(program.Expressions) != 4 {
		t.Fatalf("program expression count = %d, want 4", len(program.Expressions))
	}

	assign := expectExpr[*ast.AssignmentExpr](t, program.Expressions[0])
	assignTarget := expectExpr[*ast.Identifier](t, assign.Target)
	if assignTarget.Name != "x" {
		t.Fatalf("assignment target = %q, want %q", assignTarget.Name, "x")
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

	whileExpr := expectExpr[*ast.WhileExpr](t, program.Expressions[2])
	if _, ok := whileExpr.Condition.(*ast.Identifier); !ok {
		t.Fatalf("while condition = %T, want identifier", whileExpr.Condition)
	}

	if _, ok := whileExpr.Body.(*ast.AssignmentExpr); !ok {
		t.Fatalf("while body = %T, want assignment", whileExpr.Body)
	}

	forExpr := expectExpr[*ast.ForInExpr](t, program.Expressions[3])
	if forExpr.Binding.Name != "item" {
		t.Fatalf("for binding = %q, want %q", forExpr.Binding.Name, "item")
	}

	if _, ok := forExpr.Iterable.(*ast.Identifier); !ok {
		t.Fatalf("for iterable = %T, want identifier", forExpr.Iterable)
	}

	if _, ok := forExpr.Body.(*ast.AssignmentExpr); !ok {
		t.Fatalf("for body = %T, want assignment", forExpr.Body)
	}
}

func TestParseConditionalWithoutElse(t *testing.T) {
	program := mustParse(t, "if_without_else.molt", "if cond -> step = step + 1")
	if len(program.Expressions) != 1 {
		t.Fatalf("program expression count = %d, want 1", len(program.Expressions))
	}

	conditional := expectExpr[*ast.ConditionalExpr](t, program.Expressions[0])
	if conditional.ElseBranch != nil {
		t.Fatalf("else branch = %T, want nil", conditional.ElseBranch)
	}

	if _, ok := conditional.ThenBranch.(*ast.AssignmentExpr); !ok {
		t.Fatalf("then branch = %T, want assignment", conditional.ThenBranch)
	}
}

func TestParseMatchExpression(t *testing.T) {
	program := mustParse(t, "match.molt", ""+
		"match value {\n"+
		"  1 -> \"one\"\n"+
		"  answer -> answer\n"+
		"  _ -> nil\n"+
		"}",
	)

	if len(program.Expressions) != 1 {
		t.Fatalf("program expression count = %d, want 1", len(program.Expressions))
	}

	matchExpr := expectExpr[*ast.MatchExpr](t, program.Expressions[0])
	subject := expectExpr[*ast.Identifier](t, matchExpr.Subject)
	if subject.Name != "value" {
		t.Fatalf("subject = %q, want %q", subject.Name, "value")
	}

	if len(matchExpr.Cases) != 3 {
		t.Fatalf("case count = %d, want 3", len(matchExpr.Cases))
	}

	first := expectExpr[*ast.NumberLiteral](t, matchExpr.Cases[0].Pattern)
	second := expectExpr[*ast.Identifier](t, matchExpr.Cases[1].Pattern)
	third := expectExpr[*ast.Identifier](t, matchExpr.Cases[2].Pattern)
	if first.Value != 1 || second.Name != "answer" || third.Name != "_" {
		t.Fatalf("match patterns were not parsed correctly")
	}
}

func TestParseRecordFieldAssignment(t *testing.T) {
	program := mustParse(t, "field_assignment.molt", ""+
		"profile.name = \"bolt\"\n"+
		"profile.stats.runs = profile.stats.runs + 1",
	)

	if len(program.Expressions) != 2 {
		t.Fatalf("program expression count = %d, want 2", len(program.Expressions))
	}

	assign := expectExpr[*ast.AssignmentExpr](t, program.Expressions[0])
	target := expectExpr[*ast.FieldAccessExpr](t, assign.Target)
	base := expectExpr[*ast.Identifier](t, target.Target)
	if base.Name != "profile" || target.Name.Name != "name" {
		t.Fatalf("field assignment target mismatch")
	}

	nestedAssign := expectExpr[*ast.AssignmentExpr](t, program.Expressions[1])
	outerField := expectExpr[*ast.FieldAccessExpr](t, nestedAssign.Target)
	innerField := expectExpr[*ast.FieldAccessExpr](t, outerField.Target)
	root := expectExpr[*ast.Identifier](t, innerField.Target)
	if root.Name != "profile" || innerField.Name.Name != "stats" || outerField.Name.Name != "runs" {
		t.Fatalf("nested field assignment target mismatch")
	}
}

func TestParseTryCatch(t *testing.T) {
	program := mustParse(t, "try_catch.molt", "try import \"./lib.molt\" catch err -> err.message")
	if len(program.Expressions) != 1 {
		t.Fatalf("program expression count = %d, want 1", len(program.Expressions))
	}

	tryExpr := expectExpr[*ast.TryCatchExpr](t, program.Expressions[0])
	if _, ok := tryExpr.Body.(*ast.ImportExpr); !ok {
		t.Fatalf("try body = %T, want import", tryExpr.Body)
	}

	if tryExpr.CatchBinding.Name != "err" {
		t.Fatalf("catch binding = %q, want %q", tryExpr.CatchBinding.Name, "err")
	}

	field := expectExpr[*ast.FieldAccessExpr](t, tryExpr.CatchBranch)
	if field.Name.Name != "message" {
		t.Fatalf("field name = %q, want %q", field.Name.Name, "message")
	}
}

func TestParseLoopControlInsideConditionalBranches(t *testing.T) {
	program := mustParse(t, "loop_control.molt", "while true -> if false -> break else -> continue")
	if len(program.Expressions) != 1 {
		t.Fatalf("program expression count = %d, want 1", len(program.Expressions))
	}

	whileExpr := expectExpr[*ast.WhileExpr](t, program.Expressions[0])
	conditional := expectExpr[*ast.ConditionalExpr](t, whileExpr.Body)
	if _, ok := conditional.ThenBranch.(*ast.BreakExpr); !ok {
		t.Fatalf("then branch = %T, want break", conditional.ThenBranch)
	}

	if _, ok := conditional.ElseBranch.(*ast.ContinueExpr); !ok {
		t.Fatalf("else branch = %T, want continue", conditional.ElseBranch)
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

func TestParseRecordLiteral(t *testing.T) {
	program := mustParse(t, "records.molt", ""+
		"record { answer: 42, nested: record { ok: true }, items: [1, 2] }\n"+
		"record {}",
	)

	if len(program.Expressions) != 2 {
		t.Fatalf("program expression count = %d, want 2", len(program.Expressions))
	}

	record := expectExpr[*ast.RecordLiteral](t, program.Expressions[0])
	if len(record.Fields) != 3 {
		t.Fatalf("record field count = %d, want 3", len(record.Fields))
	}

	if record.Fields[0].Name.Name != "answer" {
		t.Fatalf("field 0 name = %q, want %q", record.Fields[0].Name.Name, "answer")
	}

	nested := expectExpr[*ast.RecordLiteral](t, record.Fields[1].Value)
	if len(nested.Fields) != 1 || nested.Fields[0].Name.Name != "ok" {
		t.Fatalf("nested record fields were not parsed correctly")
	}

	list := expectExpr[*ast.ListLiteral](t, record.Fields[2].Value)
	if len(list.Elements) != 2 {
		t.Fatalf("items field length = %d, want 2", len(list.Elements))
	}

	empty := expectExpr[*ast.RecordLiteral](t, program.Expressions[1])
	if len(empty.Fields) != 0 {
		t.Fatalf("empty record field count = %d, want 0", len(empty.Fields))
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
		{name: "invalid assignment target", input: "(x) = 1", message: "invalid assignment target; expected identifier or record field"},
		{name: "index assignment target", input: "xs[0] = 1", message: "invalid assignment target; expected identifier or record field"},
		{name: "chained relational", input: "a < b < c", message: "chained relational operators are not allowed"},
		{name: "chained equality", input: "a == b != c", message: "chained equality operators are not allowed"},
		{name: "else missing arrow", input: "if x -> y else z", message: "expected '->' after else"},
		{name: "try missing catch", input: "try x", message: "expected 'catch' after try body"},
		{name: "catch missing binding", input: "try x catch -> y", message: "expected identifier after 'catch'"},
		{name: "catch missing arrow", input: "try x catch err y", message: "expected '->' after catch binding"},
		{name: "while missing arrow", input: "while x y", message: "expected '->' after while condition"},
		{name: "for missing binding", input: "for 1 in xs -> x", message: "expected identifier after 'for'"},
		{name: "for missing in", input: "for item xs -> x", message: "expected 'in' after loop binding"},
		{name: "for missing arrow", input: "for item in xs x", message: "expected '->' after for iterable"},
		{name: "match missing subject brace", input: "match x 1 -> 2", message: "expected '{' after match subject"},
		{name: "match invalid pattern", input: "match x { [1] -> 2 }", message: "expected literal, identifier, or '_' in match pattern"},
		{name: "match missing arrow", input: "match x { 1 2 }", message: "expected '->' in match case"},
		{name: "same-line block sequence", input: "{ a b }", message: "expected line break or '}' after expression"},
		{name: "export missing name", input: "export 1", message: "expected identifier after 'export'"},
		{name: "import missing path", input: "import x", message: "expected string literal after 'import'"},
		{name: "record missing brace", input: "record answer: 1", message: "expected '{' after 'record'"},
		{name: "record missing field name", input: "record { 1: 2 }", message: "expected record field name"},
		{name: "record missing colon", input: "record { answer 42 }", message: "expected ':' after record field name"},
		{name: "record duplicate field", input: "record { answer: 1, answer: 2 }", message: `duplicate record field "answer"`},
		{name: "record trailing comma", input: "record { answer: 1, }", message: "expected record field after ','"},
		{name: "field access missing name", input: "value.", message: "expected field name after '.'"},
		{name: "field access newline", input: "value.\nname", message: "expected field name after '.'"},
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
