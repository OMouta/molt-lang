package parser

import (
	"fmt"

	"molt/internal/ast"
	"molt/internal/lexer"
	"molt/internal/source"
)

type Parser struct {
	tokens  []lexer.Token
	current int
}

func Parse(path, text string) (*ast.Program, error) {
	tokens, err := lexer.Lex(path, text)
	if err != nil {
		return nil, err
	}

	return ParseTokens(tokens)
}

func ParseFile(file *source.File) (*ast.Program, error) {
	tokens, err := lexer.LexFile(file)
	if err != nil {
		return nil, err
	}

	return ParseTokens(tokens)
}

func ParseTokens(tokens []lexer.Token) (*ast.Program, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("parser requires at least one token")
	}

	parser := &Parser{tokens: tokens}
	return parser.parseProgram()
}

func (p *Parser) parseProgram() (*ast.Program, error) {
	expressions, err := p.parseExpressionSequence(lexer.EOF, "end of file")
	if err != nil {
		return nil, err
	}

	eof, err := p.consume(lexer.EOF, "expected end of file")
	if err != nil {
		return nil, err
	}

	return &ast.Program{
		SourceSpan:  p.sequenceSpan(expressions, eof.Span),
		Expressions: expressions,
	}, nil
}

func (p *Parser) parseExpressionSequence(stop lexer.Kind, stopLabel string) ([]ast.Expr, error) {
	expressions := make([]ast.Expr, 0, 4)

	for !p.check(stop) && !p.isAtEnd() {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		expressions = append(expressions, expr)

		if p.check(stop) {
			break
		}

		if !p.startsOnLaterLine(expr.Span(), p.peek()) {
			return nil, p.errorAt(
				p.peek(),
				fmt.Sprintf("expected line break or %s after expression", stopLabel),
			)
		}
	}

	return expressions, nil
}

func (p *Parser) parseExpression() (ast.Expr, error) {
	return p.parseConditional()
}
