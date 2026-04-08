package builtins

import (
	"os"

	"molt/internal/runtime"
)

func readFileBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	path, ok := args[0].(*runtime.StringValue)
	if !ok {
		return nil, runtimeErrorf(ctx.CallSpan, "read_file expects string path, got %q", args[0].TypeName())
	}

	if path.Value == "" {
		return nil, runtimeErrorf(ctx.CallSpan, "read_file path cannot be empty")
	}

	reader := ctx.ReadFile
	if reader == nil {
		reader = os.ReadFile
	}

	data, err := reader(path.Value)
	if err != nil {
		return nil, runtimeErrorf(ctx.CallSpan, "read_file failed for %q: %v", path.Value, err)
	}

	return &runtime.StringValue{Value: string(data)}, nil
}

func writeFileBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	path, ok := args[0].(*runtime.StringValue)
	if !ok {
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"write_file expects string path as first argument, got %q",
			args[0].TypeName(),
		)
	}

	text, ok := args[1].(*runtime.StringValue)
	if !ok {
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"write_file expects string text as second argument, got %q",
			args[1].TypeName(),
		)
	}

	if path.Value == "" {
		return nil, runtimeErrorf(ctx.CallSpan, "write_file path cannot be empty")
	}

	writer := ctx.WriteFile
	if writer == nil {
		writer = defaultWriteFile
	}

	if err := writer(path.Value, []byte(text.Value)); err != nil {
		return nil, runtimeErrorf(ctx.CallSpan, "write_file failed for %q: %v", path.Value, err)
	}

	return runtime.Nil, nil
}
