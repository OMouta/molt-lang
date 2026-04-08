# Language Guide

## Core Ideas

`@molt` treats code as a runtime value. A program can:

- build code with `@{ ... }`
- transform code with `~{ ... }`
- execute code with `eval(...)`

The most important forms are:

```txt
@{ ... }
~{ pattern -> replacement }
eval(code)
import "./module.molt"
```

## Values

The current runtime includes:

- `number`
- `string`
- `boolean`
- `nil`
- `list`
- `record`
- `error`
- `function`
- `native-function`
- `code`
- `mutation`

The builtin `type(x)` returns those exact strings.

## Syntax Overview

Bindings:

```txt
x = expr
[left, right] = pair
record { name: who, stats: record { runs: count } } = profile
profile.name = expr
```

Functions:

```txt
fn add(a, b) = a + b
fn(x) = x + 1
```

Imports:

```txt
import "./math.molt"
export add
```

Blocks:

```txt
{
  expr1
  expr2
  expr3
}
```

Lists:

```txt
[1, 2, 3]
xs[0]
```

Records:

```txt
record { name: "molt", version: 1 }
record {}
profile.name = "bolt"
profile.stats.runs = profile.stats.runs + 1
profile.name
profile.stats.runs
profile["name"]
err.message
err["data"]
```

Record field assignment mutates the existing record in place. Updating an existing field keeps its position, assigning a missing field appends it at the end, and the assignment expression returns the assigned value.

Conditionals:

```txt
if cond -> expr
if cond -> expr else -> expr
try expr catch err -> expr
match value {
  1 -> "one"
  name -> name
  _ -> "other"
}
```

Loops:

```txt
while cond -> expr
while cond -> {
  step1
  step2
}
for item in items -> expr
for [left, right] in pairs -> expr
for ch in "text" -> {
  step1
  step2
}
break
continue
```

Quoted-argument sugar:

```txt
warp @{ 2 + 3 }
```

That is equivalent to `warp(@{ 2 + 3 })`.

## Operators

Arithmetic:

```txt
+ - * / %
```

Comparison:

```txt
== != < <= > >=
```

Boolean:

```txt
and or not
```

The language uses strict booleans. `if`, `and`, `or`, and `not` require real boolean values.

`while` also requires a real boolean condition. A `while` expression returns `nil` when the loop finishes.

Each `while` iteration runs in a fresh child scope rooted in the surrounding environment. Assignments can still update outer bindings, but new iteration-local bindings do not leak after the iteration ends.

List destructuring binds by exact shape: the value must be a list with the same number of elements as the pattern. Record destructuring requires a record value with each named field present; extra record fields are ignored. Mismatches raise runtime diagnostics before any bindings from that pattern are written.

`for` currently iterates over lists and strings. String iteration walks Unicode code points and yields one-character strings. A `for` expression also returns `nil`, and each iteration uses the same fresh child-scope model as `while`: outer bindings can be updated, but the loop binding and any new locals do not leak after the iteration ends. The loop binding can be either a plain identifier or the same list/record destructuring forms used by assignment.

`break` exits the nearest enclosing loop immediately. `continue` skips the rest of the current iteration and resumes with the next one. Both forms work through surrounding block scopes, but execution raises a runtime diagnostic if either form is reached outside a loop body.

`try ... catch ...` is also an expression. If the `try` body finishes normally, the whole expression returns that value and the `catch` branch is skipped. If the `try` body raises a failure, the `catch` branch runs in a fresh child scope with its binding set to an error value.

Explicit `throw(error(...))` preserves the original error value, including optional `data`. Ordinary runtime diagnostics such as invalid builtin calls or import-time runtime failures are catchable too, but they are normalized to `error(message)` values. Loop control such as `break` and `continue` is not catchable.

`match` evaluates its subject once and checks cases from top to bottom. The first matching case wins. Supported patterns for now are:

- literal patterns such as `1`, `"ok"`, `true`, `false`, and `nil`
- identifier patterns such as `name`, which always match and bind the subject value for that branch
- `_` as a wildcard pattern that matches without binding

Each matched branch runs in a fresh child scope rooted in the surrounding environment, so capture bindings stay local to that branch. Existing outer bindings can still be updated from inside the branch. There is no exhaustiveness checking yet; if no case matches, the whole `match` expression returns `nil`.

## Quote And Eval

Quotes capture the current lexical environment by reference but do not execute eagerly:

```txt
x = 10
code = @{ x + 1 }
eval(code)   # 11
```

Each `eval(code)` re-runs the quoted AST from scratch in a fresh frame rooted in the captured environment.

## Imports

Imports load another local `.molt` file relative to the importing file:

```txt
import "./math.molt"
```

Current import behavior:

- the imported file runs in its own module scope with access to builtins
- the imported file does not automatically see the caller's local bindings
- only explicitly exported top-level bindings are introduced into the current scope
- non-exported module-local bindings stay private to the module
- repeated imports of the same resolved module path share one cached module instance for that evaluation run
- direct and indirect import cycles fail with a runtime diagnostic
- the `import ...` expression itself evaluates to `nil`

Example:

```txt
# math.molt
fn add(a, b) = a + b
base = 40
export add
export base

# main.molt
import "./math.molt"
print(add(base, 2))
```

## Mutations

Mutation literals store ordered rewrite rules:

```txt
m = ~{
  + -> *
  1 -> 2
}
```

Mutations can be applied to:

- code values
- user-defined functions
- other mutation values

Mutation returns a new value. The original target is unchanged.

Supported matching forms:

- operator replacement: `+ -> *`
- identifier replacement: `x -> y`
- literal replacement: `1 -> 2`
- exact subtree replacement: `(a + b) -> (a * b)`

## Builtins

See the auto-generated **[Builtins Reference](/reference/builtins)** for a full listing of every built-in function with signatures and descriptions.

## REPL

Running `molt` with no file starts a stateful REPL.

- each successful entry evaluates in the same environment as later entries
- multiline forms such as blocks, quotes, mutation literals, and grouped expressions keep reading until they are complete
- parse and runtime diagnostics are rendered, but the session stays alive
- non-`nil` results are printed automatically
- `:help` prints the available REPL commands
- `:load path/to/file.molt` evaluates a file inside the current REPL environment
- `:history` prints previously submitted entries with line numbers
- `:quit` and `:exit` leave the session

## Display

Display is source-like and stable enough for tests.

Examples:

```txt
show([1, 2])              -> "[1, 2]"
show(record { x: 1 })     -> "record { x: 1 }"
show(@{ 2 + 3 })          -> "@{ (2 + 3) }"
show(~{ x -> y\n1 -> 2 }) -> "~{\n  x -> y\n  1 -> 2\n}"
```

## Errors

The runtime reports precise diagnostics for:

- invalid exports
- import failures
- duplicate record field names
- invalid record field access
- invalid record field assignment
- destructuring mismatches
- invalid while conditions
- invalid for loop iterables
- invalid `break` and `continue` outside loops
- invalid mutation targets
- invalid eval targets
- invalid call targets
- invalid operand types
- invalid list indices
- parse failures

Error values are ordinary runtime data. They can be stored, inspected, and passed around in user code. `throw(error(...))` is what turns one into an aborting runtime failure. CLI diagnostics are separate: they are what the evaluator reports when execution actually aborts.

CLI diagnostics include file path, line, column, source excerpts, and caret markers.
