# Getting Started

## Requirements

- Go `1.26`

## Build

From the repository root:

```powershell
go build ./...
```

## Test

```powershell
go test ./...
go vet ./...
```

To rewrite files into canonical Go formatting:

```powershell
gofmt -w .
```

## Run A Program

The CLI executes one source file at a time:

```powershell
go run ./cmd/molt ./examples/basic_mutation.molt
```

Additional command-line arguments are exposed inside the program through `args()`:

```powershell
go run ./cmd/molt ./examples/basic_mutation.molt alpha beta
```

It can also read a program from standard input:

```powershell
'print(1 + 2)' | go run ./cmd/molt -
```

If you run `molt` without a file, it starts a REPL:

```powershell
go run ./cmd/molt
```

Usage:

```txt
molt [file|-] [args...]
```

Exit codes:

- `0`: success
- `1`: CLI usage error
- `2`: source file read error
- `3`: lex or parse diagnostic
- `4`: runtime diagnostic
- `10`: internal failure

## Development Helpers

PowerShell:

```powershell
./dev.ps1 build
./dev.ps1 test
./dev.ps1 lint
./dev.ps1 format
```

Unix-like shells:

```sh
./dev.sh build
./dev.sh test
./dev.sh lint
./dev.sh format
```

## First Programs To Try

- [`examples/basic_mutation.molt`](../examples/basic_mutation.molt)
- [`examples/compare_worlds.molt`](../examples/compare_worlds.molt)
- [`examples/variant_gallery.molt`](../examples/variant_gallery.molt)
