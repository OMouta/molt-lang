# @molt

`@molt` is a small expression-oriented language for quoting code, rewriting it with first-class mutations, and executing the result explicitly.

Core primitives:

```txt
@{ ... }   # quote code
~{ ... }   # define a mutation
eval(x)    # run quoted code
```

> `@molt` is pre-alpha software. The language, implementation, and documentation are all in early stages of development. Expect breaking changes and incomplete features. Feedback and contributions are very welcome!

## Quick Start

Requirements:

- Go `1.26`

Build and test:

```powershell
go build ./...
go test ./...
go vet ./...
```

Run a program:

```powershell
go run ./cmd/molt ./examples/basic_mutation.molt
```

Format the codebase:

```powershell
gofmt -w .
```

Convenience wrappers are also available:

```powershell
./dev.ps1 build
./dev.ps1 test
./dev.ps1 lint
./dev.ps1 format
```

## Documentation

- [Docs Index](docs/README.md)
- [Getting Started](docs/getting-started.md)
- [Language Guide](docs/language-guide.md)
- [Examples](docs/examples.md)
- [Editor Support](docs/editor-support.md)
- [Contributing](docs/contributing.md)
- [Architecture](docs/architecture.md)

## Project Layout

- `cmd/molt`: CLI entrypoint
- `internal/ast`: AST types
- `internal/diagnostic`: shared diagnostics and rendering
- `internal/evaluator`: evaluation and builtin execution
- `internal/lexer`: lexical analysis
- `internal/parser`: parsing
- `internal/runtime`: runtime values, display, environments, and rewriting
- `internal/source`: source files and span tracking
- `examples`: runnable example programs
- `tests`: cross-package integration and end-to-end tests
- `docs`: published user and contributor documentation

## Example

```txt
fn add(a, b) = a + b
fn mul = add ~{ + -> * }

print(add(2, 3))
print(mul(2, 3))
```

## Contributing

Contributions are welcome! Please see the [contributing guide](docs/contributing.md) for details on how to get involved.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
