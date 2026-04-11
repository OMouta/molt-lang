package lexer

import (
	"testing"

	"molt/internal/diagnostic"
)

func TestLexProducesFinalTokenStreamWithPayloadsAndEOF(t *testing.T) {
	input := `
# file header
import "./lib.molt"
fn warp(code) = {
  xs = [1, 2.5, "hi\n\"there\""]
  if true and not false or nil == nil -> eval(code ~{ + -> * }) else -> push(xs, 3)
}
warp @{ 1 + 2 } ~ other != stuff <= more >= less < x > y
`

	tokens, err := Lex("sample.molt", input)
	if err != nil {
		t.Fatalf("Lex returned error: %v", err)
	}

	kinds := make([]Kind, len(tokens))
	values := make([]string, len(tokens))

	for i, token := range tokens {
		kinds[i] = token.Kind
		values[i] = token.Value
	}

	wantKinds := []Kind{
		Comment,
		Import, String,
		Fn, Identifier, LeftParen, Identifier, RightParen, Assign, LeftBrace,
		Identifier, Assign, LeftBracket, Number, Comma, Number, Comma, String, RightBracket,
		If, True, And, Not, False, Or, Nil, EqualEqual, Nil, Arrow,
		Identifier, LeftParen, Identifier, MutationStart, Plus, Arrow, Star, RightBrace, RightParen,
		Else, Arrow, Identifier, LeftParen, Identifier, Comma, Number, RightParen,
		RightBrace,
		Identifier, QuoteStart, Number, Plus, Number, RightBrace, Tilde, Identifier, BangEqual, Identifier, LessEqual, Identifier, GreaterEqual, Identifier, Less, Identifier, Greater, Identifier,
		EOF,
	}

	if len(kinds) != len(wantKinds) {
		t.Fatalf("token count = %d, want %d\nkinds: %#v", len(kinds), len(wantKinds), kinds)
	}

	for i := range wantKinds {
		if kinds[i] != wantKinds[i] {
			t.Fatalf("token[%d] kind = %s, want %s", i, kinds[i], wantKinds[i])
		}
	}

	checkTokenValue(t, tokens[0], "# file header")
	checkTokenValue(t, tokens[2], "./lib.molt")
	checkTokenValue(t, tokens[4], "warp")
	checkTokenValue(t, tokens[6], "code")
	checkTokenValue(t, tokens[13], "1")
	checkTokenValue(t, tokens[15], "2.5")
	checkTokenValue(t, tokens[17], "hi\n\"there\"")
	checkTokenValue(t, tokens[29], "eval")
	checkTokenValue(t, tokens[42], "xs")
	checkTokenValue(t, tokens[44], "3")
	checkTokenValue(t, tokens[47], "warp")
	checkTokenValue(t, tokens[49], "1")
	checkTokenValue(t, tokens[51], "2")
	checkTokenValue(t, tokens[54], "other")

	eof := tokens[len(tokens)-1]
	if eof.Span.Start.Offset != eof.Span.End.Offset {
		t.Fatalf("EOF token span should be empty, got [%d, %d)", eof.Span.Start.Offset, eof.Span.End.Offset)
	}
}

func TestLexMaximalMunchForOperatorsAndIntroducers(t *testing.T) {
	tokens, err := Lex("operators.molt", "@{ ~{ -> ... . - == != <= >= < > = ~ $")
	if err != nil {
		t.Fatalf("Lex returned error: %v", err)
	}

	want := []Kind{
		QuoteStart,
		MutationStart,
		Arrow,
		Ellipsis,
		Dot,
		Minus,
		EqualEqual,
		BangEqual,
		LessEqual,
		GreaterEqual,
		Less,
		Greater,
		Assign,
		Tilde,
		Dollar,
		EOF,
	}

	if len(tokens) != len(want) {
		t.Fatalf("token count = %d, want %d", len(tokens), len(want))
	}

	for i := range want {
		if tokens[i].Kind != want[i] {
			t.Fatalf("token[%d] kind = %s, want %s", i, tokens[i].Kind, want[i])
		}
	}
}

func TestLexRecognizesModuleKeywords(t *testing.T) {
	tokens, err := Lex("modules.molt", "export value\nimport \"./lib.molt\"")
	if err != nil {
		t.Fatalf("Lex returned error: %v", err)
	}

	want := []Kind{Export, Identifier, Import, String, EOF}
	if len(tokens) != len(want) {
		t.Fatalf("token count = %d, want %d", len(tokens), len(want))
	}

	for i := range want {
		if tokens[i].Kind != want[i] {
			t.Fatalf("token[%d] kind = %s, want %s", i, tokens[i].Kind, want[i])
		}
	}

	checkTokenValue(t, tokens[1], "value")
	checkTokenValue(t, tokens[3], "./lib.molt")
}

