// Package formatter implements the canonical Molt source formatter.
//
// Style rules:
//   - 2-space indentation
//   - 80-character soft line limit
//   - Compact-first: inline when fits, expand when it doesn't
//   - Trailing commas on multiline lists and records
//   - Top-level auto-grouping: imports first, blank lines around named functions
package formatter

import (
	"strconv"
	"strings"

	"molt/internal/ast"
)

const lineLimit = 80

// Format returns the canonical formatted source text for a parsed program.
// The output is deterministic and idempotent.
func Format(prog *ast.Program) string {
	var sb strings.Builder

	// Partition top-level expressions: imports go first as a group.
	var imports []ast.Expr
	var rest []ast.Expr
	for _, expr := range prog.Expressions {
		if _, ok := expr.(*ast.ImportExpr); ok {
			imports = append(imports, expr)
		} else {
			rest = append(rest, expr)
		}
	}

	// Imports: printed consecutively with no blank lines between them.
	for _, imp := range imports {
		sb.WriteString(formatExpr(imp, 0))
		sb.WriteByte('\n')
	}

	// Single blank line separating imports from the rest.
	if len(imports) > 0 && len(rest) > 0 {
		sb.WriteByte('\n')
	}

	// Remaining statements: blank lines around named function definitions.
	// Rule: blank line before a named function (if preceded by anything),
	// and blank line after a named function (if followed by anything).
	for i, expr := range rest {
		_, isFunc := expr.(*ast.NamedFunctionExpr)

		if i > 0 {
			_, prevIsFunc := rest[i-1].(*ast.NamedFunctionExpr)
			if isFunc || prevIsFunc {
				sb.WriteByte('\n')
			}
		}

		sb.WriteString(formatExpr(expr, 0))
		sb.WriteByte('\n')
	}

	// Normalize: strip trailing blank lines, ensure single newline at end.
	result := strings.TrimRight(sb.String(), "\n") + "\n"
	return result
}

// fits reports whether text can appear on one line at the given indentation
// depth without exceeding the line limit (depth*2 + len(text) <= lineLimit).
func fits(text string, depth int) bool {
	return depth*2+len(text) <= lineLimit
}

func hasNewline(s string) bool {
	return strings.Contains(s, "\n")
}

func hasNewlineInAny(parts []string) bool {
	for _, p := range parts {
		if hasNewline(p) {
			return true
		}
	}
	return false
}

// ind returns the indentation string for the given depth (2 spaces per level).
func ind(depth int) string {
	return strings.Repeat("  ", depth)
}

