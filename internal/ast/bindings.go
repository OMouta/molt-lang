package ast

import "molt/internal/source"

type BindingPattern interface {
	Expr
	bindingPatternNode()
}

type AssignmentTarget interface {
	Expr
	assignmentTargetNode()
}

func (*Identifier) bindingPatternNode()   {}
func (*Identifier) assignmentTargetNode() {}

type ListBindingPattern struct {
	SourceSpan source.Span
	Elements   []BindingPattern
}

func (n *ListBindingPattern) Span() source.Span   { return n.SourceSpan }
func (*ListBindingPattern) exprNode()             {}
func (*ListBindingPattern) bindingPatternNode()   {}
func (*ListBindingPattern) assignmentTargetNode() {}

type RecordBindingField struct {
	SourceSpan source.Span
	Name       *Identifier
	Value      BindingPattern
}

func (n *RecordBindingField) Span() source.Span { return n.SourceSpan }

type RecordBindingPattern struct {
	SourceSpan source.Span
	Fields     []*RecordBindingField
}

func (n *RecordBindingPattern) Span() source.Span   { return n.SourceSpan }
func (*RecordBindingPattern) exprNode()             {}
func (*RecordBindingPattern) bindingPatternNode()   {}
func (*RecordBindingPattern) assignmentTargetNode() {}

func (*FieldAccessExpr) assignmentTargetNode() {}
