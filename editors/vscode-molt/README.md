# Molt Language Support for VS Code

This extension adds:

- syntax highlighting for `.molt` files
- line-comment support for `#`
- bracket matching and auto-closing for `()`, `[]`, `{}`, `@{}`, and `~{}`
- a custom `.molt` file icon
- a publishable `vsce` packaging setup

## Included Grammar Coverage

The TextMate grammar highlights:

- comments
- strings and escape sequences
- numbers
- `import` and `export`
- `fn`, `if`, `else`, `and`, `or`, `not`
- `true`, `false`, `nil`
- common standard-library names like `eval`, `type`, `len`, `push`, `show`, `print`
- quote and mutation introducers: `@{` and `~{`
- operators, assignment, and arrows
- function names in `fn name(...)`
- function parameters with their own scope
- mutation rule blocks, patterns, and replacements with separate scopes

## Packaging

Install dependencies:

```powershell
npm install
```

Validate the extension manifest:

```powershell
npm run check
```

Package it with `vsce`:

```powershell
npm run package
```
