package tui

import (
	"context"
	"io"
	"os"

	"jimu/tools/generator/manifest"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
)

const noTUIEnv = "JIMU_NO_TUI"

// Options is the framework CLI's TUI configuration.
type Options struct {
	Version     string
	SourceRoot  string
	Framework   *manifest.Framework
	ProjectRoot string
	In          io.Reader
	Out         io.Writer
}

// Available reports whether both input and output are terminals and the TUI
// was not explicitly disabled. The optional disabled argument lets the Cobra
// boundary pass --no-tui without duplicating the TTY/env checks.
func Available(in io.Reader, out io.Writer, disabled ...bool) bool {
	noTUI := len(disabled) > 0 && disabled[0]
	return available(isTerminal(in), isTerminal(out), noTUI)
}

func available(inTTY, outTTY, noTUI bool) bool {
	if !inTTY || !outTTY || noTUI {
		return false
	}
	return os.Getenv(noTUIEnv) != "1"
}

func isTerminal(value any) bool {
	file, ok := value.(*os.File)
	return ok && term.IsTerminal(file.Fd())
}

// Run starts the Bubble Tea program.
func Run(ctx context.Context, opts Options) error {
	programOptions := []tea.ProgramOption{tea.WithContext(ctx)}
	if opts.In != nil {
		programOptions = append(programOptions, tea.WithInput(opts.In))
	}
	if opts.Out != nil {
		programOptions = append(programOptions, tea.WithOutput(opts.Out))
	}
	_, err := tea.NewProgram(newRootModel(opts), programOptions...).Run()
	return err
}
