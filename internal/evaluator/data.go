package evaluator

import (
	"fmt"
	"math"

	"molt/internal/ast"
	"molt/internal/runtime"
)

func (e *Evaluator) evalListLiteral(env *runtime.Environment, expr *ast.ListLiteral) (runtime.Value, error) {
	elements := make([]runtime.Value, 0, len(expr.Elements))
	for _, element := range expr.Elements {
		value, err := e.evalExpr(env, element)
		if err != nil {
			return nil, err
		}

		elements = append(elements, value)
	}

	return &runtime.ListValue{Elements: elements}, nil
}

func (e *Evaluator) evalRecordLiteral(env *runtime.Environment, expr *ast.RecordLiteral) (runtime.Value, error) {
	fields := make([]runtime.RecordField, 0, len(expr.Fields))
	for _, field := range expr.Fields {
		value, err := e.evalExpr(env, field.Value)
		if err != nil {
			return nil, err
		}

		fields = append(fields, runtime.RecordField{
			Name:  field.Name.Name,
			Value: value,
		})
	}

	return runtime.NewRecordValue(fields), nil
}

func (e *Evaluator) evalBlock(env *runtime.Environment, expr *ast.BlockExpr) (runtime.Value, error) {
	blockEnv := runtime.NewEnvironment(env)
	if len(expr.Expressions) == 0 {
		return runtime.Nil, nil
	}

	var result runtime.Value = runtime.Nil
	for _, inner := range expr.Expressions {
		value, err := e.evalExpr(blockEnv, inner)
		if err != nil {
			return nil, err
		}

		result = value
	}

	return result, nil
}

func (e *Evaluator) evalAssignment(env *runtime.Environment, expr *ast.AssignmentExpr) (runtime.Value, error) {
	switch target := expr.Target.(type) {
	case *ast.Identifier:
		value, err := e.evalExpr(env, expr.Value)
		if err != nil {
			return nil, err
		}

		env.Assign(target.Name, value)
		return value, nil
	case *ast.FieldAccessExpr:
		recordValue, err := e.evalExpr(env, target.Target)
		if err != nil {
			return nil, err
		}

		record, ok := recordValue.(*runtime.RecordValue)
		if !ok {
			return nil, e.runtimeError(target, fmt.Sprintf("cannot assign field %q on value of type %q", target.Name.Name, recordValue.TypeName()))
		}

		value, err := e.evalExpr(env, expr.Value)
		if err != nil {
			return nil, err
		}

		record.SetField(target.Name.Name, value)
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported assignment target type %T", expr.Target)
	}
}

func (e *Evaluator) evalIndex(env *runtime.Environment, expr *ast.IndexExpr) (runtime.Value, error) {
	target, err := e.evalExpr(env, expr.Target)
	if err != nil {
		return nil, err
	}

	indexValue, err := e.evalExpr(env, expr.Index)
	if err != nil {
		return nil, err
	}

	if list, ok := target.(*runtime.ListValue); ok {
		number, ok := indexValue.(*runtime.NumberValue)
		if !ok {
			return nil, e.runtimeError(expr.Index, fmt.Sprintf("list index must be a number, got %q", indexValue.TypeName()))
		}

		if number.Value < 0 || math.Trunc(number.Value) != number.Value {
			return nil, e.runtimeError(expr.Index, fmt.Sprintf("list index must be a non-negative integer, got %v", number.Value))
		}

		index := int(number.Value)
		if index >= len(list.Elements) {
			return nil, e.runtimeError(expr, fmt.Sprintf("list index %d out of bounds", index))
		}

		return list.Elements[index], nil
	}

	if record, ok := target.(*runtime.RecordValue); ok {
		name, ok := indexValue.(*runtime.StringValue)
		if !ok {
			return nil, e.runtimeError(expr.Index, fmt.Sprintf("record index must be a string, got %q", indexValue.TypeName()))
		}

		value, ok := record.GetField(name.Value)
		if !ok {
			return nil, e.runtimeError(expr.Index, fmt.Sprintf("record has no field %q", name.Value))
		}

		return value, nil
	}

	if errValue, ok := target.(*runtime.ErrorValue); ok {
		name, ok := indexValue.(*runtime.StringValue)
		if !ok {
			return nil, e.runtimeError(expr.Index, fmt.Sprintf("error index must be a string, got %q", indexValue.TypeName()))
		}

		value, ok := errValue.GetField(name.Value)
		if !ok {
			return nil, e.runtimeError(expr.Index, fmt.Sprintf("error has no field %q", name.Value))
		}

		return value, nil
	}

	return nil, e.runtimeError(expr, fmt.Sprintf("cannot index value of type %q", target.TypeName()))
}

func (e *Evaluator) evalFieldAccess(env *runtime.Environment, expr *ast.FieldAccessExpr) (runtime.Value, error) {
	target, err := e.evalExpr(env, expr.Target)
	if err != nil {
		return nil, err
	}

	if record, ok := target.(*runtime.RecordValue); ok {
		value, ok := record.GetField(expr.Name.Name)
		if !ok {
			return nil, e.runtimeError(expr.Name, fmt.Sprintf("record has no field %q", expr.Name.Name))
		}

		return value, nil
	}

	if errValue, ok := target.(*runtime.ErrorValue); ok {
		value, ok := errValue.GetField(expr.Name.Name)
		if !ok {
			return nil, e.runtimeError(expr.Name, fmt.Sprintf("error has no field %q", expr.Name.Name))
		}

		return value, nil
	}

	return nil, e.runtimeError(expr, fmt.Sprintf("cannot access field %q on value of type %q", expr.Name.Name, target.TypeName()))
}
