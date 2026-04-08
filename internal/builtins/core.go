package builtins

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"molt/internal/runtime"
)

func evalBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	code, ok := args[0].(*runtime.CodeValue)
	if !ok {
		return nil, runtimeErrorf(ctx.CallSpan, "eval expects code value, got %q", args[0].TypeName())
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

func showBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	return &runtime.StringValue{Value: runtime.ShowValue(args[0])}, nil
}

func toStringBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	switch value := args[0].(type) {
	case *runtime.StringValue:
		return &runtime.StringValue{Value: value.Value}, nil
	case *runtime.NumberValue:
		return &runtime.StringValue{Value: runtime.ShowValue(value)}, nil
	case *runtime.BooleanValue:
		return &runtime.StringValue{Value: runtime.ShowValue(value)}, nil
	case runtime.NilValue:
		return &runtime.StringValue{Value: "nil"}, nil
	default:
		return &runtime.StringValue{Value: runtime.ShowValue(args[0])}, nil
	}
}

func toNumberBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	switch value := args[0].(type) {
	case *runtime.NumberValue:
		return &runtime.NumberValue{Value: value.Value}, nil
	case *runtime.StringValue:
		text := strings.TrimSpace(value.Value)
		number, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, runtimeErrorf(ctx.CallSpan, "to_number could not parse %q", value.Value)
		}

		return &runtime.NumberValue{Value: number}, nil
	default:
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"to_number expects number or string, got %q",
			args[0].TypeName(),
		)
	}
}

func printBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	text := runtime.ShowValue(args[0])
	if value, ok := args[0].(*runtime.StringValue); ok {
		text = value.Value
	}

	if _, err := fmt.Fprintln(outputWriter(ctx.Output), text); err != nil {
		return nil, err
	}

	return runtime.Nil, nil
}

func stdinBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	text, err := io.ReadAll(inputReader(ctx.Input))
	if err != nil {
		return nil, runtimeErrorf(ctx.CallSpan, "stdin failed: %v", err)
	}

	return &runtime.StringValue{Value: string(text)}, nil
}

func inputBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	reader := bufferedInputReader(ctx.Input)
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")
			return &runtime.StringValue{Value: line}, nil
		}

		return nil, runtimeErrorf(ctx.CallSpan, "input failed: %v", err)
	}

	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return &runtime.StringValue{Value: line}, nil
}
