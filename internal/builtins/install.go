package builtins

import "molt/internal/runtime"

func Install(env *runtime.Environment) {
	install(env, "eval", 1, evalBuiltin)
	install(env, "type", 1, typeBuiltin)
	install(env, "error", -1, errorBuiltin)
	install(env, "throw", 1, throwBuiltin)
	install(env, "args", 0, argsBuiltin)
	install(env, "len", 1, lenBuiltin)
	install(env, "push", 2, pushBuiltin)
	install(env, "split", 2, splitBuiltin)
	install(env, "join", 2, joinBuiltin)
	install(env, "trim", 1, trimBuiltin)
	install(env, "lines", 1, linesBuiltin)
	install(env, "replace", 3, replaceBuiltin)
	install(env, "contains", 2, containsBuiltin)
	install(env, "keys", 1, keysBuiltin)
	install(env, "values", 1, valuesBuiltin)
	install(env, "range", -1, rangeBuiltin)
	install(env, "map", 2, mapBuiltin)
	install(env, "filter", 2, filterBuiltin)
	install(env, "show", 1, showBuiltin)
	install(env, "read_file", 1, readFileBuiltin)
	install(env, "write_file", 2, writeFileBuiltin)
	install(env, "input", 0, inputBuiltin)
	install(env, "to_string", 1, toStringBuiltin)
	install(env, "to_number", 1, toNumberBuiltin)
	install(env, "print", 1, printBuiltin)
	install(env, "stdin", 0, stdinBuiltin)
}

func install(env *runtime.Environment, name string, arity int, impl func(*runtime.CallContext, []runtime.Value) (runtime.Value, error)) {
	if _, ok := env.Get(name); ok {
		return
	}

	env.Define(name, &runtime.NativeFunctionValue{
		FunctionName: name,
		Arity:        arity,
		Impl:         impl,
	})
}
