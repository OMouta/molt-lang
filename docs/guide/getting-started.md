# Getting Started

## Installation

molt does not have versioned releases. Prebuilt binaries are published as **nightly builds** on every day that includes at least one commit. Download the latest build for your platform from the [releases page](https://github.com/OMouta/molt-lang/releases):

| Platform | Archive |
| --- | --- |
| Windows x86-64 | `molt-windows-x86_64.zip` |
| macOS Apple Silicon | `molt-macos-aarch64.zip` |
| Linux x86-64 | `molt-linux-x86_64.zip` |

Unzip the archive and place the `molt` binary somewhere on your `PATH`.

::: tip Building from source
If a nightly is not available yet or you want the absolute latest, build from source — see below.
:::

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

Useful REPL commands:

- `:help` shows the built-in commands
- `:load path/to/file.molt` loads and runs a file in the current session
- `:history` shows previously submitted entries
- `:quit` or `:exit` leaves the session

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

- [`basic/basic_mutation`](/reference/examples#basic-mutation) — functions and mutations
- [`basic/compare_worlds`](/reference/examples#compare-worlds) — eval and mutation together
- [`basic/variant_gallery`](/reference/examples#variant-gallery) — match patterns
