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
```

## Values

The current runtime includes:

- `number`
- `string`
- `boolean`
- `nil`
- `list`
- `function`
- `native-function`
- `code`
- `mutation`

The builtin `type(x)` returns those exact strings.

## Syntax Overview

Bindings:

```txt
x = expr
```

Functions:

```txt
fn add(a, b) = a + b
fn(x) = x + 1
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

Conditionals:

```txt
if cond -> expr else -> expr
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

## Quote And Eval

Quotes capture the current lexical environment by reference but do not execute eagerly:

```txt
x = 10
code = @{ x + 1 }
eval(code)   # 11
```

Each `eval(code)` re-runs the quoted AST from scratch in a fresh frame rooted in the captured environment.

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

`len(x)`
: Return the length of a list or the number of Unicode code points in a string.

`push(list, value)`
: Append to a list in place and return the same list.

`show(x)`
: Return a stable display string.

`print(x)`
: Write the `show(x)` form followed by a newline.

## Display

Display is source-like and stable enough for tests.

Examples:

```txt
show([1, 2])              -> "[1, 2]"
show(@{ 2 + 3 })          -> "@{ (2 + 3) }"
show(~{ x -> y\n1 -> 2 }) -> "~{\n  x -> y\n  1 -> 2\n}"
```

## Errors

The runtime reports precise diagnostics for:

- invalid mutation targets
- invalid eval targets
- invalid call targets
- invalid operand types
- invalid list indices
- parse failures

CLI diagnostics include file path, line, column, source excerpts, and caret markers.
