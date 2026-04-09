package runtime

import (
	"fmt"
	"strconv"
	"strings"

	"molt/internal/ast"
)

const compactDisplayLimit = 60

func ShowValue(value Value) string {
	return formatValue(value, 0)
}

func formatValue(value Value, indent int) string {
	switch v := value.(type) {
	case *NumberValue:
		return strconv.FormatFloat(v.Value, 'f', -1, 64)
	case *StringValue:
		return strconv.Quote(v.Value)
	case *BooleanValue:
		if v.Value {
			return "true"
		}
		return "false"
	case NilValue:
		return "nil"
	case *ListValue:
		return formatListValue(v, indent)
	case *RecordValue:
		return formatRecordValue(v, indent)
	case *ErrorValue:
		return formatErrorValue(v, indent)
	case *UserFunctionValue:
		return formatFunctionValue(v, indent)
	case *NativeFunctionValue:
		return "<native fn>"
	case *CodeValue:
		return formatCodeValue(v, indent)
	case *MutationValue:
		return formatMutationValue(v, indent)
	default:
		return fmt.Sprintf("<unknown %T>", value)
	}
}

func formatListValue(list *ListValue, indent int) string {
	if list == nil {
		return "nil"
	}

	parts := make([]string, 0, len(list.Elements))
	compact := "["
	multiline := false

	for i, element := range list.Elements {
		formatted := formatValue(element, indent+1)
		parts = append(parts, formatted)
		if strings.Contains(formatted, "\n") {
			multiline = true
		}

		if i > 0 {
			compact += ", "
		}
		compact += formatted
	}
	compact += "]"

	if !multiline && len(compact) <= compactDisplayLimit {
		return compact
	}

	if len(parts) == 0 {
		return "[]"
	}

	lines := make([]string, 0, len(parts)+2)
	lines = append(lines, "[")
	for i := range parts {
		part := formatValue(list.Elements[i], 0)
		line := indentString(indent+1) + indentMultiline(part, indent+1)
		if i < len(parts)-1 {
			line += ","
		}
		lines = append(lines, line)
	}
	lines = append(lines, indentString(indent)+"]")
	return strings.Join(lines, "\n")
}

func formatRecordValue(record *RecordValue, indent int) string {
	if record == nil || len(record.Fields) == 0 {
		return "record {}"
	}

	parts := make([]string, 0, len(record.Fields))
	compact := "record { "
	multiline := false

	for i, field := range record.Fields {
		formatted := field.Name + ": " + formatValue(field.Value, indent+1)
		parts = append(parts, formatted)
		if strings.Contains(formatted, "\n") {
			multiline = true
		}

		if i > 0 {
			compact += ", "
		}
		compact += formatted
	}
	compact += " }"

	if !multiline && len(compact) <= compactDisplayLimit {
		return compact
	}

	lines := []string{"record {"}
	for i := range parts {
		part := record.Fields[i].Name + ": " + formatValue(record.Fields[i].Value, 0)
		line := indentString(indent+1) + indentMultiline(part, indent+1)
		if i < len(parts)-1 {
			line += ","
		}
		lines = append(lines, line)
	}
	lines = append(lines, indentString(indent)+"}")
	return strings.Join(lines, "\n")
}

func formatErrorValue(errValue *ErrorValue, indent int) string {
	if errValue == nil {
		return `error { message: "" }`
	}

	parts := []string{`message: ` + strconv.Quote(errValue.Message)}
	if errValue.HasData {
		parts = append(parts, "data: "+formatValue(errValue.Data, indent+1))
	}

	compact := "error { " + strings.Join(parts, ", ") + " }"
	if !strings.Contains(compact, "\n") && len(compact) <= compactDisplayLimit {
		return compact
	}

	lines := []string{"error {"}
	for i, part := range parts {
		line := indentString(indent+1) + indentMultiline(part, indent+1)
		if i < len(parts)-1 {
			line += ","
		}
		lines = append(lines, line)
	}
	lines = append(lines, indentString(indent)+"}")
	return strings.Join(lines, "\n")
}

