# Contributing

## Development Commands

From the repository root:

```powershell
go test ./...
go vet ./...
go build ./...
gofmt -w .
```

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
