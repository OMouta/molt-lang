package evaluator

import (
	"fmt"

	"molt/internal/ast"
	"molt/internal/runtime"
)

func (e *Evaluator) assignBindingPattern(env *runtime.Environment, pattern ast.BindingPattern, value runtime.Value) error {
	bindings, err := e.collectBindingAssignments(pattern, value, "")
	if err != nil {
		return err
	}

	for _, binding := range bindings {
		env.Assign(binding.Name, binding.Value)
	}

	return nil
}

func (e *Evaluator) defineBindingPattern(env *runtime.Environment, pattern ast.BindingPattern, value runtime.Value) error {
	bindings, err := e.collectBindingAssignments(pattern, value, "")
	if err != nil {
		return err
	}

	for _, binding := range bindings {
		env.Define(binding.Name, binding.Value)
	}

	return nil
}

func (e *Evaluator) collectBindingAssignments(pattern ast.BindingPattern, value runtime.Value, path string) ([]runtime.Binding, error) {
	switch node := pattern.(type) {
	case *ast.Identifier:
		return []runtime.Binding{{Name: node.Name, Value: value}}, nil
	case *ast.ListBindingPattern:
		list, ok := value.(*runtime.ListValue)
		if !ok {
			return nil, e.runtimeError(node, fmt.Sprintf("list destructuring expects list, got %q%s", value.TypeName(), bindingPathSuffix(path)))
		}

		if len(list.Elements) != len(node.Elements) {
			return nil, e.runtimeError(node, fmt.Sprintf("list destructuring expected %d elements, got %d%s", len(node.Elements), len(list.Elements), bindingPathSuffix(path)))
		}

		bindings := make([]runtime.Binding, 0, len(node.Elements))
		for index, elementPattern := range node.Elements {
			elementBindings, err := e.collectBindingAssignments(elementPattern, list.Elements[index], extendBindingPath(path, fmt.Sprintf("[%d]", index)))
			if err != nil {
				return nil, err
			}

			bindings = append(bindings, elementBindings...)
		}

		return bindings, nil
	case *ast.RecordBindingPattern:
		record, ok := value.(*runtime.RecordValue)
		if !ok {
			return nil, e.runtimeError(node, fmt.Sprintf("record destructuring expects record, got %q%s", value.TypeName(), bindingPathSuffix(path)))
		}

		bindings := make([]runtime.Binding, 0, len(node.Fields))
		for _, field := range node.Fields {
			fieldValue, exists := record.GetField(field.Name.Name)
			if !exists {
				return nil, e.runtimeError(field.Name, fmt.Sprintf("record destructuring missing field %q%s", field.Name.Name, bindingPathSuffix(path)))
			}

			fieldBindings, err := e.collectBindingAssignments(field.Value, fieldValue, extendBindingPath(path, "."+field.Name.Name))
			if err != nil {
				return nil, err
			}

			bindings = append(bindings, fieldBindings...)
		}

		return bindings, nil
	default:
		return nil, fmt.Errorf("unsupported binding pattern type %T", pattern)
	}
}

func bindingPathSuffix(path string) string {
	if path == "" {
		return ""
	}

	return " at " + path
}

func extendBindingPath(path, segment string) string {
	if path == "" {
		return segment
	}

	return path + segment
}
