package runtime

import "fmt"

type Binding struct {
	Name  string
	Value Value
}

type Environment struct {
	parent   *Environment
	bindings map[string]Value
}

func NewEnvironment(parent *Environment) *Environment {
	return &Environment{
		parent:   parent,
		bindings: make(map[string]Value),
	}
}

func (e *Environment) Parent() *Environment {
	return e.parent
}

func (e *Environment) Define(name string, value Value) {
	e.bindings[name] = value
}

func (e *Environment) Get(name string) (Value, bool) {
	for env := e; env != nil; env = env.parent {
		if value, ok := env.bindings[name]; ok {
			return value, true
		}
	}

	return nil, false
}

func (e *Environment) MustGet(name string) Value {
	value, ok := e.Get(name)
	if !ok {
		panic(fmt.Sprintf("binding %q does not exist", name))
	}

	return value
}

// Assign updates the nearest existing binding in the chain. If no binding exists,
// it creates one in the current environment.
func (e *Environment) Assign(name string, value Value) {
	for env := e; env != nil; env = env.parent {
		if _, ok := env.bindings[name]; ok {
			env.bindings[name] = value
			return
		}
	}

	e.bindings[name] = value
}

func (e *Environment) HasLocal(name string) bool {
	_, ok := e.bindings[name]
	return ok
}

func (e *Environment) LocalBindings() []Binding {
	names := make([]string, 0, len(e.bindings))
	for name := range e.bindings {
		names = append(names, name)
	}

	// Keep this deterministic for tests and future debug output.
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}

	bindings := make([]Binding, 0, len(names))
	for _, name := range names {
		bindings = append(bindings, Binding{
			Name:  name,
			Value: e.bindings[name],
		})
	}

	return bindings
}
