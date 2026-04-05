package ast

import "molt/internal/source"

type CallExpr struct {
	SourceSpan source.Span
	Callee     Expr
	Arguments  []Expr
}

func (n *CallExpr) Span() source.Span { return n.SourceSpan }
func (*CallExpr) exprNode()           {}

type NamedFunctionExpr struct {
	SourceSpan source.Span
	Name       *Identifier
	Parameters []*Identifier
	Body       Expr
}

func (n *NamedFunctionExpr) Span() source.Span { return n.SourceSpan }
func (*NamedFunctionExpr) exprNode()           {}

type FunctionLiteralExpr struct {
	SourceSpan source.Span
	Parameters []*Identifier
	Body       Expr
}

func (n *FunctionLiteralExpr) Span() source.Span { return n.SourceSpan }
func (*FunctionLiteralExpr) exprNode()           {}
