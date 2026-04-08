package builtins

import (
	"molt/internal/runtime"
	"molt/internal/source"
)

type ThrownError struct {
	Value *runtime.ErrorValue
	Span  source.Span
}

func (e ThrownError) Error() string {
	if e.Value == nil {
		return "throw"
	}

	return e.Value.Message
}

func AsThrown(err error) (ThrownError, bool) {
	thrown, ok := err.(ThrownError)
	return thrown, ok
}

func throwBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	errValue, ok := args[0].(*runtime.ErrorValue)
	if !ok {
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"throw expects error value, got %q",
			args[0].TypeName(),
		)
	}

	return nil, ThrownError{
		Value: errValue,
		Span:  ctx.CallSpan,
	}
}

func errorBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	if len(args) != 1 && len(args) != 2 {
		return nil, runtimeErrorf(ctx.CallSpan, "error expects 1 or 2 arguments but got %d", len(args))
	}

	message, ok := args[0].(*runtime.StringValue)
	if !ok {
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"error expects string message as first argument, got %q",
			args[0].TypeName(),
		)
	}

	if len(args) == 1 {
		return runtime.NewErrorValue(message.Value, nil, false), nil
	}

	return runtime.NewErrorValue(message.Value, args[1], true), nil
}
