package runtime

import (
	"fmt"

	"molt/internal/ast"
)

func ApplyMutationValue(target Value, mutation *MutationValue) (Value, error) {
	if mutation == nil {
		return nil, fmt.Errorf("mutation value cannot be nil")
	}

	switch value := target.(type) {
	case *CodeValue:
		rewritten, err := Rewrite(value.Body, mutation)
		if err != nil {
			return nil, err
		}

		return &CodeValue{
			Body: rewritten,
			Env:  value.Env,
		}, nil
	case *UserFunctionValue:
		rewritten, err := Rewrite(value.Body, mutation)
		if err != nil {
			return nil, err
		}

		params := append([]string(nil), value.Parameters...)
		return &UserFunctionValue{
			Name:       value.Name,
			Parameters: params,
			Body:       rewritten,
			Env:        value.Env,
		}, nil
	case *MutationValue:
		rules := append(CloneRules(value.Rules), CloneRules(mutation.Rules)...)
		if err := ValidateMutationRules(rules); err != nil {
			return nil, err
		}

		return &MutationValue{Rules: rules}, nil
	default:
		return nil, fmt.Errorf("cannot apply mutation to value of type %q", target.TypeName())
	}
}

func RuleOperators(rule *ast.MutationRule) (string, string, bool) {
	pattern, ok := rule.Pattern.(*ast.OperatorLiteral)
	if !ok {
		return "", "", false
	}

	replacement, ok := rule.Replacement.(*ast.OperatorLiteral)
	if !ok {
		return "", "", false
	}

	return pattern.Symbol, replacement.Symbol, true
}
