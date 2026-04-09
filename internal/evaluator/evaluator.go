package evaluator

import (
	"bufio"
	"fmt"
	"io"

	"molt/internal/ast"
	"molt/internal/runtime"
	"molt/internal/source"
)

type Evaluator struct {
	output          io.Writer
	input           io.Reader
	inputBuf        *bufio.Reader
	args            []string
	readFile        func(string) ([]byte, error)
	writeFile       func(string, []byte) error
	moduleCache     map[string][]runtime.Binding
	moduleLoadStack []string
	moduleStack     []*moduleExecution
	runDepth        int
}

type moduleExecution struct {
	env        *runtime.Environment
	exported   map[string]source.Span
	exportList []string
}

type loopControlKind string

const (
	loopControlBreak    loopControlKind = "break"
	loopControlContinue loopControlKind = "continue"
)

type loopControlSignal struct {
	kind loopControlKind
	span source.Span
}

func (s loopControlSignal) Error() string {
	return string(s.kind)
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

func NewWithRuntime(input io.Reader, output io.Writer, args []string, readFile func(string) ([]byte, error), writeFile func(string, []byte) error) *Evaluator {
	return &Evaluator{
		input:     input,
		output:    output,
		args:      append([]string(nil), args...),
		readFile:  readFile,
		writeFile: writeFile,
	}
}

func EvalProgram(program *ast.Program, env *runtime.Environment) (runtime.Value, error) {
	return (&Evaluator{}).EvalProgram(program, env)
}

func (e *Evaluator) EvalProgram(program *ast.Program, env *runtime.Environment) (runtime.Value, error) {
	env = e.prepareEnvironment(env)
	e.beginRun()
	defer e.endRun()

	value, err := e.evalProgramRaw(program, env)
	if err != nil {
		return nil, e.wrapControlFlowError(err)
	}

	return value, nil
}

func (e *Evaluator) evalProgramRaw(program *ast.Program, env *runtime.Environment) (runtime.Value, error) {
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
	e.beginRun()
	defer e.endRun()

	value, err := e.evalExpr(env, expr)
	if err != nil {
		return nil, e.wrapControlFlowError(err)
	}

	return value, nil
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
	case *ast.BreakExpr:
		return nil, loopControlSignal{kind: loopControlBreak, span: node.Span()}
	case *ast.ContinueExpr:
		return nil, loopControlSignal{kind: loopControlContinue, span: node.Span()}
	case *ast.Identifier:
		value, ok := env.Get(node.Name)
		if !ok {
			return nil, e.runtimeError(node, fmt.Sprintf("undefined identifier %q", node.Name))
		}

		return value, nil
	case *ast.ExportExpr:
		return e.evalExport(env, node)
	case *ast.ImportExpr:
		return e.evalImport(env, node)
	case *ast.GroupExpr:
		return e.evalExpr(env, node.Inner)
	case *ast.ListLiteral:
		return e.evalListLiteral(env, node)
	case *ast.RecordLiteral:
		return e.evalRecordLiteral(env, node)
	case *ast.BlockExpr:
		return e.evalBlock(env, node)
	case *ast.AssignmentExpr:
		return e.evalAssignment(env, node)
	case *ast.IndexExpr:
		return e.evalIndex(env, node)
	case *ast.FieldAccessExpr:
		return e.evalFieldAccess(env, node)
	case *ast.UnaryExpr:
		return e.evalUnary(env, node)
	case *ast.BinaryExpr:
		return e.evalBinary(env, node)
	case *ast.ConditionalExpr:
		return e.evalConditional(env, node)
	case *ast.WhileExpr:
		return e.evalWhile(env, node)
	case *ast.TryCatchExpr:
		return e.evalTryCatch(env, node)
	case *ast.MatchExpr:
		return e.evalMatch(env, node)
	case *ast.ForInExpr:
		return e.evalForIn(env, node)
	case *ast.NamedFunctionExpr:
		return e.evalNamedFunction(env, node), nil
	case *ast.FunctionLiteralExpr:
		return e.makeFunctionValue(env, "", node.Parameters, node.Body), nil
	case *ast.CallExpr:
		return e.evalCall(env, node)
	case *ast.OperatorLiteral:
		return nil, e.runtimeError(node, "operator literals are only valid inside mutation rules")
	case *ast.QuoteExpr:
		return e.evalQuote(env, node)
	case *ast.UnquoteExpr:
		return nil, e.runtimeError(node, "unquote is only valid inside quotes")
	case *ast.SpliceExpr:
		return nil, e.runtimeError(node, "splice is only valid inside quotes")
	case *ast.MutationLiteralExpr:
		return e.evalMutationLiteral(node)
	case *ast.ApplyMutationExpr:
		return e.evalApplyMutation(env, node)
	default:
		return nil, fmt.Errorf("unsupported expression type %T", expr)
	}
}
