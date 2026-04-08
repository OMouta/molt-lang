package parser

import (
	"fmt"
	"strconv"

	"molt/internal/ast"
	"molt/internal/lexer"
)

func (p *Parser) parseConditional() (ast.Expr, error) {
	if p.match(lexer.Match) {
		return p.parseMatch(p.previous())
	}

	if p.match(lexer.For) {
		start := p.previous()

		bindingExpr, err := p.parseOr()
		if err != nil {
			return nil, err
		}

		binding, err := p.bindingPatternFromExpr(bindingExpr)
		if err != nil {
			return nil, p.errorAtSpan(bindingExpr.Span(), "expected identifier, list destructuring, or record destructuring after 'for'")
		}

		if _, err := p.consume(lexer.In, "expected 'in' after loop binding"); err != nil {
			return nil, err
		}

		iterable, err := p.parseAssignment()
		if err != nil {
			return nil, err
		}

		if _, err := p.consume(lexer.Arrow, "expected '->' after for iterable"); err != nil {
			return nil, err
		}

		body, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		return &ast.ForInExpr{
			SourceSpan: p.mergeSpans(start.Span, body.Span()),
			Binding:    binding,
			Iterable: iterable,
			Body:     body,
		}, nil
	}

	if p.match(lexer.While) {
		start := p.previous()

		condition, err := p.parseAssignment()
		if err != nil {
			return nil, err
		}

		if _, err := p.consume(lexer.Arrow, "expected '->' after while condition"); err != nil {
			return nil, err
		}

		body, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		return &ast.WhileExpr{
			SourceSpan: p.mergeSpans(start.Span, body.Span()),
			Condition:  condition,
			Body:       body,
		}, nil
	}

	if p.match(lexer.Try) {
		start := p.previous()

		body, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		if _, err := p.consume(lexer.Catch, "expected 'catch' after try body"); err != nil {
			return nil, err
		}

		bindingToken, err := p.consume(lexer.Identifier, "expected identifier after 'catch'")
		if err != nil {
			return nil, err
		}

		if _, err := p.consume(lexer.Arrow, "expected '->' after catch binding"); err != nil {
			return nil, err
		}

		catchBranch, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		return &ast.TryCatchExpr{
			SourceSpan: p.mergeSpans(start.Span, catchBranch.Span()),
			Body:       body,
			CatchBinding: &ast.Identifier{
				SourceSpan: bindingToken.Span,
				Name:       bindingToken.Value,
			},
			CatchBranch: catchBranch,
		}, nil
	}

	if !p.match(lexer.If) {
		return p.parseAssignment()
	}

	start := p.previous()

	condition, err := p.parseAssignment()
	if err != nil {
		return nil, err
	}

	if _, err := p.consume(lexer.Arrow, "expected '->' after if condition"); err != nil {
		return nil, err
	}

	thenBranch, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.match(lexer.Else) {
		if _, err := p.consume(lexer.Arrow, "expected '->' after else"); err != nil {
			return nil, err
		}

		elseBranch, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		return &ast.ConditionalExpr{
			SourceSpan: p.mergeSpans(start.Span, elseBranch.Span()),
			Condition:  condition,
			ThenBranch: thenBranch,
			ElseBranch: elseBranch,
		}, nil
	}

	return &ast.ConditionalExpr{
		SourceSpan: p.mergeSpans(start.Span, thenBranch.Span()),
		Condition:  condition,
		ThenBranch: thenBranch,
	}, nil
}

func (p *Parser) parseAssignment() (ast.Expr, error) {
	left, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	if !p.check(lexer.Assign) || !p.onSameLine(left.Span(), p.peek()) {
		return left, nil
	}

	p.advance()

	if p.previous().Kind != lexer.Assign {
		return left, nil
	}

	target, err := p.assignmentTargetFromExpr(left)
	if err != nil {
		return nil, err
	}

	value, err := p.parseConditional()
	if err != nil {
		return nil, err
	}

	return &ast.AssignmentExpr{
		SourceSpan: p.mergeSpans(left.Span(), value.Span()),
		Target:     target,
		Value:      value,
	}, nil
}

func (p *Parser) parseMatch(start lexer.Token) (ast.Expr, error) {
	subject, err := p.parseAssignment()
	if err != nil {
		return nil, err
	}

	if _, err := p.consume(lexer.LeftBrace, "expected '{' after match subject"); err != nil {
		return nil, err
	}

	cases := make([]*ast.MatchCase, 0, 2)
	for !p.check(lexer.RightBrace) && !p.isAtEnd() {
		matchCase, err := p.parseMatchCase()
		if err != nil {
			return nil, err
		}

		cases = append(cases, matchCase)
		if p.check(lexer.RightBrace) {
			break
		}

		if !p.startsOnLaterLine(matchCase.Span(), p.peek()) {
			return nil, p.errorAt(p.peek(), "expected line break or '}' after match case")
		}
	}

	end, err := p.consume(lexer.RightBrace, "expected '}' after match expression")
	if err != nil {
		return nil, err
	}

	return &ast.MatchExpr{
		SourceSpan: p.mergeSpans(start.Span, end.Span),
		Subject:    subject,
		Cases:      cases,
	}, nil
}

func (p *Parser) parseMatchCase() (*ast.MatchCase, error) {
	pattern, err := p.parseMatchPattern()
	if err != nil {
		return nil, err
	}

	if _, err := p.consume(lexer.Arrow, "expected '->' in match case"); err != nil {
		return nil, err
	}

	branch, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return &ast.MatchCase{
		SourceSpan: p.mergeSpans(pattern.Span(), branch.Span()),
		Pattern:    pattern,
		Branch:     branch,
	}, nil
}

func (p *Parser) parseMatchPattern() (ast.Expr, error) {
	switch {
	case p.match(lexer.Number):
		token := p.previous()
		value, err := strconv.ParseFloat(token.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid numeric token %q: %w", token.Value, err)
		}

		return &ast.NumberLiteral{SourceSpan: token.Span, Value: value}, nil
	case p.match(lexer.String):
		token := p.previous()
		return &ast.StringLiteral{SourceSpan: token.Span, Value: token.Value}, nil
	case p.match(lexer.True):
		return &ast.BooleanLiteral{SourceSpan: p.previous().Span, Value: true}, nil
	case p.match(lexer.False):
		return &ast.BooleanLiteral{SourceSpan: p.previous().Span, Value: false}, nil
	case p.match(lexer.Nil):
		return &ast.NilLiteral{SourceSpan: p.previous().Span}, nil
	case p.match(lexer.Identifier):
		token := p.previous()
		return &ast.Identifier{SourceSpan: token.Span, Name: token.Value}, nil
	default:
		return nil, p.errorAt(p.peek(), "expected literal, identifier, or '_' in match pattern")
	}
}
