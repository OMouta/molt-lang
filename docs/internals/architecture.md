# Architecture

## Overview

The implementation is organized as a straightforward pipeline:

```txt
source file -> lexer -> parser -> AST -> evaluator -> runtime values
```

Mutations add a second path:

```txt
code value -> rewrite engine -> rewritten AST -> eval
```

## Lexer

Location: `internal/lexer`

Responsibilities:

- scan source text with precise byte, line, and column tracking
- produce tokens with stable spans
- decode string escapes
- reject malformed numbers, strings, and unexpected characters with parse diagnostics

The lexer strips trivia and leaves newline sensitivity to span-aware parser logic.

## Parser

Location: `internal/parser`

Responsibilities:

- build the typed AST in `internal/ast`
- enforce precedence and associativity
- handle newline-sensitive expression sequencing
- parse keyword-led control flow such as `if`, `try`, loops, and `match`
- parse quote and mutation literals
- parse postfix chaining for calls, indexing, quoted-argument sugar, and mutation application

The parser emits shared parse diagnostics instead of ad hoc strings.

## Runtime And Evaluator

Locations: `internal/runtime`, `internal/evaluator`

Runtime responsibilities:

- define first-class value types
- manage lexical environment chains
- format values for `show` and `print`
- apply mutation rewrites

Evaluator responsibilities:

- execute AST nodes
- create closures and quoted code values
- resolve and execute imported module files
- evaluate first-match `match` expressions with branch-local capture scope
- execute builtins
- preserve captured environments for functions and quotes

The evaluator is also responsible for wiring builtin output so CLI execution and tests can both observe `print(...)`.

Current import behavior is intentionally minimal:

- imports resolve relative to the importing source file
- each imported file evaluates in an isolated module scope rooted in builtins
- imported modules expose only explicitly exported top-level bindings
- non-exported module bindings remain private inside the module environment
- imported modules are cached per evaluation run after their first successful evaluation
- import cycles are detected from the active module load stack and reported as runtime diagnostics
- module namespacing and richer export forms are left for later tasks

## Mutation And Rewrite Engine

Location: `internal/runtime`

The mutation system works in three stages:

1. Parse mutation rules into AST form.
2. Validate that each rule uses a supported pattern shape.
3. Apply rules in order using pre-order traversal and immutable AST rewriting.

Properties guaranteed by the rewrite engine:

- parent-before-child traversal
- top-to-bottom rule order
- no re-matching of replacement nodes in the same pass
- original AST immutability

Mutations can target quoted code, user-defined functions, and other mutation values.

## CLI

Location: `cmd/molt`

The CLI:

- reads a single source file
- parses and evaluates it
- streams `print(...)` output to stdout
- renders parse and runtime diagnostics to stderr
- returns stable exit codes

This keeps the executable layer thin while leaving language behavior in testable internal packages.