func TestLexRecognizesWhileKeyword(t *testing.T) {
	tokens, err := Lex("while.molt", "while true -> x")
	if err != nil {
		t.Fatalf("Lex returned error: %v", err)
	}

	want := []Kind{While, True, Arrow, Identifier, EOF}
	if len(tokens) != len(want) {
		t.Fatalf("token count = %d, want %d", len(tokens), len(want))
	}

	for i := range want {
		if tokens[i].Kind != want[i] {
			t.Fatalf("token[%d] kind = %s, want %s", i, tokens[i].Kind, want[i])
		}
	}
}

func TestLexRecognizesForInKeywords(t *testing.T) {
	tokens, err := Lex("for_in.molt", "for item in xs -> item")
	if err != nil {
		t.Fatalf("Lex returned error: %v", err)
	}

	want := []Kind{For, Identifier, In, Identifier, Arrow, Identifier, EOF}
	if len(tokens) != len(want) {
		t.Fatalf("token count = %d, want %d", len(tokens), len(want))
	}

	for i := range want {
		if tokens[i].Kind != want[i] {
			t.Fatalf("token[%d] kind = %s, want %s", i, tokens[i].Kind, want[i])
		}
	}

	checkTokenValue(t, tokens[1], "item")
	checkTokenValue(t, tokens[3], "xs")
	checkTokenValue(t, tokens[5], "item")
}

func TestLexRecognizesLoopControlKeywords(t *testing.T) {
	tokens, err := Lex("loop_control.molt", "break\ncontinue")
	if err != nil {
		t.Fatalf("Lex returned error: %v", err)
	}

	want := []Kind{Break, Continue, EOF}
	if len(tokens) != len(want) {
		t.Fatalf("token count = %d, want %d", len(tokens), len(want))
	}

	for i := range want {
		if tokens[i].Kind != want[i] {
			t.Fatalf("token[%d] kind = %s, want %s", i, tokens[i].Kind, want[i])
		}
	}
}

func TestLexRecognizesTryCatchKeywords(t *testing.T) {
	tokens, err := Lex("try_catch.molt", "try risky catch err -> err")
	if err != nil {
		t.Fatalf("Lex returned error: %v", err)
	}

	want := []Kind{Try, Identifier, Catch, Identifier, Arrow, Identifier, EOF}
	if len(tokens) != len(want) {
		t.Fatalf("token count = %d, want %d", len(tokens), len(want))
	}

	for i := range want {
		if tokens[i].Kind != want[i] {
			t.Fatalf("token[%d] kind = %s, want %s", i, tokens[i].Kind, want[i])
		}
	}

	checkTokenValue(t, tokens[1], "risky")
	checkTokenValue(t, tokens[3], "err")
	checkTokenValue(t, tokens[5], "err")
}

func TestLexRecognizesRecordSyntax(t *testing.T) {
	tokens, err := Lex("record.molt", `record { name: "molt", items: [1, 2] }`)
	if err != nil {
		t.Fatalf("Lex returned error: %v", err)
	}

	want := []Kind{
		Record, LeftBrace, Identifier, Colon, String, Comma, Identifier, Colon, LeftBracket, Number, Comma, Number, RightBracket, RightBrace, EOF,
	}

	if len(tokens) != len(want) {
		t.Fatalf("token count = %d, want %d", len(tokens), len(want))
	}

	for i := range want {
		if tokens[i].Kind != want[i] {
			t.Fatalf("token[%d] kind = %s, want %s", i, tokens[i].Kind, want[i])
		}
	}

	checkTokenValue(t, tokens[2], "name")
	checkTokenValue(t, tokens[4], "molt")
	checkTokenValue(t, tokens[6], "items")
}

func TestLexRecognizesMatchSyntax(t *testing.T) {
	tokens, err := Lex("match.molt", "match value { 1 -> \"one\"\nname -> name\n_ -> nil }")
	if err != nil {
		t.Fatalf("Lex returned error: %v", err)
	}

	want := []Kind{
		Match, Identifier, LeftBrace,
		Number, Arrow, String,
		Identifier, Arrow, Identifier,
		Identifier, Arrow, Nil,
		RightBrace, EOF,
	}

	if len(tokens) != len(want) {
		t.Fatalf("token count = %d, want %d", len(tokens), len(want))
	}

	for i := range want {
		if tokens[i].Kind != want[i] {
			t.Fatalf("token[%d] kind = %s, want %s", i, tokens[i].Kind, want[i])
		}
	}

	checkTokenValue(t, tokens[1], "value")
	checkTokenValue(t, tokens[3], "1")
	checkTokenValue(t, tokens[5], "one")
	checkTokenValue(t, tokens[6], "name")
	checkTokenValue(t, tokens[8], "name")
	checkTokenValue(t, tokens[9], "_")
}

