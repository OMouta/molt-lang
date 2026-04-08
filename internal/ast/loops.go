package ast

import "molt/internal/source"

type WhileExpr struct {
	SourceSpan source.Span
	Condition  Expr
	Body       Expr
}

func (n *WhileExpr) Span() source.Span { return n.SourceSpan }
func (*WhileExpr) exprNode()           {}

type TryCatchExpr struct {
	SourceSpan   source.Span
	Body         Expr
	CatchBinding *Identifier
	CatchBranch  Expr
}

func (n *TryCatchExpr) Span() source.Span { return n.SourceSpan }
func (*TryCatchExpr) exprNode()           {}

type MatchCase struct {
	SourceSpan source.Span
	Pattern    Expr
	Branch     Expr
}

func (n *MatchCase) Span() source.Span { return n.SourceSpan }

type MatchExpr struct {
	SourceSpan source.Span
	Subject    Expr
	Cases      []*MatchCase
}

func (n *MatchExpr) Span() source.Span { return n.SourceSpan }
func (*MatchExpr) exprNode()           {}

type ForInExpr struct {
	SourceSpan source.Span
	Binding    BindingPattern
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
