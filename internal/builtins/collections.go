package builtins

import (
	"strings"

	"molt/internal/runtime"
)

func lenBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	switch value := args[0].(type) {
	case *runtime.ListValue:
		return &runtime.NumberValue{Value: float64(len(value.Elements))}, nil
	case *runtime.StringValue:
		return &runtime.NumberValue{Value: float64(len([]rune(value.Value)))}, nil
	case *runtime.RecordValue:
		return &runtime.NumberValue{Value: float64(value.Len())}, nil
	case *runtime.ErrorValue:
		return &runtime.NumberValue{Value: float64(value.Len())}, nil
	default:
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"len expects list, string, record, or error, got %q",
			args[0].TypeName(),
		)
	}
}

func pushBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	list, ok := args[0].(*runtime.ListValue)
	if !ok {
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"push expects list as first argument, got %q",
			args[0].TypeName(),
		)
	}

	list.Elements = append(list.Elements, args[1])
	return list, nil
}

func containsBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	switch value := args[0].(type) {
	case *runtime.StringValue:
		needle, ok := args[1].(*runtime.StringValue)
		if !ok {
			return nil, runtimeErrorf(
				ctx.CallSpan,
				"contains expects string as second argument, got %q",
				args[1].TypeName(),
			)
		}

		return &runtime.BooleanValue{Value: strings.Contains(value.Value, needle.Value)}, nil
	case *runtime.RecordValue:
		key, ok := args[1].(*runtime.StringValue)
		if !ok {
			return nil, runtimeErrorf(
				ctx.CallSpan,
				"contains expects string key as second argument for records, got %q",
				args[1].TypeName(),
			)
		}

		_, exists := value.GetField(key.Value)
		return &runtime.BooleanValue{Value: exists}, nil
	case *runtime.ErrorValue:
		key, ok := args[1].(*runtime.StringValue)
		if !ok {
			return nil, runtimeErrorf(
				ctx.CallSpan,
				"contains expects string key as second argument for errors, got %q",
				args[1].TypeName(),
			)
		}

		_, exists := value.GetField(key.Value)
		return &runtime.BooleanValue{Value: exists}, nil
	default:
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"contains expects string, record, or error as first argument, got %q",
			args[0].TypeName(),
		)
	}
}

func keysBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	var keys []string
	switch value := args[0].(type) {
	case *runtime.RecordValue:
		keys = value.Keys()
	case *runtime.ErrorValue:
		keys = value.Keys()
	default:
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"keys expects record or error, got %q",
			args[0].TypeName(),
		)
	}

	elements := make([]runtime.Value, 0, len(keys))
	for _, key := range keys {
		elements = append(elements, &runtime.StringValue{Value: key})
	}

	return &runtime.ListValue{Elements: elements}, nil
}

func valuesBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	switch value := args[0].(type) {
	case *runtime.RecordValue:
		return &runtime.ListValue{Elements: value.Values()}, nil
	case *runtime.ErrorValue:
		return &runtime.ListValue{Elements: value.Values()}, nil
	default:
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"values expects record or error, got %q",
			args[0].TypeName(),
		)
	}
}

func rangeBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	if len(args) != 1 && len(args) != 2 {
		return nil, runtimeErrorf(ctx.CallSpan, "range expects 1 or 2 arguments but got %d", len(args))
	}

	start := 0
	end, err := integerArgument("range", args[0], 0, ctx.CallSpan)
	if err != nil {
		return nil, err
	}

	if len(args) == 2 {
		start, err = integerArgument("range", args[0], 0, ctx.CallSpan)
		if err != nil {
			return nil, err
		}

		end, err = integerArgument("range", args[1], 1, ctx.CallSpan)
		if err != nil {
			return nil, err
		}
	}

	if end <= start {
		return &runtime.ListValue{Elements: nil}, nil
	}

	elements := make([]runtime.Value, 0, end-start)
	for i := start; i < end; i++ {
		elements = append(elements, &runtime.NumberValue{Value: float64(i)})
	}

	return &runtime.ListValue{Elements: elements}, nil
}

func mapBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	list, ok := args[0].(*runtime.ListValue)
	if !ok {
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"map expects list as first argument, got %q",
			args[0].TypeName(),
		)
	}

	callbackArity, err := callbackArity("map", args[1], ctx.CallSpan)
	if err != nil {
		return nil, err
	}

	elements := make([]runtime.Value, 0, len(list.Elements))
	for index, element := range list.Elements {
		callbackArgs := []runtime.Value{element}
		if callbackArity == 2 {
			callbackArgs = append(callbackArgs, &runtime.NumberValue{Value: float64(index)})
		}

		value, err := invokeCallback(ctx, args[1], callbackArgs)
		if err != nil {
			return nil, err
		}

		elements = append(elements, value)
	}

	return &runtime.ListValue{Elements: elements}, nil
}

func filterBuiltin(ctx *runtime.CallContext, args []runtime.Value) (runtime.Value, error) {
	list, ok := args[0].(*runtime.ListValue)
	if !ok {
		return nil, runtimeErrorf(
			ctx.CallSpan,
			"filter expects list as first argument, got %q",
			args[0].TypeName(),
		)
	}

	callbackArity, err := callbackArity("filter", args[1], ctx.CallSpan)
	if err != nil {
		return nil, err
	}

	elements := make([]runtime.Value, 0, len(list.Elements))
	for index, element := range list.Elements {
		callbackArgs := []runtime.Value{element}
		if callbackArity == 2 {
			callbackArgs = append(callbackArgs, &runtime.NumberValue{Value: float64(index)})
		}

		value, err := invokeCallback(ctx, args[1], callbackArgs)
		if err != nil {
			return nil, err
		}

		boolean, ok := value.(*runtime.BooleanValue)
		if !ok {
			return nil, runtimeErrorf(
				ctx.CallSpan,
				"filter callback must return boolean, got %q",
				value.TypeName(),
			)
		}

		if boolean.Value {
			elements = append(elements, element)
		}
	}

	return &runtime.ListValue{Elements: elements}, nil
}
