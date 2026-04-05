package evaluator

import (
	"fmt"
	"io"
	"math"
	"os"

	"molt/internal/ast"
	"molt/internal/diagnostic"
	"molt/internal/runtime"
)

type Evaluator struct {
	output   io.Writer
	input    io.Reader
	args     []string
	readFile func(string) ([]byte, error)
}

func New(output io.Writer) *Evaluator {
	return &Evaluator{output: output}
}

func NewWithIO(input io.Reader, output io.Writer) *Evaluator {
	return &Evaluator{
		input:  input,
		output: output,
	}
}

func NewWithContext(input io.Reader, output io.Writer, args []string) *Evaluator {
	return &Evaluator{
		input:  input,
		output: output,
		args:   append([]string(nil), args...),
	}
}

func NewWithRuntime(input io.Reader, output io.Writer, args []string, readFile func(string) ([]byte, error)) *Evaluator {
	return &Evaluator{
		input:    input,
		output:   output,
		args:     append([]string(nil), args...),
		readFile: readFile,
	}
}

func EvalProgram(program *ast.Program, env *runtime.Environment) (runtime.Value, error) {
	return (&Evaluator{}).EvalProgram(program, env)
}

func (e *Evaluator) EvalProgram(program *ast.Program, env *runtime.Environment) (runtime.Value, error) {
	env = e.prepareEnvironment(env)

	if len(program.Expressions) == 0 {
		return runtime.Nil, nil
	}

	var result runtime.Value = runtime.Nil

	for _, expr := range program.Expressions {
		value, err := e.evalExpr(env, expr)
		if err != nil {
			return nil, err
		}

		result = value
	}

	return result, nil
}

func (e *Evaluator) EvalExpr(expr ast.Expr, env *runtime.Environment) (runtime.Value, error) {
	env = e.prepareEnvironment(env)

	return e.evalExpr(env, expr)
}

func (e *Evaluator) evalExpr(env *runtime.Environment, expr ast.Expr) (runtime.Value, error) {
	switch node := expr.(type) {
	case *ast.NumberLiteral:
		return &runtime.NumberValue{Value: node.Value}, nil
	case *ast.StringLiteral:
		return &runtime.StringValue{Value: node.Value}, nil
	case *ast.BooleanLiteral:
		return &runtime.BooleanValue{Value: node.Value}, nil
	case *ast.NilLiteral:
		return runtime.Nil, nil
	case *ast.Identifier:
		value, ok := env.Get(node.Name)
		if !ok {
			return nil, e.runtimeError(node, fmt.Sprintf("undefined identifier %q", node.Name))
		}

		return value, nil
	case *ast.GroupExpr:
		return e.evalExpr(env, node.Inner)
	case *ast.ListLiteral:
		return e.evalListLiteral(env, node)
	case *ast.BlockExpr:
		return e.evalBlock(env, node)
	case *ast.AssignmentExpr:
		return e.evalAssignment(env, node)
	case *ast.IndexExpr:
		return e.evalIndex(env, node)
	case *ast.UnaryExpr:
		return e.evalUnary(env, node)
	case *ast.BinaryExpr:
		return e.evalBinary(env, node)
	case *ast.ConditionalExpr:
		return e.evalConditional(env, node)
	case *ast.NamedFunctionExpr:
		return e.evalNamedFunction(env, node), nil
	case *ast.FunctionLiteralExpr:
		return e.makeFunctionValue(env, "", node.Parameters, node.Body), nil
	case *ast.CallExpr:
		return e.evalCall(env, node)
	case *ast.OperatorLiteral:
		return nil, e.runtimeError(node, "operator literals are only valid inside mutation rules")
	case *ast.QuoteExpr:
		return &runtime.CodeValue{
			Body: node.Body,
			Env:  env,
		}, nil
	case *ast.MutationLiteralExpr:
		return e.evalMutationLiteral(node)
	case *ast.ApplyMutationExpr:
		return e.evalApplyMutation(env, node)
	default:
		return nil, fmt.Errorf("unsupported expression type %T", expr)
	}
}

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
	value, err := e.evalExpr(env, expr.Value)
	if err != nil {
		return nil, err
	}

	env.Assign(expr.Target.Name, value)
	return value, nil
}

