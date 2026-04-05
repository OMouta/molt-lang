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
	case *ast.Identifier:
		return node.Name
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
		return node.Target.Name + " = " + formatExpr(node.Value, indent)
	case *ast.IndexExpr:
		return formatExpr(node.Target, indent) + "[" + formatExpr(node.Index, indent) + "]"
	case *ast.UnaryExpr:
		if node.Operator == ast.UnaryNot {
			return "not " + formatExpr(node.Operand, indent)
		}
		return "-" + formatExpr(node.Operand, indent)
	case *ast.BinaryExpr:
		return "(" + formatExpr(node.Left, indent) + " " + string(node.Operator) + " " + formatExpr(node.Right, indent) + ")"
	case *ast.ConditionalExpr:
		return "if " + formatExpr(node.Condition, indent) + " -> " + formatExpr(node.ThenBranch, indent) + " else -> " + formatExpr(node.ElseBranch, indent)
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
