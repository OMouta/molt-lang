package parser

import (
	"fmt"
	"strconv"

	"molt/internal/ast"
	"molt/internal/lexer"
)

func (p *Parser) parsePrimary() (ast.Expr, error) {
	switch {
	case p.check(lexer.Tilde) && p.peekN(1).Kind == lexer.LeftParen:
		if !p.inQuote() {
			return nil, p.errorAt(p.peek(), "unquote is only valid inside quotes")
		}

		return p.parseUnquote()
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
	case p.match(lexer.Break):
		return &ast.BreakExpr{SourceSpan: p.previous().Span}, nil
	case p.match(lexer.Continue):
		return &ast.ContinueExpr{SourceSpan: p.previous().Span}, nil
	case p.match(lexer.Identifier):
		token := p.previous()
		return &ast.Identifier{SourceSpan: token.Span, Name: token.Value}, nil
	case p.match(lexer.Export):
		return p.parseExport(p.previous())
	case p.match(lexer.Import):
		return p.parseImport(p.previous())
	case p.match(lexer.Record):
		return p.parseRecordLiteral(p.previous())
	case p.match(lexer.LeftParen):
		return p.parseParenthesized()
	case p.match(lexer.LeftBrace):
		return p.parseBlockFromStart(p.previous())
	case p.match(lexer.Fn):
		return p.parseFunction()
	case p.check(lexer.QuoteStart):
		return p.parseQuote()
	case p.check(lexer.MutationStart):
		return p.parseMutationLiteral()
	case p.match(lexer.LeftBracket):
		return p.parseListLiteral(p.previous())
	default:
		return nil, p.errorAt(p.peek(), fmt.Sprintf("expected expression, found %s", p.peek().Kind))
	}
}

func (p *Parser) parseExport(start lexer.Token) (ast.Expr, error) {
	nameToken, err := p.consume(lexer.Identifier, "expected identifier after 'export'")
	if err != nil {
		return nil, err
	}

	name := &ast.Identifier{
		SourceSpan: nameToken.Span,
		Name:       nameToken.Value,
	}

	return &ast.ExportExpr{
		SourceSpan: p.mergeSpans(start.Span, nameToken.Span),
		Name:       name,
	}, nil
}

func (p *Parser) parseImport(start lexer.Token) (ast.Expr, error) {
	pathToken, err := p.consume(lexer.String, "expected string literal after 'import'")
	if err != nil {
		return nil, err
	}

	path := &ast.StringLiteral{
		SourceSpan: pathToken.Span,
		Value:      pathToken.Value,
	}

	return &ast.ImportExpr{
		SourceSpan: p.mergeSpans(start.Span, pathToken.Span),
		Path:       path,
	}, nil
}

func (p *Parser) parseParenthesized() (ast.Expr, error) {
	start := p.previous()

	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	end, err := p.consume(lexer.RightParen, "expected ')' after expression")
	if err != nil {
		return nil, err
	}

	return &ast.GroupExpr{
		SourceSpan: p.mergeSpans(start.Span, end.Span),
		Inner:      expr,
	}, nil
}

func (p *Parser) parseBlockFromStart(start lexer.Token) (ast.Expr, error) {
	expressions, err := p.parseExpressionSequence(lexer.RightBrace, "'}'")
	if err != nil {
		return nil, err
	}

	end, err := p.consume(lexer.RightBrace, "expected '}' after block")
	if err != nil {
		return nil, err
	}

	return &ast.BlockExpr{
		SourceSpan:  p.mergeSpans(start.Span, end.Span),
		Expressions: expressions,
	}, nil
}

func (p *Parser) parseListLiteral(start lexer.Token) (ast.Expr, error) {
	elements, end, err := p.parseCommaSeparatedExpressions(lexer.RightBracket, "']'")
	if err != nil {
		return nil, err
	}

	return &ast.ListLiteral{
		SourceSpan: p.mergeSpans(start.Span, end.Span),
		Elements:   elements,
	}, nil
}

