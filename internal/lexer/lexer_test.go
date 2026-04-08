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

	checkTokenValue(t, tokens[1], "./lib.molt")
	checkTokenValue(t, tokens[3], "warp")
	checkTokenValue(t, tokens[5], "code")
	checkTokenValue(t, tokens[12], "1")
	checkTokenValue(t, tokens[14], "2.5")
	checkTokenValue(t, tokens[16], "hi\n\"there\"")
	checkTokenValue(t, tokens[28], "eval")
	checkTokenValue(t, tokens[41], "xs")
	checkTokenValue(t, tokens[43], "3")
	checkTokenValue(t, tokens[46], "warp")
	checkTokenValue(t, tokens[48], "1")
	checkTokenValue(t, tokens[50], "2")
	checkTokenValue(t, tokens[53], "other")

	eof := tokens[len(tokens)-1]
	if eof.Span.Start.Offset != eof.Span.End.Offset {
		t.Fatalf("EOF token span should be empty, got [%d, %d)", eof.Span.Start.Offset, eof.Span.End.Offset)
	}
}

func TestLexMaximalMunchForOperatorsAndIntroducers(t *testing.T) {
	tokens, err := Lex("operators.molt", "@{ ~{ -> . - == != <= >= < > = ~")
	if err != nil {
		t.Fatalf("Lex returned error: %v", err)
	}

	want := []Kind{
		QuoteStart,
		MutationStart,
		Arrow,
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
		{input: "$", message: "unexpected character '$'"},
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
