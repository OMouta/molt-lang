package parser

import (
	"fmt"

	"molt/internal/ast"
	"molt/internal/diagnostic"
	"molt/internal/lexer"
	"molt/internal/source"
)

func (p *Parser) parseCommaSeparatedExpressions(stop lexer.Kind, stopLabel string) ([]ast.Expr, lexer.Token, error) {
	expressions := make([]ast.Expr, 0, 2)

	if p.check(stop) {
		end := p.advance()
		return expressions, end, nil
	}

	for {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, lexer.Token{}, err
		}

		expressions = append(expressions, expr)

		if !p.match(lexer.Comma) {
			break
		}

		if p.check(stop) {
			break // trailing comma allowed
		}
	}

	end, err := p.consume(stop, fmt.Sprintf("expected %s after list", stopLabel))
	if err != nil {
		return nil, lexer.Token{}, err
	}

	return expressions, end, nil
}

func (p *Parser) sequenceToExpr(start, end source.Span, expressions []ast.Expr) ast.Expr {
	switch len(expressions) {
	case 0:
		return &ast.BlockExpr{
			SourceSpan:  p.mergeSpans(start, end),
			Expressions: nil,
		}
	case 1:
		return expressions[0]
	default:
		return &ast.BlockExpr{
			SourceSpan:  p.mergeSpans(start, end),
			Expressions: expressions,
		}
	}
}

func (p *Parser) sequenceSpan(expressions []ast.Expr, fallback source.Span) source.Span {
	if len(expressions) == 0 {
		return fallback
	}

	return p.mergeSpans(expressions[0].Span(), expressions[len(expressions)-1].Span())
}

func (p *Parser) mergeSpans(start, end source.Span) source.Span {
	return start.File.MustSpan(start.Start.Offset, end.End.Offset)
}

func (p *Parser) onSameLine(previous source.Span, next lexer.Token) bool {
	return previous.End.Line == next.Span.Start.Line
}

func (p *Parser) startsOnLaterLine(previous source.Span, next lexer.Token) bool {
	return next.Span.Start.Line > previous.End.Line
}

func (p *Parser) inQuote() bool {
	return p.quoteDepth > 0
}

func (p *Parser) inMutationRule() bool {
	return p.mutationDepth > 0
}

func (p *Parser) inMutationPattern() bool {
	return p.mutationPatternDepth > 0
}

func (p *Parser) consume(kind lexer.Kind, message string) (lexer.Token, error) {
	if p.check(kind) {
		return p.advance(), nil
	}

	return lexer.Token{}, p.errorAt(p.peek(), message)
}

func (p *Parser) match(kinds ...lexer.Kind) bool {
	for _, kind := range kinds {
		if p.check(kind) {
			p.advance()
			return true
		}
	}

	return false
}

func (p *Parser) check(kind lexer.Kind) bool {
	return p.peek().Kind == kind
}

func (p *Parser) advance() lexer.Token {
	token := p.peek()
	if !p.isAtEnd() {
		p.current++
	}

	return token
}

func (p *Parser) previous() lexer.Token {
	return p.tokens[p.current-1]
}

func (p *Parser) peek() lexer.Token {
	if p.current >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}

	return p.tokens[p.current]
}

func (p *Parser) peekN(distance int) lexer.Token {
	index := p.current + distance
	if index >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}

	return p.tokens[index]
}

func (p *Parser) isAtEnd() bool {
	return p.peek().Kind == lexer.EOF
}

func (p *Parser) errorAt(token lexer.Token, message string) error {
	return diagnostic.NewParseError(message, token.Span)
}

func (p *Parser) errorAtSpan(span source.Span, message string) error {
	return diagnostic.NewParseError(message, span)
}

func isMutationOperator(kind lexer.Kind) bool {
	switch kind {
	case lexer.Plus,
		lexer.Minus,
		lexer.Star,
		lexer.Slash,
		lexer.Percent,
		lexer.EqualEqual,
		lexer.BangEqual,
		lexer.Less,
		lexer.LessEqual,
		lexer.Greater,
		lexer.GreaterEqual,
		lexer.And,
		lexer.Or,
		lexer.Not:
		return true
	default:
		return false
	}
}
