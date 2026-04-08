package evaluator

import (
	"fmt"
	"math"

	"molt/internal/ast"
	"molt/internal/runtime"
)

func (e *Evaluator) evalUnary(env *runtime.Environment, expr *ast.UnaryExpr) (runtime.Value, error) {
	operand, err := e.evalExpr(env, expr.Operand)
	if err != nil {
		return nil, err
	}

	switch expr.Operator {
	case ast.UnaryNegate:
		number, ok := operand.(*runtime.NumberValue)
		if !ok {
			return nil, e.runtimeError(expr, fmt.Sprintf("operator '-' requires number operand, got %q", operand.TypeName()))
		}

		return &runtime.NumberValue{Value: -number.Value}, nil
	case ast.UnaryNot:
		boolean, ok := operand.(*runtime.BooleanValue)
		if !ok {
			return nil, e.runtimeError(expr, fmt.Sprintf("operator 'not' requires boolean operand, got %q", operand.TypeName()))
		}

		return &runtime.BooleanValue{Value: !boolean.Value}, nil
	default:
		return nil, fmt.Errorf("unsupported unary operator %q", expr.Operator)
	}
}

func (e *Evaluator) evalBinary(env *runtime.Environment, expr *ast.BinaryExpr) (runtime.Value, error) {
	switch expr.Operator {
	case ast.BinaryAnd:
		return e.evalAnd(env, expr)
	case ast.BinaryOr:
		return e.evalOr(env, expr)
	case ast.BinaryEqual, ast.BinaryNotEqual:
		return e.evalEquality(env, expr)
	case ast.BinaryLess, ast.BinaryLessEqual, ast.BinaryGreater, ast.BinaryGreaterEqual:
		return e.evalRelational(env, expr)
	case ast.BinaryAdd, ast.BinarySubtract, ast.BinaryMultiply, ast.BinaryDivide, ast.BinaryModulo:
		return e.evalArithmetic(env, expr)
	default:
		return nil, fmt.Errorf("unsupported binary operator %q", expr.Operator)
	}
}

func (e *Evaluator) evalAnd(env *runtime.Environment, expr *ast.BinaryExpr) (runtime.Value, error) {
	left, err := e.evalExpr(env, expr.Left)
	if err != nil {
		return nil, err
	}

	leftBool, ok := left.(*runtime.BooleanValue)
	if !ok {
		return nil, e.runtimeError(expr.Left, fmt.Sprintf("operator 'and' requires boolean operands, got %q", left.TypeName()))
	}

	if !leftBool.Value {
		return &runtime.BooleanValue{Value: false}, nil
	}

	right, err := e.evalExpr(env, expr.Right)
	if err != nil {
		return nil, err
	}

	rightBool, ok := right.(*runtime.BooleanValue)
	if !ok {
		return nil, e.runtimeError(expr.Right, fmt.Sprintf("operator 'and' requires boolean operands, got %q", right.TypeName()))
	}

	return &runtime.BooleanValue{Value: rightBool.Value}, nil
}

func (e *Evaluator) evalOr(env *runtime.Environment, expr *ast.BinaryExpr) (runtime.Value, error) {
	left, err := e.evalExpr(env, expr.Left)
	if err != nil {
		return nil, err
	}

	leftBool, ok := left.(*runtime.BooleanValue)
	if !ok {
		return nil, e.runtimeError(expr.Left, fmt.Sprintf("operator 'or' requires boolean operands, got %q", left.TypeName()))
	}

	if leftBool.Value {
		return &runtime.BooleanValue{Value: true}, nil
	}

	right, err := e.evalExpr(env, expr.Right)
	if err != nil {
		return nil, err
	}

	rightBool, ok := right.(*runtime.BooleanValue)
	if !ok {
		return nil, e.runtimeError(expr.Right, fmt.Sprintf("operator 'or' requires boolean operands, got %q", right.TypeName()))
	}

	return &runtime.BooleanValue{Value: rightBool.Value}, nil
}

func (e *Evaluator) evalEquality(env *runtime.Environment, expr *ast.BinaryExpr) (runtime.Value, error) {
	left, err := e.evalExpr(env, expr.Left)
	if err != nil {
		return nil, err
	}

	right, err := e.evalExpr(env, expr.Right)
	if err != nil {
		return nil, err
	}

	equal := valuesEqual(left, right)
	if expr.Operator == ast.BinaryNotEqual {
		equal = !equal
	}

	return &runtime.BooleanValue{Value: equal}, nil
}

