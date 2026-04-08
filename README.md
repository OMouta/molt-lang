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

## Example

```txt
fn add(a, b) = a + b
fn mul = add ~{ + -> * }

print(add(2, 3))
print(mul(2, 3))
```

## Contributing

Contributions are welcome! Please see the [contributing guide](CONTRIBUTING.md) for details on how to get involved.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
