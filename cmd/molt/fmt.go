package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"molt/internal/cli/exitcode"
	"molt/internal/formatter"
	"molt/internal/parser"
)

type fmtOptions struct {
	check bool
	paths []string
}

func runFmt(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	options, err := parseFmtArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		printFmtUsage(stderr)
		return exitcode.Usage
	}

	if len(options.paths) == 0 {
		options.paths = []string{"."}
	}

	if len(options.paths) == 1 && options.paths[0] == "-" {
		return runFmtStdin(options.check, stdin, stdout, stderr)
	}

	targets, exit := collectFmtTargets(options.paths, stderr)
	if exit != exitcode.Success {
		return exit
	}

	result := exitcode.Success
	for _, path := range targets {
		code := formatPath(path, options.check, stdout, stderr)
		result = mergeExitCode(result, code)
	}

	return result
}

func parseFmtArgs(args []string) (fmtOptions, error) {
	options := fmtOptions{}

	for _, arg := range args {
		switch arg {
		case "--check":
			options.check = true
		case "-":
			options.paths = append(options.paths, arg)
		default:
			if strings.HasPrefix(arg, "-") {
				return fmtOptions{}, fmt.Errorf("unknown fmt option %q", arg)
			}

			options.paths = append(options.paths, arg)
		}
	}

	if len(options.paths) > 1 {
		for _, path := range options.paths {
			if path == "-" {
				return fmtOptions{}, errors.New("'-' cannot be combined with file paths")
			}
		}
	}

	return options, nil
}

func printFmtUsage(stderr io.Writer) {
	fmt.Fprintln(stderr, "usage: molt fmt [--check] [path ...]")
}

func runFmtStdin(check bool, stdin io.Reader, stdout, stderr io.Writer) int {
	text, err := readProgramSource("-", stdin)
	if err != nil {
		fmt.Fprintf(stderr, "failed to read source from stdin: %v\n", err)
		return exitcode.SourceIO
	}

	program, err := parser.Parse("<stdin>", text)
	if err != nil {
		return reportError(err, stderr)
	}

	formatted := formatter.Format(program)
	if check {
		return checkFormatted(formatted, text)
	}

	if _, err := io.WriteString(stdout, formatted); err != nil {
		fmt.Fprintf(stderr, "failed to write formatted output: %v\n", err)
		return exitcode.SourceIO
	}

	return exitcode.Success
}

func collectFmtTargets(paths []string, stderr io.Writer) ([]string, int) {
	seen := make(map[string]struct{})
	var targets []string
	result := exitcode.Success

	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			fmt.Fprintf(stderr, "failed to read source path %q: %v\n", root, err)
			result = mergeExitCode(result, exitcode.SourceIO)
			continue
		}

		if !info.IsDir() {
			cleaned := filepath.Clean(root)
			if _, ok := seen[cleaned]; !ok {
				seen[cleaned] = struct{}{}
				targets = append(targets, cleaned)
			}
			continue
		}

		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if entry.IsDir() {
				return nil
			}

			if filepath.Ext(path) != ".molt" {
				return nil
			}

			cleaned := filepath.Clean(path)
			if _, ok := seen[cleaned]; ok {
				return nil
			}

			seen[cleaned] = struct{}{}
			targets = append(targets, cleaned)
			return nil
		})
		if err != nil {
			fmt.Fprintf(stderr, "failed to scan source path %q: %v\n", root, err)
			result = mergeExitCode(result, exitcode.SourceIO)
		}
	}

	sort.Strings(targets)
	return targets, result
}

func formatPath(path string, check bool, stdout, stderr io.Writer) int {
	text, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "failed to read source file %q: %v\n", path, err)
		return exitcode.SourceIO
	}

	program, err := parser.Parse(path, string(text))
	if err != nil {
		return reportError(err, stderr)
	}

	formatted := formatter.Format(program)
	if formatted == string(text) {
		return exitcode.Success
	}

	if check {
		return checkFormatted(formatted, string(text))
	}

	if err := replaceFileSafely(path, []byte(formatted)); err != nil {
		fmt.Fprintf(stderr, "failed to write source file %q: %v\n", path, err)
		return exitcode.SourceIO
	}

	return exitcode.Success
}

func replaceFileSafely(path string, content []byte) (err error) {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".molt-fmt-*")
	if err != nil {
		return err
	}

	tempPath := temp.Name()
	defer func() {
		if temp != nil {
			temp.Close()
		}

		if tempPath != "" {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := temp.Write(content); err != nil {
		return err
	}

	if err := temp.Chmod(info.Mode().Perm()); err != nil {
		return err
	}

	if err := temp.Close(); err != nil {
		return err
	}
	temp = nil

	backupPath := path + ".molt-fmt-backup"
	_ = os.Remove(backupPath)
	if err := os.Rename(path, backupPath); err != nil {
		return err
	}

	restoreBackup := true
	defer func() {
		if restoreBackup {
			_ = os.Remove(path)
			_ = os.Rename(backupPath, path)
		} else {
			_ = os.Remove(backupPath)
		}
	}()

	if err := os.Rename(tempPath, path); err != nil {
		return err
	}

	tempPath = ""
	restoreBackup = false
	return nil
}

func mergeExitCode(current, next int) int {
	if current == next {
		return current
	}

	switch {
	case current == exitcode.Internal || next == exitcode.Internal:
		return exitcode.Internal
	case current == exitcode.Diagnostics || next == exitcode.Diagnostics:
		return exitcode.Diagnostics
	case current == exitcode.Runtime || next == exitcode.Runtime:
		return exitcode.Runtime
	case current == exitcode.SourceIO || next == exitcode.SourceIO:
		return exitcode.SourceIO
	case current == exitcode.NeedsFormat || next == exitcode.NeedsFormat:
		return exitcode.NeedsFormat
	case current == exitcode.Usage || next == exitcode.Usage:
		return exitcode.Usage
	default:
		return exitcode.Success
	}
}

func checkFormatted(formatted, original string) int {
	if formatted != original {
		return exitcode.NeedsFormat
	}

	return exitcode.Success
}
