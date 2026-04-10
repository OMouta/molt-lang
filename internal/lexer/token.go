package lexer

import (
	"fmt"

	"molt/internal/source"
)

type Kind string

const (
	EOF Kind = "eof"

	Identifier Kind = "identifier"
	Number     Kind = "number"
	String     Kind = "string"

	Fn       Kind = "fn"
	If       Kind = "if"
	Else     Kind = "else"
	Match    Kind = "match"
	Try      Kind = "try"
	Catch    Kind = "catch"
	While    Kind = "while"
	For      Kind = "for"
	In       Kind = "in"
	Break    Kind = "break"
	Continue Kind = "continue"
	Export   Kind = "export"
	Import   Kind = "import"
	Record   Kind = "record"
	And      Kind = "and"
	Or       Kind = "or"
	Not      Kind = "not"
	True     Kind = "true"
	False    Kind = "false"
	Nil      Kind = "nil"

	LeftParen    Kind = "("
	RightParen   Kind = ")"
	LeftBrace    Kind = "{"
	RightBrace   Kind = "}"
	LeftBracket  Kind = "["
	RightBracket Kind = "]"
	Comma        Kind = ","
	Colon        Kind = ":"
	Dot          Kind = "."
	Ellipsis     Kind = "..."
	Dollar       Kind = "$"

	Assign        Kind = "="
	Arrow         Kind = "->"
	QuoteStart    Kind = "@{"
	MutationStart Kind = "~{"
	Tilde         Kind = "~"

	Plus         Kind = "+"
	Minus        Kind = "-"
	Star         Kind = "*"
	Slash        Kind = "/"
	Percent      Kind = "%"
	EqualEqual   Kind = "=="
	BangEqual    Kind = "!="
	Less         Kind = "<"
	LessEqual    Kind = "<="
	Greater      Kind = ">"
	GreaterEqual Kind = ">="
)

type Token struct {
	Kind   Kind
	Span   source.Span
	Lexeme string
	Value  string
}

func (t Token) String() string {
	if t.Value == "" {
		return fmt.Sprintf("%s@%s", t.Kind, t.Span.Start)
	}

	return fmt.Sprintf("%s(%q)@%s", t.Kind, t.Value, t.Span.Start)
}

var keywords = map[string]Kind{
	"fn":       Fn,
	"if":       If,
	"else":     Else,
	"match":    Match,
	"try":      Try,
	"catch":    Catch,
	"while":    While,
	"for":      For,
	"in":       In,
	"break":    Break,
	"continue": Continue,
	"export":   Export,
	"import":   Import,
	"record":   Record,
	"and":      And,
	"or":       Or,
	"not":      Not,
	"true":     True,
	"false":    False,
	"nil":      Nil,
}

func LookupKeyword(text string) (Kind, bool) {
	kind, ok := keywords[text]
	return kind, ok
}
