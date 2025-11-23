package fmt

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/mitchellh/cli"
)

// StarlarkExtensions contains the file extensions that should be formatted
var StarlarkExtensions = []string{
	".pinc",
	".pconf",
	".mpconf",
	".proto-validator",
	".star",
}

type cliCommand struct{}

type cliConfig struct {
	write bool
	diff  bool
	list  bool
}

func newFlagSet() (*flag.FlagSet, *cliConfig) {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: protoconf fmt [OPTION]... [path]...")
		fmt.Fprintln(flags.Output(), "")
		fmt.Fprintln(flags.Output(), "Format starlark files (.pinc, .pconf, .mpconf, .proto-validator, .star)")
		fmt.Fprintln(flags.Output(), "")
		fmt.Fprintln(flags.Output(), "If no paths are provided, formats all starlark files in the current directory recursively.")
		fmt.Fprintln(flags.Output(), "")
		flags.PrintDefaults()
	}

	config := &cliConfig{}
	flags.BoolVar(&config.write, "w", false, "Write result to (source) file instead of stdout")
	flags.BoolVar(&config.diff, "d", false, "Display diffs instead of rewriting files")
	flags.BoolVar(&config.list, "l", false, "List files whose formatting differs from protoconf fmt's")

	return flags, config
}

func (c *cliCommand) Run(args []string) int {
	flags, config := newFlagSet()
	flags.Parse(args)

	paths := flags.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	hasErrors := false
	for _, path := range paths {
		if err := processPath(path, config); err != nil {
			slog.Error("Error processing path", "path", path, "error", err)
			hasErrors = true
		}
	}

	if hasErrors {
		return 1
	}
	return 0
}

func processPath(path string, config *cliConfig) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if isStarlarkFile(p) {
				if err := formatFile(p, config); err != nil {
					slog.Error("Error formatting file", "file", p, "error", err)
				}
			}
			return nil
		})
	}

	return formatFile(path, config)
}

func isStarlarkFile(path string) bool {
	for _, ext := range StarlarkExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

func formatFile(path string, config *cliConfig) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// Parse the file using buildtools (treats it as a .bzl-like file)
	f, err := build.ParseDefault(path, src)
	if err != nil {
		return fmt.Errorf("parsing file: %w", err)
	}

	// Format the file
	formatted := build.Format(f)

	// Check if the file is already formatted
	if bytes.Equal(src, formatted) {
		return nil
	}

	if config.list {
		fmt.Println(path)
		return nil
	}

	if config.diff {
		diff := computeDiff(path, src, formatted)
		fmt.Print(diff)
		return nil
	}

	if config.write {
		// Preserve original file permissions
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("getting file info: %w", err)
		}
		if err := os.WriteFile(path, formatted, info.Mode()); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}
		fmt.Println(path)
		return nil
	}

	// Default: print to stdout
	fmt.Print(string(formatted))
	return nil
}

func computeDiff(path string, original, formatted []byte) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("diff %s\n", path))
	buf.WriteString(fmt.Sprintf("--- %s (original)\n", path))
	buf.WriteString(fmt.Sprintf("+++ %s (formatted)\n", path))

	origLines := strings.Split(string(original), "\n")
	fmtLines := strings.Split(string(formatted), "\n")

	// Simple line-by-line diff
	maxLen := len(origLines)
	if len(fmtLines) > maxLen {
		maxLen = len(fmtLines)
	}

	for i := 0; i < maxLen; i++ {
		var origLine, fmtLine string
		if i < len(origLines) {
			origLine = origLines[i]
		}
		if i < len(fmtLines) {
			fmtLine = fmtLines[i]
		}

		if origLine != fmtLine {
			if i < len(origLines) {
				buf.WriteString(fmt.Sprintf("-%s\n", origLine))
			}
			if i < len(fmtLines) {
				buf.WriteString(fmt.Sprintf("+%s\n", fmtLine))
			}
		}
	}

	return buf.String()
}

func (c *cliCommand) Help() string {
	var b bytes.Buffer
	b.WriteString(c.Synopsis())
	b.WriteString("\n\n")
	flags, _ := newFlagSet()
	flags.SetOutput(&b)
	flags.Usage()
	return b.String()
}

func (c *cliCommand) Synopsis() string {
	return "Format starlark configuration files"
}

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
	return &cliCommand{}, nil
}
