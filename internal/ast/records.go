package ast

import "molt/internal/source"

type RecordField struct {
	SourceSpan source.Span
	Name       *Identifier
	Value      Expr
}

func (n *RecordField) Span() source.Span { return n.SourceSpan }

type RecordLiteral struct {
	SourceSpan source.Span
	Fields     []*RecordField
}

func (n *RecordLiteral) Span() source.Span { return n.SourceSpan }
func (*RecordLiteral) exprNode()           {}

type FieldAccessExpr struct {
	SourceSpan source.Span
	Target     Expr
	Name       *Identifier
}

func (n *FieldAccessExpr) Span() source.Span { return n.SourceSpan }
func (*FieldAccessExpr) exprNode()           {}
