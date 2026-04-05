# Contributing

## Development Commands

From the repository root:

```powershell
go test ./...
go vet ./...
go build ./...
gofmt -w .
```

## Repository Layout

- `cmd/molt`: CLI entrypoint and CLI tests
- `internal/source`: source files, positions, and spans
- `internal/diagnostic`: structured parse/runtime diagnostics and rendering
- `internal/lexer`: tokenization
- `internal/parser`: AST construction and precedence handling
- `internal/ast`: AST nodes
- `internal/runtime`: values, environments, rewriting, and display
- `internal/evaluator`: execution and builtins
- `examples`: verified runnable programs
- `editors/vscode-molt`: VS Code syntax-highlighting and file-icon extension
- `tests`: integration and end-to-end coverage

## Working Style

- Keep behavior production-complete. Avoid placeholders and speculative shortcuts.
- Add tests with each semantic change.
- Prefer extending existing integration coverage when adding language features.
- Use `gofmt` before final verification.

## Testing Expectations

The repo relies on multiple test layers:

- package-level unit tests inside `internal/*`
- CLI tests in `cmd/molt`
- spec and regression coverage in `tests`
- CLI smoke checks against example programs

When changing syntax or semantics, update both focused tests and at least one realistic integration path when appropriate.
