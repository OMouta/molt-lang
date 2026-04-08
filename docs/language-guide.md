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
```

Loops:

```txt
while cond -> expr
while cond -> {
  step1
  step2
}
for item in items -> expr
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

`for` currently iterates over lists and strings. String iteration walks Unicode code points and yields one-character strings. A `for` expression also returns `nil`, and each iteration uses the same fresh child-scope model as `while`: outer bindings can be updated, but the loop binding and any new locals do not leak after the iteration ends.

`break` exits the nearest enclosing loop immediately. `continue` skips the rest of the current iteration and resumes with the next one. Both forms work through surrounding block scopes, but execution raises a runtime diagnostic if either form is reached outside a loop body.

`try ... catch ...` is also an expression. If the `try` body finishes normally, the whole expression returns that value and the `catch` branch is skipped. If the `try` body raises a failure, the `catch` branch runs in a fresh child scope with its binding set to an error value.

Explicit `throw(error(...))` preserves the original error value, including optional `data`. Ordinary runtime diagnostics such as invalid builtin calls or import-time runtime failures are catchable too, but they are normalized to `error(message)` values. Loop control such as `break` and `continue` is not catchable.

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

`eval(code)`
: Execute a code value.

`type(x)`
: Return the canonical runtime type name.

`args()`
: Return a fresh list of command-line arguments passed after the script path. In REPL mode it returns `[]`.

`len(x)`
: Return the length of a list, the number of Unicode code points in a string, or the number of fields in a record or error value.

`push(list, value)`
: Append to a list in place and return the same list.

Record field assignment
: `record.field = value` mutates the record in place and returns `value`.

`split(text, separator)`
: Split a string into a list of strings.

`join(parts, separator)`
: Join a list of strings into a single string.

`trim(text)`
: Trim leading and trailing whitespace from a string.

`lines(text)`
: Split a string into a list of lines. `\n`, `\r\n`, and `\r` are all treated as line breaks, and one trailing final newline does not add an extra empty line.

`replace(text, old, new)`
: Replace every occurrence of `old` inside `text` with `new`.

`contains(text, needle)`
: Return `true` if `needle` appears inside `text`, or if a record contains a field with that string name.

`keys(record)`
: Return a list of record field names in the record's display order.

`values(record)`
: Return a list of record field values in the same order as `keys(record)`.

`error(message)` / `error(message, data)`
: Build a first-class error value. Error values have type `"error"`, display as `error { ... }`, and expose `message` plus optional `data` fields through normal field access and record-style helpers such as `contains`, `keys`, `values`, and `len`.

`throw(err)`
: Raise an error value as an actual runtime failure. `throw(...)` requires a value of type `"error"`; uncaught throws become runtime diagnostics at the `throw(...)` call site, and thrown values with `data` add a diagnostic note showing that payload.

`range(end)` / `range(start, end)`
: Build an ascending list of integers with an exclusive end bound.

`map(list, fn)`
: Return a new list by applying a callback to each element. The callback may accept `(value)` or `(value, index)`.

`filter(list, fn)`
: Return a new list containing only the elements whose callback result is `true`. The callback may accept `(value)` or `(value, index)`.

`show(x)`
: Return a stable display string.

`read_file(path)`
: Read a file from disk and return its contents as a string. The path must be a non-empty string.

`write_file(path, text)`
: Write a string to a file on disk, replacing any existing contents. The path and text must both be strings, and the path must be non-empty.

`input()`
: Read one line from standard input and return it without the trailing newline. At end of input it returns `""`.

`to_string(x)`
: Convert a value to a string. Strings stay raw; other values use their readable source-like form.

`to_number(x)`
: Convert a number or numeric string to a number.

`print(x)`
: Write a user-facing text form followed by a newline. Strings print without quotes; other values use their readable display form.

`stdin()`
: Read the remaining standard input as a string. If stdin has already been consumed, it returns `""`.

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
