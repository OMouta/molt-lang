package ast

import "molt/internal/source"

type ImportExpr struct {
	SourceSpan source.Span
	Path       *StringLiteral
}

func (n *ImportExpr) Span() source.Span { return n.SourceSpan }
func (*ImportExpr) exprNode()           {}

type ExportExpr struct {
	SourceSpan source.Span
	Name       *Identifier
}

func (n *ExportExpr) Span() source.Span { return n.SourceSpan }
func (*ExportExpr) exprNode()           {}
