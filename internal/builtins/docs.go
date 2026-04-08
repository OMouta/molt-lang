package builtins

// Doc describes a single built-in function available in molt programs.
// The docgen tool reads AllDocs to generate the Builtins reference page.
type Doc struct {
	// Signatures lists every call form (some builtins support multiple arities).
	Signatures []string
	// Summary is a one-line description shown in the overview table.
	Summary string
	// Detail provides additional explanation. Optional. Supports inline Markdown.
	Detail string
}

// AllDocs contains documentation for every built-in function in registration order.
// Add a new entry here whenever a new builtin is added to install.go.
var AllDocs = []Doc{
	{
		Signatures: []string{"eval(code)"},
		Summary:    "Execute a code value and return its result.",
		Detail:     "The argument must be a `code` value produced by `@{ ... }`. Each call re-runs the captured AST from scratch in a fresh frame rooted in the captured environment.",
	},
	{
		Signatures: []string{"type(x)"},
		Summary:    "Return the runtime type name of a value as a string.",
		Detail:     "Possible values: `\"number\"`, `\"string\"`, `\"boolean\"`, `\"nil\"`, `\"list\"`, `\"record\"`, `\"error\"`, `\"function\"`, `\"native-function\"`, `\"code\"`, `\"mutation\"`.",
	},
	{
		Signatures: []string{"error(message)", "error(message, data)"},
		Summary:    "Build a first-class error value.",
		Detail:     "Error values have type `\"error\"` and expose `message` plus an optional `data` field. They can be stored, passed around, and inspected like any other value. Use `throw(err)` to raise one as a runtime failure.",
	},
	{
		Signatures: []string{"throw(err)"},
		Summary:    "Raise an error value as a runtime failure.",
		Detail:     "Requires a value of type `\"error\"`. Uncaught throws become runtime diagnostics. Thrown errors with `data` add a diagnostic note showing the payload. Loop control (`break`, `continue`) is not catchable.",
	},
	{
		Signatures: []string{"args()"},
		Summary:    "Return a list of command-line arguments passed after the script path.",
		Detail:     "Returns `[]` in REPL mode.",
	},
	{
		Signatures: []string{"len(x)"},
		Summary:    "Return the element count of a list, string, record, or error.",
		Detail:     "For strings, counts Unicode code points. For records and errors, counts fields.",
	},
	{
		Signatures: []string{"push(list, value)"},
		Summary:    "Append a value to a list in place and return the same list.",
	},
	{
		Signatures: []string{"split(text, separator)"},
		Summary:    "Split a string into a list of strings at each occurrence of separator.",
	},
	{
		Signatures: []string{"join(parts, separator)"},
		Summary:    "Join a list of strings into a single string with separator between each part.",
	},
	{
		Signatures: []string{"trim(text)"},
		Summary:    "Remove leading and trailing whitespace from a string.",
	},
	{
		Signatures: []string{"lines(text)"},
		Summary:    "Split a string into a list of lines.",
		Detail:     "`\\n`, `\\r\\n`, and `\\r` are all treated as line breaks. A single trailing newline does not produce an extra empty string.",
	},
	{
		Signatures: []string{"replace(text, old, new)"},
		Summary:    "Replace every occurrence of `old` inside `text` with `new`.",
	},
	{
		Signatures: []string{"contains(collection, needle)"},
		Summary:    "Return `true` if needle appears in a string or if a record or error has a field with that name.",
	},
	{
		Signatures: []string{"keys(record)"},
		Summary:    "Return a list of field names from a record or error in display order.",
	},
	{
		Signatures: []string{"values(record)"},
		Summary:    "Return a list of field values from a record or error in the same order as `keys`.",
	},
	{
		Signatures: []string{"range(end)", "range(start, end)"},
		Summary:    "Build an ascending list of integers with an exclusive end bound.",
	},
	{
		Signatures: []string{"map(list, fn)"},
		Summary:    "Return a new list by applying a callback to each element.",
		Detail:     "The callback may accept `(value)` or `(value, index)`.",
	},
	{
		Signatures: []string{"filter(list, fn)"},
		Summary:    "Return a new list containing only elements for which the callback returns `true`.",
		Detail:     "The callback may accept `(value)` or `(value, index)`.",
	},
	{
		Signatures: []string{"show(x)"},
		Summary:    "Return a stable, source-like display string for any value.",
	},
	{
		Signatures: []string{"read_file(path)"},
		Summary:    "Read a file from disk and return its contents as a string.",
		Detail:     "The path must be a non-empty string.",
	},
	{
		Signatures: []string{"write_file(path, text)"},
		Summary:    "Write a string to a file on disk, replacing any existing contents.",
		Detail:     "Both `path` and `text` must be strings; `path` must be non-empty.",
	},
	{
		Signatures: []string{"input()"},
		Summary:    "Read one line from standard input and return it without the trailing newline.",
		Detail:     "Returns `\"\"` at end of input.",
	},
	{
		Signatures: []string{"to_string(x)"},
		Summary:    "Convert a value to its string representation.",
		Detail:     "Strings are returned as-is. Other values use their readable display form.",
	},
	{
		Signatures: []string{"to_number(x)"},
		Summary:    "Parse a numeric string or return a number unchanged.",
		Detail:     "Raises a runtime diagnostic if the string cannot be parsed as a number.",
	},
	{
		Signatures: []string{"print(x)"},
		Summary:    "Write a value to standard output followed by a newline. Returns `nil`.",
		Detail:     "Strings are printed without quotes. Other values use their display form.",
	},
	{
		Signatures: []string{"stdin()"},
		Summary:    "Read all of standard input and return it as a string.",
		Detail:     "Returns `\"\"` if stdin has already been consumed.",
	},
}
