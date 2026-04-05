package ast

import "molt/internal/source"

// Node is implemented by every concrete AST node in the tree.
type Node interface {
	Span() source.Span
}

// Expr is implemented by every AST node that can appear in expression position.
type Expr interface {
	Node
	exprNode()
}

type UnaryOperator string

const (
	UnaryNegate UnaryOperator = "-"
	UnaryNot    UnaryOperator = "not"
)

type BinaryOperator string

const (
	BinaryAdd          BinaryOperator = "+"
	BinarySubtract     BinaryOperator = "-"
	BinaryMultiply     BinaryOperator = "*"
	BinaryDivide       BinaryOperator = "/"
	BinaryModulo       BinaryOperator = "%"
	BinaryEqual        BinaryOperator = "=="
	BinaryNotEqual     BinaryOperator = "!="
	BinaryLess         BinaryOperator = "<"
	BinaryLessEqual    BinaryOperator = "<="
	BinaryGreater      BinaryOperator = ">"
	BinaryGreaterEqual BinaryOperator = ">="
	BinaryAnd          BinaryOperator = "and"
	BinaryOr           BinaryOperator = "or"
)

type OperatorLiteral struct {
	SourceSpan source.Span
	Symbol     string
}

func (n *OperatorLiteral) Span() source.Span { return n.SourceSpan }
func (*OperatorLiteral) exprNode()           {}

type NumberLiteral struct {
	SourceSpan source.Span
	Value      float64
}

func (n *NumberLiteral) Span() source.Span { return n.SourceSpan }
func (*NumberLiteral) exprNode()           {}

type StringLiteral struct {
	SourceSpan source.Span
	Value      string
}

func (n *StringLiteral) Span() source.Span { return n.SourceSpan }
func (*StringLiteral) exprNode()           {}

type BooleanLiteral struct {
	SourceSpan source.Span
	Value      bool
}

func (n *BooleanLiteral) Span() source.Span { return n.SourceSpan }
func (*BooleanLiteral) exprNode()           {}

type NilLiteral struct {
	SourceSpan source.Span
}

func (n *NilLiteral) Span() source.Span { return n.SourceSpan }
func (*NilLiteral) exprNode()           {}

type Identifier struct {
	SourceSpan source.Span
	Name       string
}

func (n *Identifier) Span() source.Span { return n.SourceSpan }
func (*Identifier) exprNode()           {}

type GroupExpr struct {
	SourceSpan source.Span
	Inner      Expr
}

func (n *GroupExpr) Span() source.Span { return n.SourceSpan }
func (*GroupExpr) exprNode()           {}

type ListLiteral struct {
	SourceSpan source.Span
	Elements   []Expr
}

func (n *ListLiteral) Span() source.Span { return n.SourceSpan }
func (*ListLiteral) exprNode()           {}

type BlockExpr struct {
	SourceSpan  source.Span
	Expressions []Expr
}

func (n *BlockExpr) Span() source.Span { return n.SourceSpan }
func (*BlockExpr) exprNode()           {}

type AssignmentExpr struct {
	SourceSpan source.Span
	Target     *Identifier
	Value      Expr
}

func (n *AssignmentExpr) Span() source.Span { return n.SourceSpan }
func (*AssignmentExpr) exprNode()           {}

type IndexExpr struct {
	SourceSpan source.Span
	Target     Expr
	Index      Expr
}

func (n *IndexExpr) Span() source.Span { return n.SourceSpan }
func (*IndexExpr) exprNode()           {}

type UnaryExpr struct {
	SourceSpan source.Span
	Operator   UnaryOperator
	Operand    Expr
}

func (n *UnaryExpr) Span() source.Span { return n.SourceSpan }
func (*UnaryExpr) exprNode()           {}

type BinaryExpr struct {
	SourceSpan source.Span
	Left       Expr
	Operator   BinaryOperator
	Right      Expr
}

func (n *BinaryExpr) Span() source.Span { return n.SourceSpan }
func (*BinaryExpr) exprNode()           {}

type ConditionalExpr struct {
	SourceSpan source.Span
	Condition  Expr
	ThenBranch Expr
	ElseBranch Expr
}

func (n *ConditionalExpr) Span() source.Span { return n.SourceSpan }
func (*ConditionalExpr) exprNode()           {}
