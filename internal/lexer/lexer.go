package lexer

import (
	"fmt"
	"strings"

	"molt/internal/diagnostic"
	"molt/internal/source"
)

type Lexer struct {
	file   *source.File
	cursor cursor
}

func LexFile(file *source.File) ([]Token, error) {
	lexer := Lexer{
		file:   file,
		cursor: newCursor(file),
	}

	return lexer.Lex()
}

func Lex(path, text string) ([]Token, error) {
	return LexFile(source.NewFile(path, text))
}

func (l *Lexer) Lex() ([]Token, error) {
	tokens := make([]Token, 0, 32)

	for {
		if err := l.skipTrivia(); err != nil {
			return nil, err
		}

		if l.cursor.isAtEnd() {
			tokens = append(tokens, l.token(EOF, l.cursor.offset, ""))
			return tokens, nil
		}

		token, err := l.nextToken()
		if err != nil {
			return nil, err
		}

		tokens = append(tokens, token)
	}
}

func (l *Lexer) nextToken() (Token, error) {
	start := l.cursor.offset
	ch := l.cursor.advance()

	switch ch {
	case '(':
		return l.token(LeftParen, start, ""), nil
	case ')':
		return l.token(RightParen, start, ""), nil
	case '{':
		return l.token(LeftBrace, start, ""), nil
	case '}':
		return l.token(RightBrace, start, ""), nil
	case '[':
		return l.token(LeftBracket, start, ""), nil
	case ']':
		return l.token(RightBracket, start, ""), nil
	case ',':
		return l.token(Comma, start, ""), nil
	case ':':
		return l.token(Colon, start, ""), nil
	case '.':
		return l.token(Dot, start, ""), nil
	case '+':
		return l.token(Plus, start, ""), nil
	case '*':
		return l.token(Star, start, ""), nil
	case '/':
		return l.token(Slash, start, ""), nil
	case '%':
		return l.token(Percent, start, ""), nil
	case '-':
		if l.cursor.match('>') {
			return l.token(Arrow, start, ""), nil
		}

		return l.token(Minus, start, ""), nil
	case '=':
		if l.cursor.match('=') {
			return l.token(EqualEqual, start, ""), nil
		}

		return l.token(Assign, start, ""), nil
	case '!':
		if l.cursor.match('=') {
			return l.token(BangEqual, start, ""), nil
		}

		return Token{}, l.errorSpan("unexpected character '!'", start, l.cursor.offset)
	case '<':
		if l.cursor.match('=') {
			return l.token(LessEqual, start, ""), nil
		}

		return l.token(Less, start, ""), nil
	case '>':
		if l.cursor.match('=') {
			return l.token(GreaterEqual, start, ""), nil
		}

		return l.token(Greater, start, ""), nil
	case '@':
		if l.cursor.match('{') {
			return l.token(QuoteStart, start, ""), nil
		}

		return Token{}, l.errorSpan("expected '{' after '@'", start, l.cursor.offset)
	case '~':
		if l.cursor.match('{') {
			return l.token(MutationStart, start, ""), nil
		}

		return l.token(Tilde, start, ""), nil
	case '"':
		return l.lexString(start)
	default:
		switch {
		case isDigit(ch):
			return l.lexNumber(start)
		case isIdentifierStart(ch):
			return l.lexIdentifierOrKeyword(start)
		default:
			return Token{}, l.errorSpan(fmt.Sprintf("unexpected character %q", ch), start, l.cursor.offset)
		}
	}
}

func (l *Lexer) lexIdentifierOrKeyword(start int) (Token, error) {
	for isIdentifierContinue(l.cursor.peek()) {
		l.cursor.advance()
	}

	lexeme := l.file.Text()[start:l.cursor.offset]
	if kind, ok := LookupKeyword(lexeme); ok {
		return l.token(kind, start, ""), nil
	}

	return l.token(Identifier, start, lexeme), nil
}

func (l *Lexer) lexNumber(start int) (Token, error) {
	for isDigit(l.cursor.peek()) {
		l.cursor.advance()
	}

	if l.cursor.peek() == '.' {
		if !isDigit(l.cursor.peekN(1)) {
			l.consumeMalformedNumber()
			return Token{}, l.errorSpan(
				fmt.Sprintf("malformed number literal %q", l.file.Text()[start:l.cursor.offset]),
				start,
				l.cursor.offset,
			)
		}

		l.cursor.advance()

		for isDigit(l.cursor.peek()) {
			l.cursor.advance()
		}
	}

	if isIdentifierStart(l.cursor.peek()) || l.cursor.peek() == '.' {
		l.consumeMalformedNumber()
		return Token{}, l.errorSpan(
			fmt.Sprintf("malformed number literal %q", l.file.Text()[start:l.cursor.offset]),
			start,
			l.cursor.offset,
		)
	}

	return l.token(Number, start, l.file.Text()[start:l.cursor.offset]), nil
}

func (l *Lexer) consumeMalformedNumber() {
	for {
		ch := l.cursor.peek()
		if isIdentifierContinue(ch) || ch == '.' {
			l.cursor.advance()
			continue
		}

		return
	}
}

func (l *Lexer) lexString(start int) (Token, error) {
	var value strings.Builder

	for !l.cursor.isAtEnd() {
		ch := l.cursor.advance()

		switch ch {
		case '"':
			return l.token(String, start, value.String()), nil
		case '\n', '\r':
			return Token{}, l.errorSpan("unterminated string literal", start, l.cursor.offset-1)
		case '\\':
			escapeStart := l.cursor.offset - 1
			escaped, err := l.readEscape()
			if err != nil {
				return Token{}, l.errorSpan(err.Error(), escapeStart, l.cursor.offset)
			}

			value.WriteByte(escaped)
		default:
			value.WriteByte(ch)
		}
	}

	return Token{}, l.errorSpan("unterminated string literal", start, l.cursor.offset)
}

func (l *Lexer) readEscape() (byte, error) {
	if l.cursor.isAtEnd() {
		return 0, fmt.Errorf("unterminated string literal")
	}

	switch ch := l.cursor.advance(); ch {
	case '\\':
		return '\\', nil
	case '"':
		return '"', nil
	case 'e':
		return 0x1b, nil
	case 'n':
		return '\n', nil
	case 'r':
		return '\r', nil
	case 't':
		return '\t', nil
	default:
		return 0, fmt.Errorf("invalid escape sequence \\%c", ch)
	}
}

func (l *Lexer) skipTrivia() error {
	for !l.cursor.isAtEnd() {
		switch l.cursor.peek() {
		case ' ', '\t', '\n', '\r':
			l.cursor.advance()
		case '#':
			l.skipComment()
		default:
			return nil
		}
	}

	return nil
}

func (l *Lexer) skipComment() {
	for !l.cursor.isAtEnd() {
		if l.cursor.peek() == '\n' {
			return
		}

		l.cursor.advance()
	}
}

func (l *Lexer) token(kind Kind, start int, value string) Token {
	span := l.file.MustSpan(start, l.cursor.offset)

	return Token{
		Kind:   kind,
		Span:   span,
		Lexeme: l.file.Text()[start:l.cursor.offset],
		Value:  value,
	}
}

func (l *Lexer) errorSpan(message string, start, end int) error {
	if end < start {
		end = start
	}

	span := l.file.MustSpan(start, end)
	return diagnostic.NewParseError(message, span)
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isIdentifierStart(ch byte) bool {
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isIdentifierContinue(ch byte) bool {
	return isIdentifierStart(ch) || isDigit(ch)
}
