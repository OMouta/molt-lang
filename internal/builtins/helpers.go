package builtins

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"

	"molt/internal/diagnostic"
	"molt/internal/runtime"
	"molt/internal/source"
)

func callbackArity(name string, callback runtime.Value, span source.Span) (int, error) {
	switch value := callback.(type) {
	case *runtime.UserFunctionValue:
		if len(value.Parameters) == 1 || len(value.Parameters) == 2 {
			return len(value.Parameters), nil
		}
		return 0, runtimeErrorf(span, "%s callback must accept 1 or 2 arguments, got %d", name, len(value.Parameters))
	case *runtime.NativeFunctionValue:
		if value.Arity == 1 || value.Arity == 2 {
			return value.Arity, nil
		}
		return 0, runtimeErrorf(span, "%s callback must accept 1 or 2 arguments, got %d", name, value.Arity)
	default:
		return 0, runtimeErrorf(span, "%s expects function as second argument, got %q", name, callback.TypeName())
	}
}

func invokeCallback(ctx *runtime.CallContext, callback runtime.Value, args []runtime.Value) (runtime.Value, error) {
	if ctx.Invoke == nil {
		return nil, fmt.Errorf("missing callback invoker")
	}

	return ctx.Invoke(callback, args, ctx.Environment, ctx.CallSpan)
}

func integerArgument(name string, value runtime.Value, position int, span source.Span) (int, error) {
	number, ok := value.(*runtime.NumberValue)
	if !ok {
		return 0, runtimeErrorf(span, "%s expects number at argument %d, got %q", name, position+1, value.TypeName())
	}

	if math.Trunc(number.Value) != number.Value {
		return 0, runtimeErrorf(span, "%s expects integer at argument %d, got %v", name, position+1, number.Value)
	}

	return int(number.Value), nil
}

func inputReader(reader io.Reader) io.Reader {
	if reader != nil {
		return reader
	}

	return os.Stdin
}

func bufferedInputReader(reader io.Reader) *bufio.Reader {
	switch value := reader.(type) {
	case *bufio.Reader:
		return value
	case nil:
		return bufio.NewReader(os.Stdin)
	default:
		return bufio.NewReader(value)
	}
}

func defaultWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func outputWriter(writer io.Writer) io.Writer {
	if writer != nil {
		return writer
	}

	return os.Stdout
}

func runtimeErrorf(span source.Span, format string, args ...any) error {
	return diagnostic.NewRuntimeError(fmt.Sprintf(format, args...), span)
}