func (p *Parser) parseRecordLiteral(start lexer.Token) (ast.Expr, error) {
	if _, err := p.consume(lexer.LeftBrace, "expected '{' after 'record'"); err != nil {
		return nil, err
	}

	fields := make([]*ast.RecordField, 0, 2)
	seen := make(map[string]struct{})

	if p.check(lexer.RightBrace) {
		end := p.advance()
		return &ast.RecordLiteral{
			SourceSpan: p.mergeSpans(start.Span, end.Span),
			Fields:     fields,
		}, nil
	}

	for {
		nameToken, err := p.consume(lexer.Identifier, "expected record field name")
		if err != nil {
			return nil, err
		}

		if _, exists := seen[nameToken.Value]; exists {
			return nil, p.errorAt(nameToken, fmt.Sprintf("duplicate record field %q", nameToken.Value))
		}
		seen[nameToken.Value] = struct{}{}

		if _, err := p.consume(lexer.Colon, "expected ':' after record field name"); err != nil {
			return nil, err
		}

		value, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		name := &ast.Identifier{
			SourceSpan: nameToken.Span,
			Name:       nameToken.Value,
		}

		fields = append(fields, &ast.RecordField{
			SourceSpan: p.mergeSpans(nameToken.Span, value.Span()),
			Name:       name,
			Value:      value,
		})

		if !p.match(lexer.Comma) {
			break
		}

		if p.check(lexer.RightBrace) {
			return nil, p.errorAt(p.peek(), "expected record field after ','")
		}
	}

	end, err := p.consume(lexer.RightBrace, "expected '}' after record literal")
	if err != nil {
		return nil, err
	}

	return &ast.RecordLiteral{
		SourceSpan: p.mergeSpans(start.Span, end.Span),
		Fields:     fields,
	}, nil
}

func (p *Parser) finishCall(callee ast.Expr) (ast.Expr, error) {
	if _, err := p.consume(lexer.LeftParen, "expected '(' to start call"); err != nil {
		return nil, err
	}

	arguments, end, err := p.parseCommaSeparatedExpressions(lexer.RightParen, "')'")
	if err != nil {
		return nil, err
	}

	return &ast.CallExpr{
		SourceSpan: p.mergeSpans(callee.Span(), end.Span),
		Callee:     callee,
		Arguments:  arguments,
	}, nil
}

func (p *Parser) finishIndex(target ast.Expr) (ast.Expr, error) {
	if _, err := p.consume(lexer.LeftBracket, "expected '[' to start index expression"); err != nil {
		return nil, err
	}

	index, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	end, err := p.consume(lexer.RightBracket, "expected ']' after index expression")
	if err != nil {
		return nil, err
	}

	return &ast.IndexExpr{
		SourceSpan: p.mergeSpans(target.Span(), end.Span),
		Target:     target,
		Index:      index,
	}, nil
}

func (p *Parser) finishFieldAccess(target ast.Expr) (ast.Expr, error) {
	dot, err := p.consume(lexer.Dot, "expected '.' to start field access")
	if err != nil {
		return nil, err
	}

	if !p.onSameLine(dot.Span, p.peek()) {
		return nil, p.errorAt(p.peek(), "expected field name after '.'")
	}

	nameToken, err := p.consume(lexer.Identifier, "expected field name after '.'")
	if err != nil {
		return nil, err
	}

	name := &ast.Identifier{
		SourceSpan: nameToken.Span,
		Name:       nameToken.Value,
	}

	return &ast.FieldAccessExpr{
		SourceSpan: p.mergeSpans(target.Span(), nameToken.Span),
		Target:     target,
		Name:       name,
	}, nil
}

func (p *Parser) parseFunction() (ast.Expr, error) {
	start := p.previous()

	if p.check(lexer.Identifier) {
		nameToken := p.advance()
		name := &ast.Identifier{SourceSpan: nameToken.Span, Name: nameToken.Value}

		if !p.check(lexer.LeftParen) {
			if _, err := p.consume(lexer.Assign, "expected '=' after function name"); err != nil {
				return nil, err
			}

			value, err := p.parseExpression()
			if err != nil {
				return nil, err
			}

			return &ast.AssignmentExpr{
				SourceSpan: p.mergeSpans(start.Span, value.Span()),
				Target:     name,
				Value:      value,
			}, nil
		}

		parameters, err := p.parseParameters()
		if err != nil {
			return nil, err
		}

		if _, err := p.consume(lexer.Assign, "expected '=' after function signature"); err != nil {
			return nil, err
		}

		body, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		return &ast.NamedFunctionExpr{
			SourceSpan: p.mergeSpans(start.Span, body.Span()),
			Name:       name,
			Parameters: parameters,
			Body:       body,
		}, nil
	}

	parameters, err := p.parseParameters()
	if err != nil {
		return nil, err
	}

	if _, err := p.consume(lexer.Assign, "expected '=' after function signature"); err != nil {
		return nil, err
	}

	body, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return &ast.FunctionLiteralExpr{
		SourceSpan: p.mergeSpans(start.Span, body.Span()),
		Parameters: parameters,
		Body:       body,
	}, nil
}