func (e *Evaluator) evalIndex(env *runtime.Environment, expr *ast.IndexExpr) (runtime.Value, error) {
	target, err := e.evalExpr(env, expr.Target)
	if err != nil {
		return nil, err
	}

	list, ok := target.(*runtime.ListValue)
	if !ok {
		return nil, e.runtimeError(expr, fmt.Sprintf("cannot index value of type %q", target.TypeName()))
	}

	indexValue, err := e.evalExpr(env, expr.Index)
	if err != nil {
		return nil, err
	}

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

func (e *Evaluator) evalConditional(env *runtime.Environment, expr *ast.ConditionalExpr) (runtime.Value, error) {
	condition, err := e.evalExpr(env, expr.Condition)
	if err != nil {
		return nil, err
	}

	boolean, ok := condition.(*runtime.BooleanValue)
	if !ok {
		return nil, e.runtimeError(expr.Condition, fmt.Sprintf("if condition must be boolean, got %q", condition.TypeName()))
	}

	if boolean.Value {
		return e.evalExpr(env, expr.ThenBranch)
	}

	return e.evalExpr(env, expr.ElseBranch)
}

func (e *Evaluator) evalNamedFunction(env *runtime.Environment, expr *ast.NamedFunctionExpr) runtime.Value {
	function := e.makeFunctionValue(env, expr.Name.Name, expr.Parameters, expr.Body)
	env.Define(expr.Name.Name, function)
	return function
}

func (e *Evaluator) evalMutationLiteral(expr *ast.MutationLiteralExpr) (runtime.Value, error) {
	rules := runtime.CloneRules(expr.Rules)
	if err := runtime.ValidateMutationRules(rules); err != nil {
		return nil, e.runtimeError(expr, err.Error())
	}

	return &runtime.MutationValue{Rules: rules}, nil
}

func (e *Evaluator) evalApplyMutation(env *runtime.Environment, expr *ast.ApplyMutationExpr) (runtime.Value, error) {
	target, err := e.evalExpr(env, expr.Target)
	if err != nil {
		return nil, err
	}

	mutationValue, err := e.evalExpr(env, expr.Mutation)
	if err != nil {
		return nil, err
	}

	mutation, ok := mutationValue.(*runtime.MutationValue)
	if !ok {
		return nil, e.runtimeError(expr.Mutation, fmt.Sprintf("expected mutation value, got %q", mutationValue.TypeName()))
	}

	rewritten, err := runtime.ApplyMutationValue(target, mutation)
	if err != nil {
		return nil, e.runtimeError(expr.Target, err.Error())
	}

	return rewritten, nil
}

func (e *Evaluator) makeFunctionValue(env *runtime.Environment, name string, parameters []*ast.Identifier, body ast.Expr) *runtime.UserFunctionValue {
	names := make([]string, 0, len(parameters))
	for _, parameter := range parameters {
		names = append(names, parameter.Name)
	}

	return &runtime.UserFunctionValue{
		Name:       name,
		Parameters: names,
		Body:       body,
		Env:        env,
	}
}

func (e *Evaluator) evalCall(env *runtime.Environment, expr *ast.CallExpr) (runtime.Value, error) {
	callee, err := e.evalExpr(env, expr.Callee)
	if err != nil {
		return nil, err
	}

	args := make([]runtime.Value, 0, len(expr.Arguments))
	for _, argumentExpr := range expr.Arguments {
		argument, err := e.evalExpr(env, argumentExpr)
		if err != nil {
			return nil, err
		}

		args = append(args, argument)
	}

	switch fn := callee.(type) {
	case *runtime.UserFunctionValue:
		if len(args) != len(fn.Parameters) {
			return nil, e.runtimeError(expr, arityMessage(len(fn.Parameters), len(args)))
		}

		callEnv := runtime.NewEnvironment(fn.Env)
		for i, parameter := range fn.Parameters {
			callEnv.Define(parameter, args[i])
		}

		return e.evalExpr(callEnv, fn.Body)
	case runtime.NativeFunction:
		if native, ok := callee.(*runtime.NativeFunctionValue); ok && native.Arity >= 0 && len(args) != native.Arity {
			return nil, e.runtimeError(expr, arityMessage(native.Arity, len(args)))
		}

		result, err := fn.Call(&runtime.CallContext{
			FunctionName: fn.Name(),
			Environment:  env,
			Arguments:    e.arguments(),
			CallSpan:     expr.Span(),
			EvalCode:     e.evalCodeValue,
			ReadFile:     e.readFileFunc(),
			Input:        e.inputReader(),
			Output:       e.outputWriter(),
		}, args)
		if err != nil {
			return nil, err
		}

		return result, nil
	default:
		return nil, e.runtimeError(expr.Callee, fmt.Sprintf("cannot call value of type %q", callee.TypeName()))
	}
}

func (e *Evaluator) runtimeError(node ast.Expr, message string) error {
	return diagnostic.NewRuntimeError(message, node.Span())
}

func (e *Evaluator) prepareEnvironment(env *runtime.Environment) *runtime.Environment {
	if env == nil {
		env = runtime.NewEnvironment(nil)
	}

	e.ensureBuiltins(env)
	return env
}

func (e *Evaluator) ensureBuiltins(env *runtime.Environment) {
	if _, ok := env.Get("eval"); !ok {
		env.Define("eval", &runtime.NativeFunctionValue{
			FunctionName: "eval",
			Arity:        1,
			Impl:         evalBuiltin,
		})
	}

	if _, ok := env.Get("type"); !ok {
		env.Define("type", &runtime.NativeFunctionValue{
			FunctionName: "type",
			Arity:        1,
			Impl:         typeBuiltin,
		})
	}

	if _, ok := env.Get("args"); !ok {
		env.Define("args", &runtime.NativeFunctionValue{
			FunctionName: "args",
			Arity:        0,
			Impl:         argsBuiltin,
		})
	}

	if _, ok := env.Get("len"); !ok {
		env.Define("len", &runtime.NativeFunctionValue{
			FunctionName: "len",
			Arity:        1,
			Impl:         lenBuiltin,
		})
	}

	if _, ok := env.Get("push"); !ok {
		env.Define("push", &runtime.NativeFunctionValue{
			FunctionName: "push",
			Arity:        2,
			Impl:         pushBuiltin,
		})
	}

	if _, ok := env.Get("show"); !ok {
		env.Define("show", &runtime.NativeFunctionValue{
			FunctionName: "show",
			Arity:        1,
			Impl:         showBuiltin,
		})
	}

	if _, ok := env.Get("read_file"); !ok {
		env.Define("read_file", &runtime.NativeFunctionValue{
			FunctionName: "read_file",
			Arity:        1,
			Impl:         readFileBuiltin,
		})
	}

	if _, ok := env.Get("print"); !ok {
		env.Define("print", &runtime.NativeFunctionValue{
			FunctionName: "print",
			Arity:        1,
			Impl:         printBuiltin,
		})
	}

	if _, ok := env.Get("stdin"); !ok {
		env.Define("stdin", &runtime.NativeFunctionValue{
			FunctionName: "stdin",
			Arity:        0,
			Impl:         stdinBuiltin,
		})
	}
}

func (e *Evaluator) evalCodeValue(code *runtime.CodeValue) (runtime.Value, error) {
	if code == nil {
		return nil, fmt.Errorf("nil code value")
	}

	captured := code.Env
	if captured == nil {
		captured = runtime.NewEnvironment(nil)
	}

	e.ensureBuiltins(captured)

	frame := runtime.NewEnvironment(captured)
	e.ensureBuiltins(frame)

	return e.evalExpr(frame, code.Body)
}

func evalBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	code, ok := args[0].(*runtime.CodeValue)
	if !ok {
		return nil, diagnostic.NewRuntimeError(
			fmt.Sprintf("eval expects code value, got %q", args[0].TypeName()),
			ctx.CallSpan,
		)
	}

	if ctx.EvalCode == nil {
		return nil, fmt.Errorf("eval builtin is missing evaluator callback")
	}

	return ctx.EvalCode(code)
}

func typeBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	return &runtime.StringValue{Value: args[0].TypeName()}, nil
}

func argsBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	values := make([]runtime.Value, 0, len(ctx.Arguments))
	for _, arg := range ctx.Arguments {
		values = append(values, &runtime.StringValue{Value: arg})
	}

	return &runtime.ListValue{Elements: values}, nil
}

func lenBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	switch value := args[0].(type) {
	case *runtime.ListValue:
		return &runtime.NumberValue{Value: float64(len(value.Elements))}, nil
	case *runtime.StringValue:
		return &runtime.NumberValue{Value: float64(len([]rune(value.Value)))}, nil
	default:
		return nil, diagnostic.NewRuntimeError(
			fmt.Sprintf("len expects list or string, got %q", args[0].TypeName()),
			ctx.CallSpan,
		)
	}
}

func pushBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	list, ok := args[0].(*runtime.ListValue)
	if !ok {
		return nil, diagnostic.NewRuntimeError(
			fmt.Sprintf("push expects list as first argument, got %q", args[0].TypeName()),
			ctx.CallSpan,
		)
	}

	list.Elements = append(list.Elements, args[1])
	return list, nil
}

func showBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	return &runtime.StringValue{Value: runtime.ShowValue(args[0])}, nil
}

func readFileBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	path, ok := args[0].(*runtime.StringValue)
	if !ok {
		return nil, diagnostic.NewRuntimeError(
			fmt.Sprintf("read_file expects string path, got %q", args[0].TypeName()),
			ctx.CallSpan,
		)
	}

	if path.Value == "" {
		return nil, diagnostic.NewRuntimeError("read_file path cannot be empty", ctx.CallSpan)
	}

	reader := ctx.ReadFile
	if reader == nil {
		reader = os.ReadFile
	}

	data, err := reader(path.Value)
	if err != nil {
		return nil, diagnostic.NewRuntimeError(
			fmt.Sprintf("read_file failed for %q: %v", path.Value, err),
			ctx.CallSpan,
		)
	}

	return &runtime.StringValue{Value: string(data)}, nil
}

func printBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	if _, err := fmt.Fprintln(outputWriter(ctx.Output), runtime.ShowValue(args[0])); err != nil {
		return nil, err
	}

	return runtime.Nil, nil
}

func stdinBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	text, err := io.ReadAll(inputReader(ctx.Input))
	if err != nil {
		return nil, diagnostic.NewRuntimeError(
			fmt.Sprintf("stdin failed: %v", err),
			ctx.CallSpan,
		)
	}

	return &runtime.StringValue{Value: string(text)}, nil
}

func (e *Evaluator) outputWriter() io.Writer {
	return outputWriter(e.output)
}

func (e *Evaluator) inputReader() io.Reader {
	return inputReader(e.input)
}

func (e *Evaluator) arguments() []string {
	return append([]string(nil), e.args...)
}

func (e *Evaluator) readFileFunc() func(string) ([]byte, error) {
	if e.readFile != nil {
		return e.readFile
	}

	return os.ReadFile
}

func inputReader(reader io.Reader) io.Reader {
	if reader != nil {
		return reader
	}

	return os.Stdin
}

func outputWriter(writer io.Writer) io.Writer {
	if writer != nil {
		return writer
	}

	return os.Stdout
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

func arityMessage(expected, got int) string {
	return fmt.Sprintf("expected %d arguments but got %d", expected, got)
}
