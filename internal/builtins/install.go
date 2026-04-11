package builtins

import "molt/internal/runtime"

type bindingSpec struct {
	Name  string
	Arity int
	Impl  func(*runtime.CallContext, []runtime.Value) (runtime.Value, error)
	Doc   SymbolDoc
}

type moduleSpec struct {
	Path     string
	Summary  string
	Bindings []bindingSpec
}

var moduleSpecs = []moduleSpec{
	{
		Path:    "std:meta",
		Summary: "Metaprogramming, reflection, and conversion helpers.",
		Bindings: []bindingSpec{
			{
				Name:  "eval",
				Arity: 1,
				Impl:  evalBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"eval(code)"},
					Summary:    "Execute a code value and return its result.",
					Detail:     "The argument must be a `code` value produced by `@{ ... }`. Each call re-runs the captured AST from scratch in a fresh frame rooted in the captured environment.",
				},
			},
			{
				Name:  "type",
				Arity: 1,
				Impl:  typeBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"type(x)"},
					Summary:    "Return the runtime type name of a value as a string.",
					Detail:     "Possible values: `\"number\"`, `\"string\"`, `\"boolean\"`, `\"nil\"`, `\"list\"`, `\"record\"`, `\"error\"`, `\"function\"`, `\"native-function\"`, `\"code\"`, `\"mutation\"`.",
				},
			},
			{
				Name:  "show",
				Arity: 1,
				Impl:  showBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"show(x)"},
					Summary:    "Return a stable, source-like display string for any value.",
				},
			},
			{
				Name:  "to_string",
				Arity: 1,
				Impl:  toStringBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"to_string(x)"},
					Summary:    "Convert a value to its string representation.",
					Detail:     "Strings are returned as-is. Other values use their readable display form.",
				},
			},
			{
				Name:  "to_number",
				Arity: 1,
				Impl:  toNumberBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"to_number(x)"},
					Summary:    "Parse a numeric string or return a number unchanged.",
					Detail:     "Raises a runtime diagnostic if the string cannot be parsed as a number.",
				},
			},
		},
	},
	{
		Path:    "std:errors",
		Summary: "First-class error construction and raising.",
		Bindings: []bindingSpec{
			{
				Name:  "error",
				Arity: -1,
				Impl:  errorBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"error(message)", "error(message, data)"},
					Summary:    "Build a first-class error value.",
					Detail:     "Error values have type `\"error\"` and expose `message` plus an optional `data` field. They can be stored, passed around, and inspected like any other value. Use `throw(err)` to raise one as a runtime failure.",
				},
			},
			{
				Name:  "throw",
				Arity: 1,
				Impl:  throwBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"throw(err)"},
					Summary:    "Raise an error value as a runtime failure.",
					Detail:     "Requires a value of type `\"error\"`. Uncaught throws become runtime diagnostics. Thrown errors with `data` add a diagnostic note showing the payload. Loop control (`break`, `continue`) is not catchable.",
				},
			},
		},
	},
	{
		Path:    "std:cli",
		Summary: "Command-line process context.",
		Bindings: []bindingSpec{
			{
				Name:  "args",
				Arity: 0,
				Impl:  argsBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"args()"},
					Summary:    "Return a list of command-line arguments passed after the script path.",
					Detail:     "Returns `[]` in REPL mode.",
				},
			},
		},
	},
	{
		Path:    "std:collections",
		Summary: "Generic collection helpers for lists, strings, records, and errors.",
		Bindings: []bindingSpec{
			{
				Name:  "len",
				Arity: 1,
				Impl:  lenBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"len(x)"},
					Summary:    "Return the element count of a list, string, record, or error.",
					Detail:     "For strings, counts Unicode code points. For records and errors, counts fields.",
				},
			},
			{
				Name:  "push",
				Arity: 2,
				Impl:  pushBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"push(list, value)"},
					Summary:    "Append a value to a list in place and return the same list.",
				},
			},
			{
				Name:  "contains",
				Arity: 2,
				Impl:  containsBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"contains(collection, needle)"},
					Summary:    "Return `true` if needle appears in a string or if a record or error has a field with that name.",
				},
			},
			{
				Name:  "keys",
				Arity: 1,
				Impl:  keysBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"keys(record)"},
					Summary:    "Return a list of field names from a record or error in display order.",
				},
			},
			{
				Name:  "values",
				Arity: 1,
				Impl:  valuesBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"values(record)"},
					Summary:    "Return a list of field values from a record or error in the same order as `keys`.",
				},
			},
			{
				Name:  "range",
				Arity: -1,
				Impl:  rangeBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"range(end)", "range(start, end)"},
					Summary:    "Build an ascending list of integers with an exclusive end bound.",
				},
			},
			{
				Name:  "map",
				Arity: 2,
				Impl:  mapBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"map(list, fn)"},
					Summary:    "Return a new list by applying a callback to each element.",
					Detail:     "The callback may accept `(value)` or `(value, index)`.",
				},
			},
			{
				Name:  "filter",
				Arity: 2,
				Impl:  filterBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"filter(list, fn)"},
					Summary:    "Return a new list containing only elements for which the callback returns `true`.",
					Detail:     "The callback may accept `(value)` or `(value, index)`.",
				},
			},
		},
	},
	{
		Path:    "std:strings",
		Summary: "String manipulation helpers.",
		Bindings: []bindingSpec{
			{
				Name:  "split",
				Arity: 2,
				Impl:  splitBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"split(text, separator)"},
					Summary:    "Split a string into a list of strings at each occurrence of separator.",
				},
			},
			{
				Name:  "join",
				Arity: 2,
				Impl:  joinBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"join(parts, separator)"},
					Summary:    "Join a list of strings into a single string with separator between each part.",
				},
			},
			{
				Name:  "trim",
				Arity: 1,
				Impl:  trimBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"trim(text)"},
					Summary:    "Remove leading and trailing whitespace from a string.",
				},
			},
			{
				Name:  "lines",
				Arity: 1,
				Impl:  linesBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"lines(text)"},
					Summary:    "Split a string into a list of lines.",
					Detail:     "`\\n`, `\\r\\n`, and `\\r` are all treated as line breaks. A single trailing newline does not produce an extra empty string.",
				},
			},
			{
				Name:  "replace",
				Arity: 3,
				Impl:  replaceBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"replace(text, old, new)"},
					Summary:    "Replace every occurrence of `old` inside `text` with `new`.",
				},
			},
		},
	},
	{
		Path:    "std:io",
		Summary: "Console and file I/O.",
		Bindings: []bindingSpec{
			{
				Name:  "read_file",
				Arity: 1,
				Impl:  readFileBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"read_file(path)"},
					Summary:    "Read a file from disk and return its contents as a string.",
					Detail:     "The path must be a non-empty string.",
				},
			},
			{
				Name:  "write_file",
				Arity: 2,
				Impl:  writeFileBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"write_file(path, text)"},
					Summary:    "Write a string to a file on disk, replacing any existing contents.",
					Detail:     "Both `path` and `text` must be strings; `path` must be non-empty.",
				},
			},
			{
				Name:  "input",
				Arity: 0,
				Impl:  inputBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"input()"},
					Summary:    "Read one line from standard input and return it without the trailing newline.",
					Detail:     "Returns `\"\"` at end of input.",
				},
			},
			{
				Name:  "print",
				Arity: 1,
				Impl:  printBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"print(x)"},
					Summary:    "Write a value to standard output followed by a newline. Returns `nil`.",
					Detail:     "Strings are printed without quotes. Other values use their display form.",
				},
			},
			{
				Name:  "stdin",
				Arity: 0,
				Impl:  stdinBuiltin,
				Doc: SymbolDoc{
					Signatures: []string{"stdin()"},
					Summary:    "Read all of standard input and return it as a string.",
					Detail:     "Returns `\"\"` if stdin has already been consumed.",
				},
			},
		},
	},
}