func (p *Parser) parseParameters() ([]*ast.Identifier, error) {
	if _, err := p.consume(lexer.LeftParen, "expected '(' after 'fn'"); err != nil {
		return nil, err
	}

	parameters := make([]*ast.Identifier, 0, 2)

	if p.check(lexer.RightParen) {
		p.advance()
		return parameters, nil
	}

	for {
		token, err := p.consume(lexer.Identifier, "expected parameter name")
		if err != nil {
			return nil, err
		}

		parameters = append(parameters, &ast.Identifier{
			SourceSpan: token.Span,
			Name:       token.Value,
		})

		if !p.match(lexer.Comma) {
			break
		}

		if p.check(lexer.RightParen) {
			return nil, p.errorAt(p.peek(), "expected parameter name after ','")
		}
	}

	if _, err := p.consume(lexer.RightParen, "expected ')' after parameter list"); err != nil {
		return nil, err
	}

	return parameters, nil
}

func (p *Parser) parseQuote() (ast.Expr, error) {
	start, err := p.consume(lexer.QuoteStart, "expected '@{' to start quote")
	if err != nil {
		return nil, err
	}

	p.quoteDepth++
	expressions, err := p.parseExpressionSequence(lexer.RightBrace, "'}'")
	p.quoteDepth--
	if err != nil {
		return nil, err
	}

	end, err := p.consume(lexer.RightBrace, "expected '}' after quote")
	if err != nil {
		return nil, err
	}

	body := p.sequenceToExpr(start.Span, end.Span, expressions)

	return &ast.QuoteExpr{
		SourceSpan: p.mergeSpans(start.Span, end.Span),
		Body:       body,
	}, nil
}

func (p *Parser) parseUnquote() (ast.Expr, error) {
	tilde, err := p.consume(lexer.Tilde, "expected '~' to start unquote")
	if err != nil {
		return nil, err
	}

	if _, err := p.consume(lexer.LeftParen, "expected '(' after '~' in unquote"); err != nil {
		return nil, err
	}

	quoteDepth := p.quoteDepth
	p.quoteDepth = 0
	expr, err := p.parseExpression()
	p.quoteDepth = quoteDepth
	if err != nil {
		return nil, err
	}

	end, err := p.consume(lexer.RightParen, "expected ')' after unquote")
	if err != nil {
		return nil, err
	}

	return &ast.UnquoteExpr{
		SourceSpan: p.mergeSpans(tilde.Span, end.Span),
		Expression: expr,
	}, nil
}

func (p *Parser) parseMutationLiteral() (ast.Expr, error) {
	start, err := p.consume(lexer.MutationStart, "expected '~{' to start mutation")
	if err != nil {
		return nil, err
	}

	rules := make([]*ast.MutationRule, 0, 2)

	for !p.check(lexer.RightBrace) && !p.isAtEnd() {
		rule, err := p.parseMutationRule()
		if err != nil {
			return nil, err
		}

		rules = append(rules, rule)

		if p.check(lexer.RightBrace) {
			break
		}

		if !p.startsOnLaterLine(rule.Span(), p.peek()) {
			return nil, p.errorAt(p.peek(), "expected line break or '}' after mutation rule")
		}
	}

	end, err := p.consume(lexer.RightBrace, "expected '}' after mutation literal")
	if err != nil {
		return nil, err
	}

	return &ast.MutationLiteralExpr{
		SourceSpan: p.mergeSpans(start.Span, end.Span),
		Rules:      rules,
	}, nil
}

func (p *Parser) parseMutationRule() (*ast.MutationRule, error) {
	pattern, err := p.parseMutationSide(true)
	if err != nil {
		return nil, err
	}

	if _, err := p.consume(lexer.Arrow, "expected '->' in mutation rule"); err != nil {
		return nil, err
	}

	replacement, err := p.parseMutationSide(false)
	if err != nil {
		return nil, err
	}

	return &ast.MutationRule{
		SourceSpan:  p.mergeSpans(pattern.Span(), replacement.Span()),
		Pattern:     pattern,
		Replacement: replacement,
	}, nil
}

func (p *Parser) parseMutationSide(pattern bool) (ast.Expr, error) {
	if p.canParseBareOperator(pattern) {
		token := p.advance()
		return &ast.OperatorLiteral{
			SourceSpan: token.Span,
			Symbol:     string(token.Kind),
		}, nil
	}

	return p.parseExpression()
}

func (p *Parser) canParseBareOperator(pattern bool) bool {
	if !isMutationOperator(p.peek().Kind) {
		return false
	}

	next := p.peekN(1)
	if pattern {
		return next.Kind == lexer.Arrow
	}

	if next.Kind == lexer.RightBrace || next.Kind == lexer.EOF {
		return true
	}

	return next.Span.Start.Line > p.peek().Span.End.Line
}
