package evaluator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"molt/internal/ast"
	"molt/internal/builtins"
	"molt/internal/diagnostic"
	"molt/internal/parser"
	"molt/internal/runtime"
	"molt/internal/source"
)

func (e *Evaluator) evalExport(env *runtime.Environment, expr *ast.ExportExpr) (runtime.Value, error) {
	if expr.Name == nil {
		return nil, fmt.Errorf("export expression missing name")
	}

	module := e.currentModule()
	if module == nil {
		return nil, e.runtimeError(expr, "export is only allowed inside imported modules")
	}

	if env != module.env {
		return nil, e.runtimeError(expr, "export is only allowed at module top level")
	}

	if _, exists := module.exported[expr.Name.Name]; exists {
		return nil, e.runtimeError(expr.Name, fmt.Sprintf("duplicate export %q", expr.Name.Name))
	}

	module.exported[expr.Name.Name] = expr.Name.Span()
	module.exportList = append(module.exportList, expr.Name.Name)
	return runtime.Nil, nil
}

func (e *Evaluator) evalImport(env *runtime.Environment, expr *ast.ImportExpr) (runtime.Value, error) {
	if expr.Name == nil {
		return nil, fmt.Errorf("import expression missing name")
	}

	if expr.Path == nil {
		return nil, fmt.Errorf("import expression missing path")
	}

	if expr.Path.Value == "" {
		return nil, e.runtimeError(expr.Path, "import path cannot be empty")
	}

	var bindings []runtime.Binding
	var err error

	if strings.HasPrefix(expr.Path.Value, "std:") {
		bindings, err = e.loadStandardModule(expr, expr.Path.Value)
	} else {
		resolvedPath, resolveErr := resolveImportPath(expr.Span().File.Path(), expr.Path.Value)
		if resolveErr != nil {
			return nil, e.runtimeError(expr.Path, fmt.Sprintf("failed to resolve import %q: %v", expr.Path.Value, resolveErr))
		}
		bindings, err = e.loadModule(expr, resolvedPath)
	}

	if err != nil {
		return nil, err
	}

	env.Assign(expr.Name.Name, bindingsToRecord(bindings))
	return runtime.Nil, nil
}

func bindingsToRecord(bindings []runtime.Binding) *runtime.RecordValue {
	fields := make([]runtime.RecordField, len(bindings))
	for i, b := range bindings {
		fields[i] = runtime.RecordField{Name: b.Name, Value: b.Value}
	}
	return runtime.NewRecordValue(fields)
}

func (e *Evaluator) loadStandardModule(expr *ast.ImportExpr, path string) ([]runtime.Binding, error) {
	if bindings, ok := e.moduleCache[path]; ok {
		return cloneBindings(bindings), nil
	}

	bindings, ok := builtins.ModuleBindings(path)
	if !ok {
		return nil, e.runtimeError(expr.Path, fmt.Sprintf("unknown standard module %q", path))
	}

	e.moduleCache[path] = cloneBindings(bindings)
	return bindings, nil
}

func (e *Evaluator) loadModule(expr *ast.ImportExpr, resolvedPath string) ([]runtime.Binding, error) {
	if bindings, ok := e.moduleCache[resolvedPath]; ok {
		return cloneBindings(bindings), nil
	}

	if cycleIndex := e.moduleLoadIndex(resolvedPath); cycleIndex >= 0 {
		cycle := append(append([]string(nil), e.moduleLoadStack[cycleIndex:]...), resolvedPath)
		return nil, e.runtimeError(expr.Path, "import cycle detected: "+formatImportCycle(cycle))
	}

	e.moduleLoadStack = append(e.moduleLoadStack, resolvedPath)
	defer func() {
		e.moduleLoadStack = e.moduleLoadStack[:len(e.moduleLoadStack)-1]
	}()

	data, err := e.readFileFunc()(resolvedPath)
	if err != nil {
		return nil, e.runtimeError(expr.Path, fmt.Sprintf("import failed for %q: %v", expr.Path.Value, err))
	}

	program, err := parser.Parse(resolvedPath, string(data))
	if err != nil {
		return nil, err
	}

	moduleEnv := e.newModuleEnvironment()
	module := &moduleExecution{
		env:      moduleEnv,
		exported: make(map[string]source.Span),
	}

	e.moduleStack = append(e.moduleStack, module)
	defer func() {
		e.moduleStack = e.moduleStack[:len(e.moduleStack)-1]
	}()

	if _, err := e.evalProgramRaw(program, moduleEnv); err != nil {
		return nil, err
	}

	bindings, err := e.resolveModuleExports(module)
	if err != nil {
		return nil, err
	}

	e.moduleCache[resolvedPath] = cloneBindings(bindings)
	return bindings, nil
}

func (e *Evaluator) resolveModuleExports(module *moduleExecution) ([]runtime.Binding, error) {
	if module == nil {
		return nil, fmt.Errorf("nil module execution")
	}

	bindings := make([]runtime.Binding, 0, len(module.exportList))
	for _, name := range module.exportList {
		if !module.env.HasLocal(name) {
			span := module.exported[name]
			return nil, diagnostic.NewRuntimeError(fmt.Sprintf("exported name %q is not defined at module top level", name), span)
		}

		bindings = append(bindings, runtime.Binding{
			Name:  name,
			Value: module.env.MustGet(name),
		})
	}

	return bindings, nil
}

func (e *Evaluator) moduleLoadIndex(path string) int {
	for index, current := range e.moduleLoadStack {
		if current == path {
			return index
		}
	}

	return -1
}

func (e *Evaluator) currentModule() *moduleExecution {
	if len(e.moduleStack) == 0 {
		return nil
	}

	return e.moduleStack[len(e.moduleStack)-1]
}

func cloneBindings(bindings []runtime.Binding) []runtime.Binding {
	if len(bindings) == 0 {
		return nil
	}

	cloned := make([]runtime.Binding, len(bindings))
	copy(cloned, bindings)
	return cloned
}

func formatImportCycle(paths []string) string {
	display := make([]string, 0, len(paths))
	for _, path := range paths {
		display = append(display, filepath.ToSlash(path))
	}

	return strings.Join(display, " -> ")
}

func resolveImportPath(importerPath, importPath string) (string, error) {
	normalized := filepath.FromSlash(importPath)
	if filepath.IsAbs(normalized) {
		return filepath.Clean(normalized), nil
	}

	baseDir, err := baseDirectoryForSource(importerPath)
	if err != nil {
		return "", err
	}

	return filepath.Clean(filepath.Join(baseDir, normalized)), nil
}

func baseDirectoryForSource(path string) (string, error) {
	switch {
	case path == "", path == "-", isVirtualSourcePath(path):
		return os.Getwd()
	case filepath.IsAbs(path):
		return filepath.Dir(path), nil
	default:
		return filepath.Dir(path), nil
	}
}

func isVirtualSourcePath(path string) bool {
	return strings.HasPrefix(path, "<") && strings.HasSuffix(path, ">")
}
