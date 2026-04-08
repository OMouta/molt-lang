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

type RecordField struct {
	Name  string
	Value Value
}

type RecordValue struct {
	Fields     []RecordField
	fieldIndex map[string]int
}

func NewRecordValue(fields []RecordField) *RecordValue {
	cloned := make([]RecordField, len(fields))
	copy(cloned, fields)

	fieldIndex := make(map[string]int, len(cloned))
	for index, field := range cloned {
		fieldIndex[field.Name] = index
	}

	return &RecordValue{
		Fields:     cloned,
		fieldIndex: fieldIndex,
	}
}

func (v *RecordValue) TypeName() string { return typenames.Record }

func (v *RecordValue) GetField(name string) (Value, bool) {
	if v == nil {
		return nil, false
	}

	if v.fieldIndex == nil {
		v.fieldIndex = make(map[string]int, len(v.Fields))
		for index, field := range v.Fields {
			v.fieldIndex[field.Name] = index
		}
	}

	index, ok := v.fieldIndex[name]
	if !ok {
		return nil, false
	}

	return v.Fields[index].Value, true
}

func (v *RecordValue) SetField(name string, value Value) bool {
	if v == nil {
		return false
	}

	if v.fieldIndex == nil {
		v.fieldIndex = make(map[string]int, len(v.Fields))
		for index, field := range v.Fields {
			v.fieldIndex[field.Name] = index
		}
	}

	if index, ok := v.fieldIndex[name]; ok {
		v.Fields[index].Value = value
		return true
	}

	v.Fields = append(v.Fields, RecordField{Name: name, Value: value})
	v.fieldIndex[name] = len(v.Fields) - 1
	return false
}

func (v *RecordValue) Len() int {
	if v == nil {
		return 0
	}

	return len(v.Fields)
}

func (v *RecordValue) Keys() []string {
	if v == nil || len(v.Fields) == 0 {
		return nil
	}

	keys := make([]string, 0, len(v.Fields))
	for _, field := range v.Fields {
		keys = append(keys, field.Name)
	}

	return keys
}

func (v *RecordValue) Values() []Value {
	if v == nil || len(v.Fields) == 0 {
		return nil
	}

	values := make([]Value, 0, len(v.Fields))
	for _, field := range v.Fields {
		values = append(values, field.Value)
	}

	return values
}

type ErrorValue struct {
	Message string
	Data    Value
	HasData bool
}

func NewErrorValue(message string, data Value, hasData bool) *ErrorValue {
	return &ErrorValue{
		Message: message,
		Data:    data,
		HasData: hasData,
	}
}

func (v *ErrorValue) TypeName() string { return typenames.Error }

func (v *ErrorValue) GetField(name string) (Value, bool) {
	if v == nil {
		return nil, false
	}

	switch name {
	case "message":
		return &StringValue{Value: v.Message}, true
	case "data":
		if !v.HasData {
			return nil, false
		}
		return v.Data, true
	default:
		return nil, false
	}
}

func (v *ErrorValue) Len() int {
	if v == nil {
		return 0
	}

	if v.HasData {
		return 2
	}

	return 1
}

func (v *ErrorValue) Keys() []string {
	if v == nil {
		return nil
	}

	if v.HasData {
		return []string{"message", "data"}
	}

	return []string{"message"}
}

func (v *ErrorValue) Values() []Value {
	if v == nil {
		return nil
	}

	values := []Value{&StringValue{Value: v.Message}}
	if v.HasData {
		values = append(values, v.Data)
	}

	return values
}

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
	Arguments    []string
	CallSpan     source.Span
	EvalCode     func(*CodeValue) (Value, error)
	Invoke       func(Value, []Value, *Environment, source.Span) (Value, error)
	ReadFile     func(string) ([]byte, error)
	WriteFile    func(string, []byte) error
	Output       io.Writer
	Input        io.Reader
}
