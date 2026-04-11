package ast

import "molt/internal/source"

type CommentExpr struct {
	SourceSpan source.Span
	Text       string
}

func (n *CommentExpr) Span() source.Span { return n.SourceSpan }
func (*CommentExpr) exprNode()           {}
