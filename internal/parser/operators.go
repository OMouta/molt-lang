package parser

import (
	"molt/internal/ast"
	"molt/internal/lexer"
)

func (p *Parser) parseOr() (ast.Expr, error) {
	return p.parseLeftAssociativeBinary(
		p.parseAnd,
		map[lexer.Kind]ast.BinaryOperator{
			lexer.Or: ast.BinaryOr,
		},
	)
}

func (p *Parser) parseAnd() (ast.Expr, error) {
	return p.parseLeftAssociativeBinary(
		p.parseEquality,
		map[lexer.Kind]ast.BinaryOperator{
			lexer.And: ast.BinaryAnd,
		},
	)
}

func (p *Parser) parseEquality() (ast.Expr, error) {
	return p.parseNonAssociativeBinary(
		p.parseRelational,
		map[lexer.Kind]ast.BinaryOperator{
			lexer.EqualEqual: ast.BinaryEqual,
			lexer.BangEqual:  ast.BinaryNotEqual,
		},
		"chained equality operators are not allowed",
	)
}

func (p *Parser) parseRelational() (ast.Expr, error) {
	return p.parseNonAssociativeBinary(
		p.parseAdditive,
		map[lexer.Kind]ast.BinaryOperator{
			lexer.Less:         ast.BinaryLess,
			lexer.LessEqual:    ast.BinaryLessEqual,
			lexer.Greater:      ast.BinaryGreater,
			lexer.GreaterEqual: ast.BinaryGreaterEqual,
		},
		"chained relational operators are not allowed",
	)
}

func (p *Parser) parseAdditive() (ast.Expr, error) {
	return p.parseLeftAssociativeBinary(
		p.parseMultiplicative,
		map[lexer.Kind]ast.BinaryOperator{
			lexer.Plus:  ast.BinaryAdd,
			lexer.Minus: ast.BinarySubtract,
		},
	)
}

func (p *Parser) parseMultiplicative() (ast.Expr, error) {
	return p.parseLeftAssociativeBinary(
		p.parseUnary,
		map[lexer.Kind]ast.BinaryOperator{
			lexer.Star:    ast.BinaryMultiply,
			lexer.Slash:   ast.BinaryDivide,
			lexer.Percent: ast.BinaryModulo,
		},
	)
}

func (p *Parser) parseUnary() (ast.Expr, error) {
	if p.match(lexer.Not) {
		operator := p.previous()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}

		return &ast.UnaryExpr{
			SourceSpan: p.mergeSpans(operator.Span, operand.Span()),
			Operator:   ast.UnaryNot,
			Operand:    operand,
		}, nil
	}

	if p.match(lexer.Minus) {
		operator := p.previous()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}

		return &ast.UnaryExpr{
			SourceSpan: p.mergeSpans(operator.Span, operand.Span()),
			Operator:   ast.UnaryNegate,
			Operand:    operand,
		}, nil
	}

	return p.parsePostfix()
}

func (p *Parser) parsePostfix() (ast.Expr, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	return p.finishPostfixChain(expr, true)
}

func (p *Parser) parseMutationOperand() (ast.Expr, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	return p.finishPostfixChain(expr, false)
}

func (p *Parser) finishPostfixChain(expr ast.Expr, allowMutationApply bool) (ast.Expr, error) {
	var err error

	for {
		if p.check(lexer.LeftParen) && p.onSameLine(expr.Span(), p.peek()) {
			expr, err = p.finishCall(expr)
			if err != nil {
				return nil, err
			}

			continue
		}

		if p.check(lexer.LeftBracket) && p.onSameLine(expr.Span(), p.peek()) {
			expr, err = p.finishIndex(expr)
			if err != nil {
				return nil, err
			}

			continue
		}

		if p.check(lexer.Dot) && p.onSameLine(expr.Span(), p.peek()) {
			expr, err = p.finishFieldAccess(expr)
			if err != nil {
				return nil, err
			}

			continue
		}

		if allowMutationApply && p.check(lexer.MutationStart) && p.onSameLine(expr.Span(), p.peek()) {
			mutation, err := p.parseMutationLiteral()
			if err != nil {
				return nil, err
			}

			expr = &ast.ApplyMutationExpr{
				SourceSpan: p.mergeSpans(expr.Span(), mutation.Span()),
				Target:     expr,
				Mutation:   mutation,
			}

			continue
		}

		if allowMutationApply && p.check(lexer.Tilde) && p.onSameLine(expr.Span(), p.peek()) {
			tilde, err := p.consume(lexer.Tilde, "expected '~' to start mutation application")
			if err != nil {
				return nil, err
			}

			if !p.onSameLine(tilde.Span, p.peek()) {
				return nil, p.errorAt(p.peek(), "expected mutation after '~'")
			}

			mutation, err := p.parseMutationOperand()
			if err != nil {
				return nil, err
			}

			expr = &ast.ApplyMutationExpr{
				SourceSpan: p.mergeSpans(expr.Span(), mutation.Span()),
				Target:     expr,
				Mutation:   mutation,
			}

			continue
		}

		if p.check(lexer.QuoteStart) && p.onSameLine(expr.Span(), p.peek()) {
			quote, err := p.parseQuote()
			if err != nil {
				return nil, err
			}

			expr = &ast.CallExpr{
				SourceSpan: p.mergeSpans(expr.Span(), quote.Span()),
				Callee:     expr,
				Arguments:  []ast.Expr{quote},
			}

			continue
		}

		return expr, nil
	}
}

func (p *Parser) parseLeftAssociativeBinary(
	parseOperand func() (ast.Expr, error),
	operators map[lexer.Kind]ast.BinaryOperator,
) (ast.Expr, error) {
	left, err := parseOperand()
	if err != nil {
		return nil, err
	}

	for {
		operator, ok := operators[p.peek().Kind]
		if !ok || !p.onSameLine(left.Span(), p.peek()) {
			return left, nil
		}

		p.advance()

		right, err := parseOperand()
		if err != nil {
			return nil, err
		}

		left = &ast.BinaryExpr{
			SourceSpan: p.mergeSpans(left.Span(), right.Span()),
			Left:       left,
			Operator:   operator,
			Right:      right,
		}
	}
}

func (p *Parser) parseNonAssociativeBinary(
	parseOperand func() (ast.Expr, error),
	operators map[lexer.Kind]ast.BinaryOperator,
	message string,
) (ast.Expr, error) {
	left, err := parseOperand()
	if err != nil {
		return nil, err
	}

	operator, ok := operators[p.peek().Kind]
	if !ok || !p.onSameLine(left.Span(), p.peek()) {
		return left, nil
	}

	p.advance()

	right, err := parseOperand()
	if err != nil {
		return nil, err
	}

	if _, chained := operators[p.peek().Kind]; chained {
		return nil, p.errorAt(p.peek(), message)
	}

	return &ast.BinaryExpr{
		SourceSpan: p.mergeSpans(left.Span(), right.Span()),
		Left:       left,
		Operator:   operator,
		Right:      right,
	}, nil
}
