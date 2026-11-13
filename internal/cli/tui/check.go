package tui

import (
	"context"
	"fmt"
	"strings"

	"jimu/internal/cli/actions"
	"jimu/tools/generator"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type checkStep int

const (
	checkStepForm checkStep = iota
	checkStepRunning
	checkStepDone
	checkStepFailed
)

type checkRunMsg struct {
	result *generator.CheckResult
	err    error
}

type checkRunFunc func(context.Context, actions.CheckInput) (*generator.CheckResult, error)

type checkWizard struct {
	opts   Options
	ctx    context.Context
	run    checkRunFunc
	dir    textinput.Model
	focus  int
	deep   bool
	step   checkStep
	err    string
	result *generator.CheckResult
}

func newCheckWizard(opts Options, ctx context.Context, run checkRunFunc) *checkWizard {
	if ctx == nil {
		ctx = context.Background()
	}
	if run == nil {
		run = actions.RunCheck
	}
	dir := textinput.New()
	dir.Prompt = "Directory: "
	dir.SetValue(".")
	dir.CharLimit = 512
	dir.Width = 60
	dir.Cursor.SetMode(cursor.CursorStatic)
	dir.Focus()
	return &checkWizard{opts: opts, ctx: ctx, run: run, dir: dir}
}

func (m *checkWizard) Init() tea.Cmd { return m.dir.Focus() }

func (m *checkWizard) input() actions.CheckInput {
	return actions.CheckInput{
		Dir:        strings.TrimSpace(m.dir.Value()),
		SourceRoot: m.opts.SourceRoot,
		Framework:  m.opts.Framework,
		Deep:       m.deep,
	}
}

func (m *checkWizard) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.dir.Width = max(20, msg.Width-4)
		return nil, false
	case checkRunMsg:
		if msg.err != nil {
			m.step = checkStepFailed
			m.err = msg.err.Error()
			return nil, false
		}
		m.step = checkStepDone
		m.result = msg.result
		return nil, false
	case tea.KeyMsg:
		switch m.step {
		case checkStepForm:
			return m.updateForm(msg)
		case checkStepDone, checkStepFailed:
			return nil, msg.String() == "esc"
		}
	default:
		var cmd tea.Cmd
		m.dir, cmd = m.dir.Update(msg)
		return cmd, false
	}
	return nil, false
}

func (m *checkWizard) updateForm(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "esc":
		return nil, true
	case "tab", "down", "up", "shift+tab":
		m.focus = 1 - m.focus
		if m.focus == 1 {
			m.dir.Blur()
		} else {
			m.dir.Focus()
		}
		return nil, false
	case " ":
		if m.focus == 1 {
			m.deep = !m.deep
			return nil, false
		}
	case "enter":
		m.step = checkStepRunning
		input := m.input()
		return func() tea.Msg {
			result, err := m.run(m.ctx, input)
			return checkRunMsg{result: result, err: err}
		}, false
	}
	var cmd tea.Cmd
	m.dir, cmd = m.dir.Update(msg)
	return cmd, false
}

func (m *checkWizard) View() string {
	switch m.step {
	case checkStepRunning:
		return titleStyle.Render("Checking project") + "\n\n" + m.input().Dir + "\n\n" +
			helpStyle.Render("ctrl+c cancel")
	case checkStepFailed:
		return failStyle.Render("✗ FAILED") + "\n\n" + m.err + "\n\n" +
			helpStyle.Render("esc back · ctrl+c quit")
	case checkStepDone:
		return m.resultView()
	default:
		return m.formView()
	}
}

func (m *checkWizard) formView() string {
	deep := "[ ]"
	if m.deep {
		deep = "[x]"
	}
	prefix := "  "
	if m.focus == 1 {
		prefix = "› "
	}
	return titleStyle.Render("Check project") + "\n\n" +
		m.dir.View() + "\n" +
		prefix + "Deep check " + deep + "\n\n" +
		helpStyle.Render("tab next · space toggle · enter run · esc back · ctrl+c quit")
}

func (m *checkWizard) resultView() string {
	if m.result == nil {
		return failStyle.Render("✗ FAILED") + "\n\nno result"
	}
	status, symbol, style := checkStatusLabels(m.result.Status())
	var b strings.Builder
	b.WriteString(style.Render(symbol + " " + status))
	b.WriteByte('\n')
	fmt.Fprintf(&b, "  directory   %s\n", m.result.Dir)
	fmt.Fprintf(&b, "  capabilities %d\n", len(m.result.Capabilities))
	fmt.Fprintf(&b, "  managed     %d file(s)\n", len(m.result.ManagedFiles))
	for _, diagnostic := range m.result.Diagnostics {
		fmt.Fprintf(&b, "  [%s/%s] %s\n", diagnostic.Severity, diagnostic.Code, diagnostic.Message)
	}
	b.WriteByte('\n')
	b.WriteString(helpStyle.Render("esc back · ctrl+c quit"))
	return b.String()
}

func checkStatusLabels(status generator.CheckStatus) (string, string, interface{ Render(...string) string }) {
	switch status {
	case generator.CheckStatusPassed:
		return "PASSED", "✓", passStyle
	case generator.CheckStatusWarnings:
		return "WARNINGS", "!", warnStyle
	default:
		return "FAILED", "✗", failStyle
	}
}