func formatFunctionValue(fn *UserFunctionValue, indent int) string {
	head := "fn"
	if fn.Name != "" {
		head += " " + fn.Name
	}
	head += "(" + strings.Join(fn.Parameters, ", ") + ") = "

	body := formatExpr(fn.Body, indent)
	if strings.Contains(body, "\n") || len(head)+len(body) > compactDisplayLimit {
		body = indentMultiline(formatExpr(fn.Body, indent+1), indent+1)
		return head + "\n" + indentString(indent+1) + body
	}

	return head + body
}

func formatCodeValue(code *CodeValue, indent int) string {
	return formatDelimitedExpr("@{", "}", code.Body, indent)
}

func formatMutationValue(mutation *MutationValue, indent int) string {
	if mutation == nil || len(mutation.Rules) == 0 {
		return "~{ }"
	}

	compactParts := make([]string, 0, len(mutation.Rules))
	for _, rule := range mutation.Rules {
		part := formatMutationRule(rule, indent)
		if strings.Contains(part, "\n") {
			return formatMutationValueMultiline(mutation, indent)
		}
		compactParts = append(compactParts, part)
	}

	compact := "~{ " + strings.Join(compactParts, " ") + " }"
	if len(compact) <= compactDisplayLimit && len(mutation.Rules) == 1 {
		return compact
	}

	return formatMutationValueMultiline(mutation, indent)
}

func formatMutationValueMultiline(mutation *MutationValue, indent int) string {
	lines := []string{"~{"}
	for _, rule := range mutation.Rules {
		lines = append(lines, indentString(indent+1)+indentMultiline(formatMutationRule(rule, indent+1), indent+1))
	}
	lines = append(lines, indentString(indent)+"}")
	return strings.Join(lines, "\n")
}

func formatMutationRule(rule *ast.MutationRule, indent int) string {
	return formatExpr(rule.Pattern, indent) + " -> " + formatExpr(rule.Replacement, indent)
}

func formatMatchExpr(match *ast.MatchExpr, indent int) string {
	if match == nil {
		return "match nil {}"
	}

	if len(match.Cases) == 0 {
		return "match " + formatExpr(match.Subject, indent) + " {}"
	}

	compactParts := make([]string, 0, len(match.Cases))
	for _, matchCase := range match.Cases {
		part := formatExpr(matchCase.Pattern, indent+1) + " -> " + formatExpr(matchCase.Branch, indent+1)
		if strings.Contains(part, "\n") {
			return formatMatchExprMultiline(match, indent)
		}
		compactParts = append(compactParts, part)
	}

	compact := "match " + formatExpr(match.Subject, indent) + " { " + strings.Join(compactParts, " ") + " }"
	if len(compact) <= compactDisplayLimit && len(match.Cases) == 1 {
		return compact
	}

	return formatMatchExprMultiline(match, indent)
}

func formatMatchExprMultiline(match *ast.MatchExpr, indent int) string {
	lines := []string{"match " + formatExpr(match.Subject, indent) + " {"}
	for _, matchCase := range match.Cases {
		part := formatExpr(matchCase.Pattern, indent+1) + " -> " + formatExpr(matchCase.Branch, indent+1)
		lines = append(lines, indentString(indent+1)+indentMultiline(part, indent+1))
	}
	lines = append(lines, indentString(indent)+"}")
	return strings.Join(lines, "\n")
}

func formatDelimitedExpr(open, close string, body ast.Expr, indent int) string {
	inner := formatQuotedBody(body, indent)
	if !strings.Contains(inner, "\n") && len(open)+1+len(inner)+1+len(close) <= compactDisplayLimit {
		return open + " " + inner + " " + close
	}

	lines := []string{open}
	for _, line := range strings.Split(inner, "\n") {
		lines = append(lines, indentString(indent+1)+line)
	}
	lines = append(lines, indentString(indent)+close)
	return strings.Join(lines, "\n")
}

func formatQuotedBody(expr ast.Expr, indent int) string {
	block, ok := expr.(*ast.BlockExpr)
	if !ok {
		return formatExpr(expr, indent+1)
	}

	if len(block.Expressions) == 0 {
		return ""
	}

	lines := make([]string, 0, len(block.Expressions))
	for _, item := range block.Expressions {
		lines = append(lines, formatExpr(item, indent+1))
	}
	return strings.Join(lines, "\n")
}