// formatExpr formats a single AST expression node. The returned string has no
// leading indentation on its first line; the caller is responsible for that.
// Continuation lines (after embedded newlines) carry the correct indentation
// for their logical depth.
func formatExpr(node ast.Expr, depth int) string {
	switch n := node.(type) {

	// --- Literals ---
	case *ast.NumberLiteral:
		return strconv.FormatFloat(n.Value, 'f', -1, 64)
	case *ast.StringLiteral:
		return strconv.Quote(n.Value)
	case *ast.BooleanLiteral:
		if n.Value {
			return "true"
		}
		return "false"
	case *ast.NilLiteral:
		return "nil"
	case *ast.OperatorLiteral:
		return n.Symbol

	// --- Identifiers and access ---
	case *ast.Identifier:
		return n.Name
	case *ast.FieldAccessExpr:
		return formatExpr(n.Target, depth) + "." + n.Name.Name
	case *ast.IndexExpr:
		return formatExpr(n.Target, depth) + "[" + formatExpr(n.Index, depth) + "]"

	// --- Grouping ---
	case *ast.GroupExpr:
		return "(" + formatExpr(n.Inner, depth) + ")"

	// --- Operators ---
	case *ast.UnaryExpr:
		if n.Operator == ast.UnaryNot {
			return "not " + formatExpr(n.Operand, depth)
		}
		return "-" + formatExpr(n.Operand, depth)
	case *ast.BinaryExpr:
		return formatExpr(n.Left, depth) + " " + string(n.Operator) + " " + formatExpr(n.Right, depth)

	// --- Assignment ---
	case *ast.AssignmentExpr:
		return formatExpr(n.Target, depth) + " = " + formatExpr(n.Value, depth)

	// --- Collections ---
	case *ast.ListLiteral:
		return formatList(n.Elements, depth)
	case *ast.ListBindingPattern:
		return formatListBinding(n.Elements, depth)
	case *ast.RecordLiteral:
		return formatRecord(n.Fields, depth)
	case *ast.RecordBindingPattern:
		return formatRecordBinding(n.Fields, depth)

	// --- Block ---
	case *ast.BlockExpr:
		return formatBlock(n.Expressions, depth)

	// --- Functions ---
	case *ast.NamedFunctionExpr:
		return formatNamedFunction(n, depth)
	case *ast.FunctionLiteralExpr:
		return formatFunctionLiteral(n, depth)
	case *ast.CallExpr:
		return formatCall(n, depth)

	// --- Control flow ---
	case *ast.ConditionalExpr:
		return formatConditional(n, depth)
	case *ast.WhileExpr:
		return formatWhile(n, depth)
	case *ast.ForInExpr:
		return formatForIn(n, depth)
	case *ast.BreakExpr:
		return "break"
	case *ast.ContinueExpr:
		return "continue"
	case *ast.MatchExpr:
		return formatMatch(n, depth)
	case *ast.TryCatchExpr:
		return formatTryCatch(n, depth)

	// --- Modules ---
	case *ast.ImportExpr:
		return formatImport(n)
	case *ast.ExportExpr:
		return "export " + n.Name.Name

	// --- Meta ---
	case *ast.QuoteExpr:
		return formatQuote(n, depth)
	case *ast.UnquoteExpr:
		return "~(" + formatExpr(n.Expression, depth) + ")"
	case *ast.SpliceExpr:
		return "~[" + formatExpr(n.Expression, depth) + "]"
	case *ast.MutationCaptureExpr:
		return "$" + n.Name.Name
	case *ast.MutationWildcardExpr:
		return "_"
	case *ast.MutationRestCaptureExpr:
		return "...$" + n.Name.Name
	case *ast.MutationLiteralExpr:
		return formatMutation(n.Rules, depth)
	case *ast.ApplyMutationExpr:
		target := formatExpr(n.Target, depth)
		mutation := formatExpr(n.Mutation, depth)
		// When the mutation is a literal ~{ ... }, it directly follows the target
		// with only a space (e.g. "add ~{ + -> * }"). For named mutations the
		// infix " ~ " operator is used (e.g. "code ~ simplify").
		if _, ok := n.Mutation.(*ast.MutationLiteralExpr); ok {
			return target + " " + mutation
		}
		return target + " ~ " + mutation

	default:
		return "<unknown>"
	}
}

// --- Collection helpers ---

func formatList(elements []ast.Expr, depth int) string {
	if len(elements) == 0 {
		return "[]"
	}

	parts := make([]string, len(elements))
	for i, el := range elements {
		parts[i] = formatExpr(el, depth+1)
	}

	// Try inline form.
	inline := "[" + strings.Join(parts, ", ") + "]"
	if !hasNewlineInAny(parts) && fits(inline, depth) {
		return inline
	}

	// Multiline with trailing comma.
	var sb strings.Builder
	sb.WriteString("[\n")
	for _, part := range parts {
		sb.WriteString(ind(depth + 1))
		sb.WriteString(part)
		sb.WriteString(",\n")
	}
	sb.WriteString(ind(depth))
	sb.WriteByte(']')
	return sb.String()
}

