package builtins

// SymbolDoc describes one exported function inside a standard module.
type SymbolDoc struct {
	Signatures []string
	Summary    string
	Detail     string
}

// ModuleDoc describes one importable standard module.
type ModuleDoc struct {
	Path    string
	Summary string
	Symbols []SymbolDoc
}