func TestLexRecognizesMutationCaptureSyntax(t *testing.T) {
	tokens, err := Lex("mutation_capture.molt", "~{ (_ + 0) -> _\n[1, ...$tail, 3] -> [0, ...$tail] }")
	if err != nil {
		t.Fatalf("Lex returned error: %v", err)
	}

	want := []Kind{
		MutationStart, LeftParen, Identifier, Plus, Number, RightParen, Arrow, Identifier,
		LeftBracket, Number, Comma, Ellipsis, Dollar, Identifier, Comma, Number, RightBracket,
		Arrow, LeftBracket, Number, Comma, Ellipsis, Dollar, Identifier, RightBracket,
		RightBrace, EOF,
	}

	if len(tokens) != len(want) {
		t.Fatalf("token count = %d, want %d", len(tokens), len(want))
	}

	for i := range want {
		if tokens[i].Kind != want[i] {
			t.Fatalf("token[%d] kind = %s, want %s", i, tokens[i].Kind, want[i])
		}
	}
}

func TestLexRecognizesFieldAccessSyntax(t *testing.T) {
	tokens, err := Lex("field_access.molt", `profile.name`)
	if err != nil {
		t.Fatalf("Lex returned error: %v", err)
	}

	want := []Kind{Identifier, Dot, Identifier, EOF}
	if len(tokens) != len(want) {
		t.Fatalf("token count = %d, want %d", len(tokens), len(want))
	}

	for i := range want {
		if tokens[i].Kind != want[i] {
			t.Fatalf("token[%d] kind = %s, want %s", i, tokens[i].Kind, want[i])
		}
	}

	checkTokenValue(t, tokens[0], "profile")
	checkTokenValue(t, tokens[2], "name")
}

func TestLexRejectsMalformedNumbers(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		message string
		span    string
	}{
		{name: "identifier suffix", input: "12abc", message: `malformed number literal "12abc"`, span: "bad.molt:1:1"},
		{name: "trailing dot", input: "1.", message: `malformed number literal "1."`, span: "bad.molt:1:1"},
		{name: "extra dot", input: "1.2.3", message: `malformed number literal "1.2.3"`, span: "bad.molt:1:1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Lex("bad.molt", tc.input)
			parseErr := expectParseError(t, err)

			if got := parseErr.Diagnostic().Message; got != tc.message {
				t.Fatalf("message = %q, want %q", got, tc.message)
			}

			if got := parseErr.Diagnostic().Location(); got != tc.span {
				t.Fatalf("location = %q, want %q", got, tc.span)
			}
		})
	}
}

func TestLexRejectsMalformedStrings(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		message string
		line    int
		column  int
	}{
		{name: "unterminated eof", input: "\"oops", message: "unterminated string literal", line: 1, column: 1},
		{name: "unterminated newline", input: "\"oops\nx", message: "unterminated string literal", line: 1, column: 1},
		{name: "invalid escape", input: "\"\\x\"", message: "invalid escape sequence \\x", line: 1, column: 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Lex("string.molt", tc.input)
			parseErr := expectParseError(t, err)
			diag := parseErr.Diagnostic()

			if diag.Message != tc.message {
				t.Fatalf("message = %q, want %q", diag.Message, tc.message)
			}

			if diag.Span.Start.Line != tc.line || diag.Span.Start.Column != tc.column {
				t.Fatalf(
					"start = %d:%d, want %d:%d",
					diag.Span.Start.Line,
					diag.Span.Start.Column,
					tc.line,
					tc.column,
				)
			}
		})
	}
}

func TestLexDecodesEscapeSequencesIncludingEscapeByte(t *testing.T) {
	tokens, err := Lex("escapes.molt", `"a\eb\n"`)
	if err != nil {
		t.Fatalf("Lex returned error: %v", err)
	}

	if len(tokens) != 2 {
		t.Fatalf("token count = %d, want 2", len(tokens))
	}

	if tokens[0].Kind != String {
		t.Fatalf("token[0] kind = %s, want %s", tokens[0].Kind, String)
	}

	checkTokenValue(t, tokens[0], "a\x1bb\n")
}

func TestLexRejectsUnexpectedCharacters(t *testing.T) {
	tests := []struct {
		input   string
		message string
	}{
		{input: "@", message: "expected '{' after '@'"},
		{input: "!", message: "unexpected character '!'"},
		{input: "..", message: "unexpected character '.'"},
	}

	for _, tc := range tests {
		_, err := Lex("bad.molt", tc.input)
		parseErr := expectParseError(t, err)

		if got := parseErr.Diagnostic().Message; got != tc.message {
			t.Fatalf("input %q message = %q, want %q", tc.input, got, tc.message)
		}
	}
}

func checkTokenValue(t *testing.T, token Token, want string) {
	t.Helper()

	if token.Value != want {
		t.Fatalf("token %s value = %q, want %q", token.Kind, token.Value, want)
	}
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