func (e *Evaluator) evalRelational(env *runtime.Environment, expr *ast.BinaryExpr) (runtime.Value, error) {
	left, err := e.evalExpr(env, expr.Left)
	if err != nil {
		return nil, err
	}

	right, err := e.evalExpr(env, expr.Right)
	if err != nil {
		return nil, err
	}

	leftNumber, ok := left.(*runtime.NumberValue)
	if !ok {
		return nil, e.runtimeError(expr.Left, fmt.Sprintf("operator %q requires number operands, got %q", expr.Operator, left.TypeName()))
	}

	rightNumber, ok := right.(*runtime.NumberValue)
	if !ok {
		return nil, e.runtimeError(expr.Right, fmt.Sprintf("operator %q requires number operands, got %q", expr.Operator, right.TypeName()))
	}

	var result bool
	switch expr.Operator {
	case ast.BinaryLess:
		result = leftNumber.Value < rightNumber.Value
	case ast.BinaryLessEqual:
		result = leftNumber.Value <= rightNumber.Value
	case ast.BinaryGreater:
		result = leftNumber.Value > rightNumber.Value
	case ast.BinaryGreaterEqual:
		result = leftNumber.Value >= rightNumber.Value
	default:
		return nil, fmt.Errorf("unsupported relational operator %q", expr.Operator)
	}

	return &runtime.BooleanValue{Value: result}, nil
}

func (e *Evaluator) evalArithmetic(env *runtime.Environment, expr *ast.BinaryExpr) (runtime.Value, error) {
	left, err := e.evalExpr(env, expr.Left)
	if err != nil {
		return nil, err
	}

	right, err := e.evalExpr(env, expr.Right)
	if err != nil {
		return nil, err
	}

	leftNumber, ok := left.(*runtime.NumberValue)
	if !ok {
		return nil, e.runtimeError(expr.Left, fmt.Sprintf("operator %q requires number operands, got %q", expr.Operator, left.TypeName()))
	}

	rightNumber, ok := right.(*runtime.NumberValue)
	if !ok {
		return nil, e.runtimeError(expr.Right, fmt.Sprintf("operator %q requires number operands, got %q", expr.Operator, right.TypeName()))
	}

	switch expr.Operator {
	case ast.BinaryAdd:
		return &runtime.NumberValue{Value: leftNumber.Value + rightNumber.Value}, nil
	case ast.BinarySubtract:
		return &runtime.NumberValue{Value: leftNumber.Value - rightNumber.Value}, nil
	case ast.BinaryMultiply:
		return &runtime.NumberValue{Value: leftNumber.Value * rightNumber.Value}, nil
	case ast.BinaryDivide:
		return &runtime.NumberValue{Value: leftNumber.Value / rightNumber.Value}, nil
	case ast.BinaryModulo:
		return &runtime.NumberValue{Value: math.Mod(leftNumber.Value, rightNumber.Value)}, nil
	default:
		return nil, fmt.Errorf("unsupported arithmetic operator %q", expr.Operator)
	}
}

func valuesEqual(left, right runtime.Value) bool {
	switch l := left.(type) {
	case *runtime.NumberValue:
		r, ok := right.(*runtime.NumberValue)
		return ok && l.Value == r.Value
	case *runtime.StringValue:
		r, ok := right.(*runtime.StringValue)
		return ok && l.Value == r.Value
	case *runtime.BooleanValue:
		r, ok := right.(*runtime.BooleanValue)
		return ok && l.Value == r.Value
	case runtime.NilValue:
		_, ok := right.(runtime.NilValue)
		return ok
	case *runtime.ListValue:
		r, ok := right.(*runtime.ListValue)
		return ok && l == r
	case *runtime.RecordValue:
		r, ok := right.(*runtime.RecordValue)
		return ok && l == r
	case *runtime.ErrorValue:
		r, ok := right.(*runtime.ErrorValue)
		return ok && l == r
	case *runtime.UserFunctionValue:
		r, ok := right.(*runtime.UserFunctionValue)
		return ok && l == r
	case *runtime.NativeFunctionValue:
		r, ok := right.(*runtime.NativeFunctionValue)
		return ok && l == r
	case *runtime.CodeValue:
		r, ok := right.(*runtime.CodeValue)
		return ok && l == r
	case *runtime.MutationValue:
		r, ok := right.(*runtime.MutationValue)
		return ok && l == r
	default:
		return false
	}
}
