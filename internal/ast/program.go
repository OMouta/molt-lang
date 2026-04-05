package ast

import "molt/internal/source"

type Program struct {
	SourceSpan  source.Span
	Expressions []Expr
}

func (p *Program) Span() source.Span { return p.SourceSpan }
