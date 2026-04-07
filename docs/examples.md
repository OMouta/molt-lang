# Examples

The `examples/` folder contains runnable programs that are verified by the test suite and exercised by the CLI smoke checks.

## Files

[`examples/basic_mutation.molt`](../examples/basic_mutation.molt)
: Basic function mutation. Prints the original and mutated results.

Expected output:

```txt
5
6
```

[`examples/compare_worlds.molt`](../examples/compare_worlds.molt)
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

[`examples/records.molt`](../examples/records.molt)
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

[`examples/variant_gallery.molt`](../examples/variant_gallery.molt)
: A fuller end-to-end program combining functions, quotes, mutations, lists, `push`, `type`, `len`, and `eval`.

Expected output:

```txt
[6, 7, "code"]
```

[`examples/guessing_game.molt`](../examples/guessing_game.molt)
: Interactive example using `input()` and `to_number()` to keep asking for guesses until the fixed secret number is found.

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
go run ./cmd/molt ./examples/basic_mutation.molt
go run ./cmd/molt ./examples/compare_worlds.molt
go run ./cmd/molt ./examples/import_export/main.molt
go run ./cmd/molt ./examples/records.molt
go run ./cmd/molt ./examples/variant_gallery.molt
go run ./cmd/molt ./examples/guessing_game.molt
```