func formatListBinding(elements []ast.BindingPattern, depth int) string {
	if len(elements) == 0 {
		return "[]"
	}

	parts := make([]string, len(elements))
	for i, el := range elements {
		parts[i] = formatExpr(el, depth+1)
	}

	inline := "[" + strings.Join(parts, ", ") + "]"
	if !hasNewlineInAny(parts) && fits(inline, depth) {
		return inline
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	for _, part := range parts {
		sb.WriteString(ind(depth + 1))
		sb.WriteString(part)
		sb.WriteString(",\n")
	}
	sb.WriteString(ind(depth))
	sb.WriteByte(']')
	return sb.String()
}

func formatRecord(fields []*ast.RecordField, depth int) string {
	if len(fields) == 0 {
		return "record {}"
	}

	fieldStrs := make([]string, len(fields))
	for i, f := range fields {
		fieldStrs[i] = f.Name.Name + ": " + formatExpr(f.Value, depth+1)
	}

	// Try inline form.
	inline := "record { " + strings.Join(fieldStrs, ", ") + " }"
	if !hasNewlineInAny(fieldStrs) && fits(inline, depth) {
		return inline
	}

	// Multiline with trailing comma.
	var sb strings.Builder
	sb.WriteString("record {\n")
	for _, fs := range fieldStrs {
		sb.WriteString(ind(depth + 1))
		sb.WriteString(fs)
		sb.WriteString(",\n")
	}
	sb.WriteString(ind(depth))
	sb.WriteByte('}')
	return sb.String()
}

func formatRecordBinding(fields []*ast.RecordBindingField, depth int) string {
	if len(fields) == 0 {
		return "record {}"
	}

	fieldStrs := make([]string, len(fields))
	for i, f := range fields {
		fieldStrs[i] = f.Name.Name + ": " + formatExpr(f.Value, depth+1)
	}

	inline := "record { " + strings.Join(fieldStrs, ", ") + " }"
	if !hasNewlineInAny(fieldStrs) && fits(inline, depth) {
		return inline
	}

	var sb strings.Builder
	sb.WriteString("record {\n")
	for _, fs := range fieldStrs {
		sb.WriteString(ind(depth + 1))
		sb.WriteString(fs)
		sb.WriteString(",\n")
	}
	sb.WriteString(ind(depth))
	sb.WriteByte('}')
	return sb.String()
}

// --- Block ---

// formatBlock formats a block expression. Single-expression blocks are kept
// inline as "{ expr }" if they fit; multi-expression blocks always expand.
func formatBlock(exprs []ast.Expr, depth int) string {
	if len(exprs) == 0 {
		return "{}"
	}

	if len(exprs) == 1 {
		inner := formatExpr(exprs[0], depth)
		inline := "{ " + inner + " }"
		if !hasNewline(inner) && fits(inline, depth) {
			return inline
		}
	}

	var sb strings.Builder
	sb.WriteString("{\n")
	for _, expr := range exprs {
		formatted := formatExpr(expr, depth+1)
		sb.WriteString(ind(depth + 1))
		sb.WriteString(formatted)
		sb.WriteByte('\n')
	}
	sb.WriteString(ind(depth))
	sb.WriteByte('}')
	return sb.String()
}

// formatBodyAsBlock wraps a non-block expression in "{ expr }" for use as a
// while/for body. If the expression is already a BlockExpr, formats it
// directly. This ensures while/for bodies are always braced canonically.
func formatBodyAsBlock(expr ast.Expr, depth int) string {
	if block, ok := expr.(*ast.BlockExpr); ok {
		return formatBlock(block.Expressions, depth)
	}
	formatted := formatExpr(expr, depth)
	inline := "{ " + formatted + " }"
	if !hasNewline(formatted) && fits(inline, depth) {
		return inline
	}
	return "{\n" + ind(depth+1) + formatted + "\n" + ind(depth) + "}"
}

// --- Functions ---

func formatParams(params []*ast.Identifier) string {
	names := make([]string, len(params))
	for i, p := range params {
		names[i] = p.Name
	}
	return strings.Join(names, ", ")
}

func formatNamedFunction(n *ast.NamedFunctionExpr, depth int) string {
	head := "fn " + n.Name.Name + "(" + formatParams(n.Parameters) + ") = "
	body := formatExpr(n.Body, depth)
	inline := head + body
	if !hasNewline(body) && fits(inline, depth) {
		return inline
	}
	return head + body
}

func formatFunctionLiteral(n *ast.FunctionLiteralExpr, depth int) string {
	head := "fn(" + formatParams(n.Parameters) + ") = "
	body := formatExpr(n.Body, depth)
	inline := head + body
	if !hasNewline(body) && fits(inline, depth) {
		return inline
	}
	return head + body
}

func formatCall(n *ast.CallExpr, depth int) string {
	callee := formatExpr(n.Callee, depth)
	args := make([]string, len(n.Arguments))
	for i, arg := range n.Arguments {
		args[i] = formatExpr(arg, depth)
	}
	return callee + "(" + strings.Join(args, ", ") + ")"
}

// --- Control flow ---

func formatConditional(n *ast.ConditionalExpr, depth int) string {
	cond := formatExpr(n.Condition, depth)
	then := formatExpr(n.ThenBranch, depth)

	if n.ElseBranch == nil {
		return "if " + cond + " -> " + then
	}

	else_ := formatExpr(n.ElseBranch, depth)
	inline := "if " + cond + " -> " + then + " else -> " + else_
	if !hasNewline(then) && !hasNewline(else_) && fits(inline, depth) {
		return inline
	}
	// Attach "else ->" to the end of the then branch (after its closing "}" if
	// it's a block, or on a new line if it's a plain expression).
	if hasNewline(then) {
		return "if " + cond + " -> " + then + " else -> " + else_
	}
	return "if " + cond + " -> " + then + "\n" + ind(depth) + "else -> " + else_
}

func formatWhile(n *ast.WhileExpr, depth int) string {
	cond := formatExpr(n.Condition, depth)
	body := formatBodyAsBlock(n.Body, depth)
	return "while " + cond + " -> " + body
}

func formatForIn(n *ast.ForInExpr, depth int) string {
	binding := formatExpr(n.Binding, depth)
	iterable := formatExpr(n.Iterable, depth)
	body := formatBodyAsBlock(n.Body, depth)
	return "for " + binding + " in " + iterable + " -> " + body
}

func formatMatch(n *ast.MatchExpr, depth int) string {
	subject := formatExpr(n.Subject, depth)
	if len(n.Cases) == 0 {
		return "match " + subject + " {}"
	}

	// Match is always multiline.
	var sb strings.Builder
	sb.WriteString("match ")
	sb.WriteString(subject)
	sb.WriteString(" {\n")
	for _, c := range n.Cases {
		pattern := formatExpr(c.Pattern, depth+1)
		branch := formatExpr(c.Branch, depth+1)
		sb.WriteString(ind(depth + 1))
		sb.WriteString(pattern)
		sb.WriteString(" -> ")
		sb.WriteString(branch)
		sb.WriteByte('\n')
	}
	sb.WriteString(ind(depth))
	sb.WriteByte('}')
	return sb.String()
}

func formatTryCatch(n *ast.TryCatchExpr, depth int) string {
	body := formatExpr(n.Body, depth)
	catch := formatExpr(n.CatchBranch, depth)
	binding := n.CatchBinding.Name

	inline := "try " + body + " catch " + binding + " -> " + catch
	if !hasNewline(body) && !hasNewline(catch) && fits(inline, depth) {
		return inline
	}
	// Attach "catch" to the end of body (handles block-body case naturally).
	return "try " + body + " catch " + binding + " -> " + catch
}

// --- Modules ---

func formatImport(n *ast.ImportExpr) string {
	switch n.Kind {
	case ast.ImportModule:
		result := "import " + strconv.Quote(n.Path.Value)
		if n.Name != nil {
			result += " as " + n.Name.Name
		}
		return result
	case ast.ImportNamed:
		var namesPart string
		if len(n.Names) == 1 {
			// Canonical: single name without braces.
			namesPart = n.Names[0].Name
		} else {
			names := make([]string, len(n.Names))
			for i, name := range n.Names {
				names[i] = name.Name
			}
			namesPart = "{" + strings.Join(names, ", ") + "}"
		}
		return "import " + namesPart + " from " + strconv.Quote(n.Path.Value)
	}
	return "import " + strconv.Quote(n.Path.Value)
}

// --- Meta ---

func formatQuote(n *ast.QuoteExpr, depth int) string {
	inner := formatQuoteBody(n.Body, depth)
	inline := "@{ " + inner + " }"
	if !hasNewline(inner) && fits(inline, depth) {
		return inline
	}

	var sb strings.Builder
	sb.WriteString("@{\n")
	for _, line := range strings.Split(inner, "\n") {
		sb.WriteString(ind(depth + 1))
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	sb.WriteString(ind(depth))
	sb.WriteByte('}')
	return sb.String()
}

// formatQuoteBody formats the body of a @{} quote. For block bodies, each
// expression is formatted on its own line without the outer braces.
func formatQuoteBody(expr ast.Expr, depth int) string {
	block, ok := expr.(*ast.BlockExpr)
	if !ok {
		return formatExpr(expr, depth+1)
	}
	if len(block.Expressions) == 0 {
		return ""
	}
	lines := make([]string, len(block.Expressions))
	for i, item := range block.Expressions {
		lines[i] = formatExpr(item, depth+1)
	}
	return strings.Join(lines, "\n")
}

func formatMutation(rules []*ast.MutationRule, depth int) string {
	if len(rules) == 0 {
		return "~{}"
	}

	parts := make([]string, len(rules))
	for i, rule := range rules {
		parts[i] = formatExpr(rule.Pattern, depth+1) + " -> " + formatExpr(rule.Replacement, depth+1)
	}

	// Single rule: try inline form.
	if len(parts) == 1 {
		inline := "~{ " + parts[0] + " }"
		if !hasNewline(parts[0]) && fits(inline, depth) {
			return inline
		}
	}

	// Multiline (always for 2+ rules).
	var sb strings.Builder
	sb.WriteString("~{\n")
	for _, part := range parts {
		sb.WriteString(ind(depth + 1))
		sb.WriteString(part)
		sb.WriteByte('\n')
	}
	sb.WriteString(ind(depth))
	sb.WriteByte('}')
	return sb.String()
}
