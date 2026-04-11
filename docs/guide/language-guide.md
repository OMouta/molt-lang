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

After `import "std:meta"`, `type(x)` returns those exact strings.

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
import "std:io"
import "./math.molt"
import "./math.molt" as m
import base from "./math.molt"
import {base, add_secret} from "./math.molt"
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

Explicit `throw(error(...))` preserves the original error value, including optional `data`. Ordinary runtime diagnostics such as invalid standard-library calls or import-time runtime failures are catchable too, but they are normalized to `error(message)` values. Loop control such as `break` and `continue` is not catchable.

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

Quotes also support single-node interpolation with `~(expr)`:

```txt
part = @{ 1 + 2 }
code = @{ ~(part) * 3 }
eval(code)   # 9
```

`~(expr)` evaluates `expr` when the surrounding quote is created, expects a `code` value, and inserts that code value's body as one AST node. It is only valid inside quotes, and it only works in ordinary expression positions. Identifier-only slots such as function names, parameter names, and record field names are not interpolated by this form.

Unquote inserts syntax, not the captured environment of the inserted fragment. The final quoted result still captures the environment of the outer quote that is being built.

Malformed interpolation shapes are rejected when the quote is created, before any `~(...)` or `~[...]` expression is evaluated. That keeps errors local to the quoted template and avoids partial side effects from interpolation helpers.

Quotes also support splicing with `~[expr]` when a quote position expects multiple child expressions:

```txt
items = @{ [1, 2] }
steps = @{ total = 1
total = total + 2 }

listCode = @{ [0, ~[items], 3] }
callCode = @{ range(~[items]) }
blockCode = @{ ~[steps]
total }
```

`~[expr]` is only valid inside quotes, and only in:

- list literal element positions
- call argument positions
- block or top-level quote sequence positions

For list and call positions, `expr` must produce a quoted list such as `@{ [1, 2] }`. For block positions, `expr` must produce a quoted block such as:

```txt
@{
  a = 1
  b = 2
}
```

Use `~(expr)` when you want to insert exactly one AST node. Use `~[expr]` when you want to insert zero or more sibling nodes from a quoted list or quoted block.

When a quote contains interpolation, `show(code)` keeps the original `~(...)` and `~[...]` template instead of only showing the expanded result. That makes generated code easier to debug without changing what `eval(code)` executes.

Quotes are not automatically hygienic. Interpolated syntax is inserted into the final quoted program and resolves names in that generated lexical scope when `eval(code)` runs.

```txt
x = 2
outer = 10
fragment = @{ x + outer }
maker = @{ fn(x) = ~(fragment) }
f = eval(maker)
f(5)   # 15
```

In that example, the inserted `x` from `fragment` resolves to the generated function parameter, while `outer` still comes from the outer quote's captured environment.

As a safety rail, interpolation stays in ordinary expression positions. You cannot directly interpolate new assignment targets, destructuring binders, function parameter names, or other binding-introducing slots.

## Imports

There are two import forms.

### Module import

```txt
import "./math.molt"
import "std:io"
import "./math.molt" as m
import "std:io" as io
```

Loads the module and binds all its exports as a namespace record. The binding name is either the explicit `as` alias or is derived automatically from the path stem (`math` from `"./math.molt"`, `io` from `"std:io"`).

### Named import

```txt
import base from "./math.molt"
import {base, add_secret} from "./math.molt"
import print from "std:io"
```

Loads the module and binds the named exports directly into scope. Use braces for multiple names. Raises a runtime error if the module does not export any of the requested names.

### General rules

- there is no implicit prelude and no ambient builtins
- `std:` imports load built-in standard modules directly
- local imported files run in their own module scope with no automatic access to standard modules
- the imported file does not automatically see the caller's local bindings
- only explicitly exported top-level bindings are visible after import — non-exported bindings stay private to the module
- repeated imports of the same resolved module path share one cached module instance for that evaluation run
- direct and indirect import cycles fail with a runtime diagnostic
- the `import ...` expression itself evaluates to `nil`

### Example

```txt
# math.molt
base = 40
secret = 41
fn add_secret(x) = x + secret
export base
export add_secret

# main.molt — module import (namespace access)
import "std:io"
import "./math.molt"
io.print(math.base)
io.print(math.add_secret(1))

# main.molt — named import (direct access)
import "std:io"
import {base, add_secret} from "./math.molt"
io.print(base)
io.print(add_secret(1))

# main.molt — module import with alias
import "std:io"
import "./math.molt" as m
io.print(m.base)
```

## Mutations

Mutation literals store ordered rewrite rules:

```txt
m = ~{
  + -> *
  1 -> 2
  ($x + 0) -> $x
  [1, ...$tail, 4] -> [0, ...$tail]
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
- named captures in reusable patterns: `($x + 0) -> $x`
- wildcard matches: `(_ + 0) -> 0`
- rest captures in list, block, and call-argument sequences: `[1, ...$tail, 4] -> [0, ...$tail]`

`$name` binds one matched subtree and reuses it in the replacement template. Repeating the same capture name in one pattern requires the matched subtrees to be structurally equal.

`_` matches any single subtree without binding it. `...$name` matches zero or more sibling expressions in list literal elements, block expression sequences, and call argument lists. Only one rest capture is allowed in a single sequence pattern, and a rest capture must stay a rest capture when reused in the replacement.

Mutation substitution follows the same non-hygienic rule as quote interpolation: captured syntax is reinserted as syntax, not as a closure over its old identifier meanings.

```txt
x = 2
outer = 10
wrap = ~{ $body -> fn(x) = $body }
f = eval(@{ x + outer } ~ wrap)
f(7)   # 17
```

Here the generated parameter `x` shadows the free `x` inside `$body`, while `outer` still resolves through the code value's captured environment.

## Standard Library

```txt
import "std:io"
import "std:meta"
import "std:collections"
```

See the auto-generated **[Standard Library Reference](/reference/standard-library)** for the full module listing and exported functions.

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

Display is source-like and stable enough for tests. Import `std:meta` when you want to call `show(...)`.

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
