package runtime

import (
	"fmt"
	"io"

	"molt/internal/ast"
	"molt/internal/runtime/typenames"
	"molt/internal/source"
)

// Value is implemented by every runtime value in @molt.
type Value interface {
	TypeName() string
}

type NumberValue struct {
	Value float64
}

func (v *NumberValue) TypeName() string { return typenames.Number }

type StringValue struct {
	Value string
}

func (v *StringValue) TypeName() string { return typenames.String }

type BooleanValue struct {
	Value bool
}

func (v *BooleanValue) TypeName() string { return typenames.Boolean }

type NilValue struct{}

func (NilValue) TypeName() string { return typenames.Nil }

var Nil = NilValue{}

type ListValue struct {
	Elements []Value
}

func (v *ListValue) TypeName() string { return typenames.List }

type UserFunctionValue struct {
	Name       string
	Parameters []string
	Body       ast.Expr
	Env        *Environment
}

func (v *UserFunctionValue) TypeName() string { return typenames.Function }

type NativeFunction interface {
	Value
	Name() string
	Call(*CallContext, []Value) (Value, error)
}

type NativeFunctionValue struct {
	FunctionName string
	Arity        int
	Impl         func(*CallContext, []Value) (Value, error)
}

func (v *NativeFunctionValue) TypeName() string { return typenames.NativeFunction }
func (v *NativeFunctionValue) Name() string     { return v.FunctionName }

func (v *NativeFunctionValue) Call(ctx *CallContext, args []Value) (Value, error) {
	if v.Impl == nil {
		return nil, fmt.Errorf("native function %q has no implementation", v.FunctionName)
	}

	return v.Impl(ctx, args)
}

type CodeValue struct {
	Body ast.Expr
	Env  *Environment
}

func (v *CodeValue) TypeName() string { return typenames.Code }

type MutationValue struct {
	Rules []*ast.MutationRule
}

func (v *MutationValue) TypeName() string { return typenames.Mutation }

type CallContext struct {
	FunctionName string
	Environment  *Environment
	CallSpan     source.Span
	EvalCode     func(*CodeValue) (Value, error)
	Output       io.Writer
}
