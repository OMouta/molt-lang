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

[`examples/variant_gallery.molt`](../examples/variant_gallery.molt)
: A fuller end-to-end program combining functions, quotes, mutations, lists, `push`, `type`, `len`, and `eval`.

Expected output:

```txt
[6, 7, "code"]
```

## Running Examples

```powershell
go run ./cmd/molt ./examples/basic_mutation.molt
go run ./cmd/molt ./examples/compare_worlds.molt
go run ./cmd/molt ./examples/variant_gallery.molt
```
