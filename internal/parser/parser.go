package parser

import (
	"fmt"
	"strconv"

	"molt/internal/ast"
	"molt/internal/diagnostic"
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

func (p *Parser) parseConditional() (ast.Expr, error) {
	if p.match(lexer.For) {
		start := p.previous()

		bindingToken, err := p.consume(lexer.Identifier, "expected identifier after 'for'")
		if err != nil {
			return nil, err
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
			Binding: &ast.Identifier{
				SourceSpan: bindingToken.Span,
				Name:       bindingToken.Value,
			},
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

	switch left.(type) {
	case *ast.Identifier, *ast.FieldAccessExpr:
	default:
		return nil, p.errorAt(p.previous(), "invalid assignment target; expected identifier or record field")
	}

	value, err := p.parseConditional()
	if err != nil {
		return nil, err
	}

	return &ast.AssignmentExpr{
		SourceSpan: p.mergeSpans(left.Span(), value.Span()),
		Target:     left,
		Value:      value,
	}, nil
}

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

func (p *Parser) parsePrimary() (ast.Expr, error) {
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

	expressions, err := p.parseExpressionSequence(lexer.RightBrace, "'}'")
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
			return nil, lexer.Token{}, p.errorAt(p.peek(), "expected expression after ','")
		}
	}

	end, err := p.consume(stop, fmt.Sprintf("expected %s after list", stopLabel))
	if err != nil {
		return nil, lexer.Token{}, err
	}

	return expressions, end, nil
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