func ModuleDocs() []ModuleDoc {
	docs := make([]ModuleDoc, 0, len(moduleSpecs))
	for _, module := range moduleSpecs {
		symbols := make([]SymbolDoc, 0, len(module.Bindings))
		for _, binding := range module.Bindings {
			symbols = append(symbols, binding.Doc)
		}

		docs = append(docs, ModuleDoc{
			Path:    module.Path,
			Summary: module.Summary,
			Symbols: symbols,
		})
	}

	return docs
}

func IsModule(path string) bool {
	_, ok := findModule(path)
	return ok
}

func ModuleBindings(path string) ([]runtime.Binding, bool) {
	module, ok := findModule(path)
	if !ok {
		return nil, false
	}

	bindings := make([]runtime.Binding, 0, len(module.Bindings))
	for _, binding := range module.Bindings {
		bindings = append(bindings, runtime.Binding{
			Name:  binding.Name,
			Value: nativeFunction(binding),
		})
	}

	return bindings, true
}

func findModule(path string) (moduleSpec, bool) {
	for _, module := range moduleSpecs {
		if module.Path == path {
			return module, true
		}
	}

	return moduleSpec{}, false
}

func nativeFunction(binding bindingSpec) *runtime.NativeFunctionValue {
	return &runtime.NativeFunctionValue{
		FunctionName: binding.Name,
		Arity:        binding.Arity,
		Impl:         binding.Impl,
	}
}
