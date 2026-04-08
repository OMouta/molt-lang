# Examples

The `examples/` folder contains runnable programs that are verified by the test suite and exercised by the CLI smoke checks.

## Files

[`examples/basic/basic_mutation.molt`](../examples/basic/basic_mutation.molt)
: Basic function mutation. Prints the original and mutated results.

Expected output:

```txt
5
6
```

[`examples/others/colors.molt`](../examples/others/colors.molt)
: Prints the same message using several ANSI terminal colors through string escape sequences.

[`examples/basic/compare_worlds.molt`](../examples/basic/compare_worlds.molt)
: Evaluates the same quoted code before and after a mutation.

Expected output:

```txt
5
6
```

[`examples/import_export/main.molt`](../examples/import_export/main.molt)
: Imports explicitly exported module bindings from a neighboring file and calls an exported function that still closes over private module state.

Expected output:

```txt
40
42
```

[`examples/basic/records.molt`](../examples/basic/records.molt)
: Builds a record value in normal user code, reads nested fields, and demonstrates the record helper builtins while preserving field order.

Expected output:

```txt
record { name: "molt", stats: record { runs: 3 } }
molt
3
["name", "stats"]
["molt", record { runs: 3 }]
true
2
record
```

[`examples/errors/error_values.molt`](../examples/errors/error_values.molt)
: Constructs first-class error values, inspects their fields, and shows their stable display form without aborting execution.

Expected output:

```txt
error
missing file
note.txt
["message", "data"]
error {
  message: "missing file",
  data: record { path: "note.txt" }
}
```

[`examples/errors/try_catch.molt`](../examples/errors/try_catch.molt)
: Handles an imported thrown error, a normal runtime failure, and a direct `throw(...)` using `try ... catch err -> ...`.

Expected output:

```txt
["helper failed", "import"]
len expects list, string, record, or error, got "number"
name
```

[`examples/errors/throw_error.molt`](../examples/errors/throw_error.molt)
: Raises an error value intentionally with `throw(...)`, demonstrating how uncaught throws turn into runtime diagnostics with preserved throw-site spans and error-data notes.

Expected diagnostic excerpt:

```txt
runtime error: config file not found
note: error data: record { path: "settings.json" }
```

[`examples/basic/variant_gallery.molt`](../examples/basic/variant_gallery.molt)
: A fuller end-to-end program combining functions, quotes, mutations, lists, `push`, `type`, `len`, and `eval`.

Expected output:

```txt
[6, 7, "code"]
```

[`examples/loops/while_loop.molt`](../examples/loops/while_loop.molt)
: Increments a counter with a `while` loop until the condition becomes false.

Expected output:

```txt
3
```

[`examples/loops/for_loop.molt`](../examples/loops/for_loop.molt)
: Iterates over a list and a string with `for ... in ...`, showing accumulation and character collection.

Expected output:

```txt
6
["o", "k"]
```

[`examples/loops/break_continue.molt`](../examples/loops/break_continue.molt)
: Uses `continue` to skip one item and `break` to stop the loop early once the target is found.

Expected output:

```txt
[1, 3]
```

[`examples/others/guessing_game.molt`](../examples/others/guessing_game.molt)
: Interactive example using `while`, `input()`, and `to_number()` to keep asking for guesses until the fixed secret number is found.

Example session:

```txt
guess a number between 1 and 10
3
too low
guess a number between 1 and 10
9
too high
guess a number between 1 and 10
7
you got it!
```

## Running Examples

```powershell
go run ./cmd/molt ./examples/basic/basic_mutation.molt
go run ./cmd/molt ./examples/others/colors.molt
go run ./cmd/molt ./examples/basic/compare_worlds.molt
go run ./cmd/molt ./examples/import_export/main.molt
go run ./cmd/molt ./examples/basic/records.molt
go run ./cmd/molt ./examples/errors/error_values.molt
go run ./cmd/molt ./examples/errors/try_catch.molt
go run ./cmd/molt ./examples/errors/throw_error.molt
go run ./cmd/molt ./examples/basic/variant_gallery.molt
go run ./cmd/molt ./examples/loops/while_loop.molt
go run ./cmd/molt ./examples/loops/for_loop.molt
go run ./cmd/molt ./examples/loops/break_continue.molt
go run ./cmd/molt ./examples/others/guessing_game.molt
```
