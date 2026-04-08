package ast

import "molt/internal/source"

type WhileExpr struct {
	SourceSpan source.Span
	Condition  Expr
	Body       Expr
}

func (n *WhileExpr) Span() source.Span { return n.SourceSpan }
func (*WhileExpr) exprNode()           {}

type ForInExpr struct {
	SourceSpan source.Span
	Binding    *Identifier
	Iterable   Expr
	Body       Expr
}

func (n *ForInExpr) Span() source.Span { return n.SourceSpan }
func (*ForInExpr) exprNode()           {}

type BreakExpr struct {
	SourceSpan source.Span
}

func (n *BreakExpr) Span() source.Span { return n.SourceSpan }
func (*BreakExpr) exprNode()           {}

type ContinueExpr struct {
	SourceSpan source.Span
}

func (n *ContinueExpr) Span() source.Span { return n.SourceSpan }
func (*ContinueExpr) exprNode()           {}
