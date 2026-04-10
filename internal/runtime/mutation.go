package runtime

import (
	"fmt"

	"molt/internal/ast"
)

func ValidateMutationRules(rules []*ast.MutationRule) error {
	for i, rule := range rules {
		if err := ValidateMutationRule(rule); err != nil {
			return fmt.Errorf("invalid mutation rule %d: %w", i+1, err)
		}
	}

	return nil
}

func ValidateMutationRule(rule *ast.MutationRule) error {
	if rule == nil {
		return fmt.Errorf("mutation rule cannot be nil")
	}

	patternOperator := isOperatorLiteral(rule.Pattern)
	replacementOperator := isOperatorLiteral(rule.Replacement)

	switch {
	case patternOperator && !replacementOperator:
		return fmt.Errorf("operator replacement rules must replace one operator with another")
	case !patternOperator && replacementOperator:
		return fmt.Errorf("operator literals are only valid in operator replacement rules")
	}

	if err := validateMutationExpr(rule.Pattern); err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	if err := validateMutationExpr(rule.Replacement); err != nil {
		return fmt.Errorf("invalid replacement: %w", err)
	}

	patternCaptures := make(map[string]struct{})
	if err := collectMutationCaptures(rule.Pattern, patternCaptures); err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	replacementCaptures := make(map[string]struct{})
	if err := collectMutationCaptures(rule.Replacement, replacementCaptures); err != nil {
		return fmt.Errorf("invalid replacement: %w", err)
	}

	for name := range replacementCaptures {
		if _, ok := patternCaptures[name]; !ok {
			return fmt.Errorf("invalid replacement: capture %q is not bound in the pattern", name)
		}
	}

	return nil
}

func Rewrite(expr ast.Expr, mutation *MutationValue) (ast.Expr, error) {
	if mutation == nil {
		return nil, fmt.Errorf("mutation value cannot be nil")
	}

	if err := ValidateMutationRules(mutation.Rules); err != nil {
		return nil, err
	}

	rewritten := CloneExpr(expr)
	for _, rule := range mutation.Rules {
		var err error
		rewritten, err = ApplyRule(rewritten, rule)
		if err != nil {
			return nil, err
		}
	}

	return rewritten, nil
}

func ApplyRule(expr ast.Expr, rule *ast.MutationRule) (ast.Expr, error) {
	if err := ValidateMutationRule(rule); err != nil {
		return nil, err
	}

	rewritten, _ := rewriteWithRule(expr, rule)
	return rewritten, nil
}
