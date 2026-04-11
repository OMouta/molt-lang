package ast

import "molt/internal/source"

// ImportKind distinguishes how a module is imported.
type ImportKind int

const (
	// ImportModule loads a module and binds it as a namespace record.
	// Syntax: import "./path" or import "./path" as alias
	ImportModule ImportKind = iota

	// ImportNamed pulls one or more named exports directly from a module.
	// Syntax: import name from "./path" or import {a, b} from "./path"
	ImportNamed
)

type ImportExpr struct {
	SourceSpan source.Span
	Kind       ImportKind
	// ImportModule: the alias identifier (nil = auto-derive from path stem).
	Name *Identifier
	// ImportNamed: one or more export names to import and bind directly.
	Names []*Identifier
	Path  *StringLiteral
}

func (n *ImportExpr) Span() source.Span { return n.SourceSpan }
func (*ImportExpr) exprNode()           {}

type ExportExpr struct {
	SourceSpan source.Span
	Name       *Identifier
}

func (n *ExportExpr) Span() source.Span { return n.SourceSpan }
func (*ExportExpr) exprNode()           {}
