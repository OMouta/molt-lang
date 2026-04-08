package evaluator

import (
	"bufio"
	"io"
	"os"

	"molt/internal/builtins"
	"molt/internal/runtime"
)

func (e *Evaluator) prepareEnvironment(env *runtime.Environment) *runtime.Environment {
	if env == nil {
		env = runtime.NewEnvironment(nil)
	}

	builtins.Install(env)
	return env
}

func (e *Evaluator) beginRun() {
	if e.runDepth == 0 {
		e.moduleCache = make(map[string][]runtime.Binding)
		e.moduleLoadStack = nil
	}

	e.runDepth++
}

func (e *Evaluator) endRun() {
	if e.runDepth == 0 {
		return
	}

	e.runDepth--
	if e.runDepth == 0 {
		e.moduleCache = nil
		e.moduleLoadStack = nil
		e.moduleStack = nil
	}
}

func (e *Evaluator) newModuleEnvironment() *runtime.Environment {
	base := runtime.NewEnvironment(nil)
	builtins.Install(base)
	return runtime.NewEnvironment(base)
}

func (e *Evaluator) outputWriter() io.Writer {
	return outputWriter(e.output)
}

func (e *Evaluator) inputReader() io.Reader {
	return e.bufferedInputReader()
}

func (e *Evaluator) arguments() []string {
	return append([]string(nil), e.args...)
}

func (e *Evaluator) readFileFunc() func(string) ([]byte, error) {
	if e.readFile != nil {
		return e.readFile
	}

	return os.ReadFile
}

func (e *Evaluator) writeFileFunc() func(string, []byte) error {
	if e.writeFile != nil {
		return e.writeFile
	}

	return defaultWriteFile
}

func (e *Evaluator) bufferedInputReader() *bufio.Reader {
	if e.inputBuf != nil {
		return e.inputBuf
	}

	e.inputBuf = bufferedInputReader(e.input)
	return e.inputBuf
}

func inputReader(reader io.Reader) io.Reader {
	if reader != nil {
		return reader
	}

	return os.Stdin
}

func bufferedInputReader(reader io.Reader) *bufio.Reader {
	switch value := reader.(type) {
	case *bufio.Reader:
		return value
	case nil:
		return bufio.NewReader(os.Stdin)
	default:
		return bufio.NewReader(value)
	}
}

func defaultWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func outputWriter(writer io.Writer) io.Writer {
	if writer != nil {
		return writer
	}

	return os.Stdout
}
