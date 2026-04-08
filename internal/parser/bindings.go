package parser

import (
	"fmt"

	"molt/internal/ast"
)

func (p *Parser) bindingPatternFromExpr(expr ast.Expr) (ast.BindingPattern, error) {
	switch node := expr.(type) {
	case *ast.Identifier:
		return node, nil
	case *ast.ListLiteral:
		elements := make([]ast.BindingPattern, 0, len(node.Elements))
		for _, element := range node.Elements {
			pattern, err := p.bindingPatternFromExpr(element)
			if err != nil {
				return nil, err
			}

			elements = append(elements, pattern)
		}

		return &ast.ListBindingPattern{
			SourceSpan: node.Span(),
			Elements:   elements,
		}, nil
	case *ast.RecordLiteral:
		fields := make([]*ast.RecordBindingField, 0, len(node.Fields))
		for _, field := range node.Fields {
			pattern, err := p.bindingPatternFromExpr(field.Value)
			if err != nil {
				return nil, err
			}

			fields = append(fields, &ast.RecordBindingField{
				SourceSpan: field.Span(),
				Name:       field.Name,
				Value:      pattern,
			})
		}

		return &ast.RecordBindingPattern{
			SourceSpan: node.Span(),
			Fields:     fields,
		}, nil
	default:
		return nil, p.errorAtSpan(expr.Span(), fmt.Sprintf("expected identifier, list destructuring, or record destructuring, found %T", expr))
	}
}

func (p *Parser) assignmentTargetFromExpr(expr ast.Expr) (ast.AssignmentTarget, error) {
	switch node := expr.(type) {
	case *ast.FieldAccessExpr:
		return node, nil
	default:
		pattern, err := p.bindingPatternFromExpr(expr)
		if err != nil {
			return nil, p.errorAtSpan(expr.Span(), "invalid assignment target; expected identifier, list destructuring, record destructuring, or record field")
		}

		return pattern.(ast.AssignmentTarget), nil
	}
}