func formatExpr(expr ast.Expr, indent int) string {
	switch node := expr.(type) {
	case *ast.OperatorLiteral:
		return node.Symbol
	case *ast.NumberLiteral:
		return strconv.FormatFloat(node.Value, 'f', -1, 64)
	case *ast.StringLiteral:
		return strconv.Quote(node.Value)
	case *ast.BooleanLiteral:
		if node.Value {
			return "true"
		}
		return "false"
	case *ast.NilLiteral:
		return "nil"
	case *ast.BreakExpr:
		return "break"
	case *ast.ContinueExpr:
		return "continue"
	case *ast.Identifier:
		return node.Name
	case *ast.ExportExpr:
		return "export " + formatExpr(node.Name, indent)
	case *ast.ImportExpr:
		return "import " + formatExpr(node.Path, indent)
	case *ast.GroupExpr:
		return "(" + formatExpr(node.Inner, indent) + ")"
	case *ast.ListLiteral:
		parts := make([]string, 0, len(node.Elements))
		compact := "["
		multiline := false
		for i, element := range node.Elements {
			part := formatExpr(element, indent+1)
			parts = append(parts, part)
			if strings.Contains(part, "\n") {
				multiline = true
			}
			if i > 0 {
				compact += ", "
			}
			compact += part
		}
		compact += "]"
		if !multiline && len(compact) <= compactDisplayLimit {
			return compact
		}
		lines := []string{"["}
		for i := range parts {
			part := formatExpr(node.Elements[i], 0)
			line := indentString(indent+1) + indentMultiline(part, indent+1)
			if i < len(parts)-1 {
				line += ","
			}
			lines = append(lines, line)
		}
		lines = append(lines, indentString(indent)+"]")
		return strings.Join(lines, "\n")
	case *ast.ListBindingPattern:
		parts := make([]string, 0, len(node.Elements))
		compact := "["
		multiline := false
		for i, element := range node.Elements {
			part := formatExpr(element, indent+1)
			parts = append(parts, part)
			if strings.Contains(part, "\n") {
				multiline = true
			}
			if i > 0 {
				compact += ", "
			}
			compact += part
		}
		compact += "]"
		if !multiline && len(compact) <= compactDisplayLimit {
			return compact
		}
		lines := []string{"["}
		for i := range parts {
			part := formatExpr(node.Elements[i], 0)
			line := indentString(indent+1) + indentMultiline(part, indent+1)
			if i < len(parts)-1 {
				line += ","
			}
			lines = append(lines, line)
		}
		lines = append(lines, indentString(indent)+"]")
		return strings.Join(lines, "\n")
	case *ast.RecordLiteral:
		if len(node.Fields) == 0 {
			return "record {}"
		}

		parts := make([]string, 0, len(node.Fields))
		compact := "record { "
		multiline := false
		for i, field := range node.Fields {
			part := field.Name.Name + ": " + formatExpr(field.Value, indent+1)
			parts = append(parts, part)
			if strings.Contains(part, "\n") {
				multiline = true
			}
			if i > 0 {
				compact += ", "
			}
			compact += part
		}
		compact += " }"
		if !multiline && len(compact) <= compactDisplayLimit {
			return compact
		}
		lines := []string{"record {"}
		for i := range parts {
			part := node.Fields[i].Name.Name + ": " + formatExpr(node.Fields[i].Value, 0)
			line := indentString(indent+1) + indentMultiline(part, indent+1)
			if i < len(parts)-1 {
				line += ","
			}
			lines = append(lines, line)
		}
		lines = append(lines, indentString(indent)+"}")
		return strings.Join(lines, "\n")
	case *ast.RecordBindingPattern:
		if len(node.Fields) == 0 {
			return "record {}"
		}

		parts := make([]string, 0, len(node.Fields))
		compact := "record { "
		multiline := false
		for i, field := range node.Fields {
			part := field.Name.Name + ": " + formatExpr(field.Value, indent+1)
			parts = append(parts, part)
			if strings.Contains(part, "\n") {
				multiline = true
			}
			if i > 0 {
				compact += ", "
			}
			compact += part
		}
		compact += " }"
		if !multiline && len(compact) <= compactDisplayLimit {
			return compact
		}
		lines := []string{"record {"}
		for i := range parts {
			part := node.Fields[i].Name.Name + ": " + formatExpr(node.Fields[i].Value, 0)
			line := indentString(indent+1) + indentMultiline(part, indent+1)
			if i < len(parts)-1 {
				line += ","
			}
			lines = append(lines, line)
		}
		lines = append(lines, indentString(indent)+"}")
		return strings.Join(lines, "\n")
	case *ast.BlockExpr:
		if len(node.Expressions) == 0 {
			return "{\n" + indentString(indent) + "}"
		}
		lines := []string{"{"}
		for _, item := range node.Expressions {
			lines = append(lines, indentString(indent+1)+indentMultiline(formatExpr(item, indent+1), indent+1))
		}
		lines = append(lines, indentString(indent)+"}")
		return strings.Join(lines, "\n")
	case *ast.AssignmentExpr:
		return formatExpr(node.Target, indent) + " = " + formatExpr(node.Value, indent)
	case *ast.IndexExpr:
		return formatExpr(node.Target, indent) + "[" + formatExpr(node.Index, indent) + "]"
	case *ast.FieldAccessExpr:
		return formatExpr(node.Target, indent) + "." + node.Name.Name
	case *ast.UnaryExpr:
		if node.Operator == ast.UnaryNot {
			return "not " + formatExpr(node.Operand, indent)
		}
		return "-" + formatExpr(node.Operand, indent)
	case *ast.BinaryExpr:
		return "(" + formatExpr(node.Left, indent) + " " + string(node.Operator) + " " + formatExpr(node.Right, indent) + ")"
	case *ast.ConditionalExpr:
		text := "if " + formatExpr(node.Condition, indent) + " -> " + formatExpr(node.ThenBranch, indent)
		if node.ElseBranch != nil {
			text += " else -> " + formatExpr(node.ElseBranch, indent)
		}
		return text
	case *ast.WhileExpr:
		return "while " + formatExpr(node.Condition, indent) + " -> " + formatExpr(node.Body, indent)
	case *ast.TryCatchExpr:
		return "try " + formatExpr(node.Body, indent) + " catch " + node.CatchBinding.Name + " -> " + formatExpr(node.CatchBranch, indent)
	case *ast.MatchExpr:
		return formatMatchExpr(node, indent)
	case *ast.ForInExpr:
		return "for " + formatExpr(node.Binding, indent) + " in " + formatExpr(node.Iterable, indent) + " -> " + formatExpr(node.Body, indent)
	case *ast.CallExpr:
		args := make([]string, 0, len(node.Arguments))
		for _, arg := range node.Arguments {
			args = append(args, formatExpr(arg, indent))
		}
		return formatExpr(node.Callee, indent) + "(" + strings.Join(args, ", ") + ")"
	case *ast.NamedFunctionExpr:
		parameters := make([]string, 0, len(node.Parameters))
		for _, parameter := range node.Parameters {
			parameters = append(parameters, parameter.Name)
		}
		return "fn " + node.Name.Name + "(" + strings.Join(parameters, ", ") + ") = " + formatExpr(node.Body, indent)
	case *ast.FunctionLiteralExpr:
		parameters := make([]string, 0, len(node.Parameters))
		for _, parameter := range node.Parameters {
			parameters = append(parameters, parameter.Name)
		}
		return "fn(" + strings.Join(parameters, ", ") + ") = " + formatExpr(node.Body, indent)
	case *ast.QuoteExpr:
		return formatDelimitedExpr("@{", "}", node.Body, indent)
	case *ast.UnquoteExpr:
		return "~(" + formatExpr(node.Expression, indent) + ")"
	case *ast.SpliceExpr:
		return "~[" + formatExpr(node.Expression, indent) + "]"
	case *ast.MutationLiteralExpr:
		return formatMutationValue(&MutationValue{Rules: node.Rules}, indent)
	case *ast.ApplyMutationExpr:
		return formatExpr(node.Target, indent) + " ~ " + formatExpr(node.Mutation, indent)
	default:
		return fmt.Sprintf("<expr %T>", expr)
	}
}

func indentString(depth int) string {
	return strings.Repeat("  ", depth)
}

func indentMultiline(text string, depth int) string {
	indent := "\n" + indentString(depth)
	return strings.ReplaceAll(text, "\n", indent)
}
