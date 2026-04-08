package builtins

import (
	"strings"

	"molt/internal/runtime"
)

func splitBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	value, ok := args[0].(*runtime.StringValue)
	if !ok {
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"split expects string as first argument, got %q",
			args[0].TypeName(),
		)
	}

	separator, ok := args[1].(*runtime.StringValue)
	if !ok {
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"split expects string as second argument, got %q",
			args[1].TypeName(),
		)
	}

	parts := strings.Split(value.Value, separator.Value)
	elements := make([]runtime.Value, 0, len(parts))
	for _, part := range parts {
		elements = append(elements, &runtime.StringValue{Value: part})
	}

	return &runtime.ListValue{Elements: elements}, nil
}

func joinBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	list, ok := args[0].(*runtime.ListValue)
	if !ok {
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"join expects list as first argument, got %q",
			args[0].TypeName(),
		)
	}

	separator, ok := args[1].(*runtime.StringValue)
	if !ok {
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"join expects string as second argument, got %q",
			args[1].TypeName(),
		)
	}

	parts := make([]string, 0, len(list.Elements))
	for i, element := range list.Elements {
		item, ok := element.(*runtime.StringValue)
		if !ok {
			return nil, runtimeErrorf(
				ctx.CallSpan,
				"join expects list of strings, but element %d has type %q",
				i,
				element.TypeName(),
			)
		}

		parts = append(parts, item.Value)
	}

	return &runtime.StringValue{Value: strings.Join(parts, separator.Value)}, nil
}

func trimBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	value, ok := args[0].(*runtime.StringValue)
	if !ok {
		return nil, runtimeErrorf(ctx.CallSpan, "trim expects string, got %q", args[0].TypeName())
	}

	return &runtime.StringValue{Value: strings.TrimSpace(value.Value)}, nil
}

func linesBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	value, ok := args[0].(*runtime.StringValue)
	if !ok {
		return nil, runtimeErrorf(ctx.CallSpan, "lines expects string, got %q", args[0].TypeName())
	}

	normalized := strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(value.Value)
	parts := strings.Split(normalized, "\n")
	if len(parts) == 1 && parts[0] == "" {
		return &runtime.ListValue{Elements: nil}, nil
	}

	if strings.HasSuffix(normalized, "\n") {
		parts = parts[:len(parts)-1]
	}

	elements := make([]runtime.Value, 0, len(parts))
	for _, part := range parts {
		elements = append(elements, &runtime.StringValue{Value: part})
	}

	return &runtime.ListValue{Elements: elements}, nil
}

func replaceBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	value, ok := args[0].(*runtime.StringValue)
	if !ok {
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"replace expects string as first argument, got %q",
			args[0].TypeName(),
		)
	}

	old, ok := args[1].(*runtime.StringValue)
	if !ok {
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"replace expects string as second argument, got %q",
			args[1].TypeName(),
		)
	}

	newValue, ok := args[2].(*runtime.StringValue)
	if !ok {
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"replace expects string as third argument, got %q",
			args[2].TypeName(),
		)
	}

	return &runtime.StringValue{Value: strings.ReplaceAll(value.Value, old.Value, newValue.Value)}, nil
}
