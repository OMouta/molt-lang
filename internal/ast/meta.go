package ast

import "molt/internal/source"

type QuoteExpr struct {
	SourceSpan source.Span
	Body       Expr
}

func (n *QuoteExpr) Span() source.Span { return n.SourceSpan }
func (*QuoteExpr) exprNode()           {}

type UnquoteExpr struct {
	SourceSpan source.Span
	Expression Expr
}

func (n *UnquoteExpr) Span() source.Span { return n.SourceSpan }
func (*UnquoteExpr) exprNode()           {}

type MutationLiteralExpr struct {
	SourceSpan source.Span
	Rules      []*MutationRule
}

func (n *MutationLiteralExpr) Span() source.Span { return n.SourceSpan }
func (*MutationLiteralExpr) exprNode()           {}

type MutationRule struct {
	SourceSpan  source.Span
	Pattern     Expr
	Replacement Expr
}

func (n *MutationRule) Span() source.Span { return n.SourceSpan }

type ApplyMutationExpr struct {
	SourceSpan source.Span
	Target     Expr
	Mutation   Expr
}

func (n *ApplyMutationExpr) Span() source.Span { return n.SourceSpan }
func (*ApplyMutationExpr) exprNode()           {}
