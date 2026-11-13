package tui

import (
	"context"
	"fmt"
	"strings"

	"jimu/internal/cli/actions"
	"jimu/tools/generator"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type newStep int

const (
	newStepForm newStep = iota
	newStepSummary
	newStepRunning
	newStepDone
	newStepFailed
)

type newRunMsg struct {
	result *generator.Result
	err    error
}

type newRunFunc func(context.Context, actions.NewInput) (*generator.Result, error)

type newField struct {
	label        string
	defaultValue string
}

var newFields = []newField{
	{label: "Directory"},
	{label: "Module"},
	{label: "Profile", defaultValue: "minimal"},
	{label: "Capabilities (--with)"},
	{label: "Shape"},
	{label: "Database driver", defaultValue: "mysql"},
	{label: "Tests", defaultValue: "selected"},
	{label: "Tools", defaultValue: "auto"},
	{label: "Deploy components"},
}

type newWizard struct {
	opts     Options
	ctx      context.Context
	run      newRunFunc
	inputs   []textinput.Model
	focus    int
	step     newStep
	err      string
	progress viewport.Model
	result   *generator.Result
	submits  int
}

func newNewWizard(opts Options, ctx context.Context, run newRunFunc) *newWizard {
	if ctx == nil {
		ctx = context.Background()
	}
	if run == nil {
		run = actions.RunNew
	}
	inputs := make([]textinput.Model, 0, len(newFields))
	for _, field := range newFields {
		input := textinput.New()
		input.Prompt = field.label + ": "
		input.SetValue(field.defaultValue)
		input.CharLimit = 512
		input.Width = 60
		inputs = append(inputs, input)
	}
	inputs[0].Focus()
	progress := viewport.New(60, 6)
	return &newWizard{
		opts:     opts,
		ctx:      ctx,
		run:      run,
		inputs:   inputs,
		progress: progress,
	}
}

func (m *newWizard) input() actions.NewInput {
	value := func(index int) string { return strings.TrimSpace(m.inputs[index].Value()) }
	return actions.NewInput{
		Dir:        value(0),
		Module:     value(1),
		Profile:    value(2),
		With:       value(3),
		Shape:      value(4),
		DBDriver:   value(5),
		Tests:      value(6),
		Tools:      value(7),
		Deploy:     value(8),
		SourceRoot: m.opts.SourceRoot,
		Framework:  m.opts.Framework,
	}
}

func (m *newWizard) Update(msg tea.Msg) (newWizard, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		return *m, nil, false
	case newRunMsg:
		m.progress.SetContent(m.progress.View() + "\nfinalizing project...\n")
		if msg.err != nil {
			m.step = newStepFailed
			m.err = msg.err.Error()
			return *m, nil, false
		}
		m.step = newStepDone
		m.result = msg.result
		return *m, nil, false
	case tea.KeyMsg:
		switch m.step {
		case newStepForm:
			return m.updateForm(msg)
		case newStepSummary:
			return m.updateSummary(msg)
		case newStepDone, newStepFailed:
			if msg.String() == "esc" {
				return *m, nil, true
			}
		}
	}
	return *m, nil, false
}

func (m *newWizard) updateForm(msg tea.KeyMsg) (newWizard, tea.Cmd, bool) {
	switch msg.String() {
	case "esc":
		return *m, nil, true
	case "tab", "down":
		m.focusNext(1)
		return *m, nil, false
	case "shift+tab", "up":
		m.focusNext(-1)
		return *m, nil, false
	case "enter":
		if strings.TrimSpace(m.inputs[0].Value()) == "" {
			m.err = "directory is required"
			return *m, nil, false
		}
		if err := actions.ValidateNew(m.input()); err != nil {
			m.err = err.Error()
			return *m, nil, false
		}
		m.err = ""
		m.step = newStepSummary
		return *m, nil, false
	}
	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	return *m, cmd, false
}

func (m *newWizard) updateSummary(msg tea.KeyMsg) (newWizard, tea.Cmd, bool) {
	switch msg.String() {
	case "esc":
		m.step = newStepForm
		return *m, nil, false
	case "enter":
		if m.submits > 0 {
			return *m, nil, false
		}
		if err := actions.ValidateNew(m.input()); err != nil {
			m.err = err.Error()
			m.step = newStepForm
			return *m, nil, false
		}
		m.err = ""
		m.submits++
		m.step = newStepRunning
		m.progress.SetContent("validating inputs...\ngenerating project...\n")
		return *m, m.runCmd(), false
	}
	return *m, nil, false
}

func (m *newWizard) focusNext(direction int) {
	m.inputs[m.focus].Blur()
	m.focus = (m.focus + direction + len(m.inputs)) % len(m.inputs)
	m.inputs[m.focus].Focus()
}

func (m *newWizard) runCmd() tea.Cmd {
	input := m.input()
	return func() tea.Msg {
		result, err := m.run(m.ctx, input)
		return newRunMsg{result: result, err: err}
	}
}

func (m *newWizard) resize(width, height int) {
	inputWidth := max(20, width-4)
	for index := range m.inputs {
		m.inputs[index].Width = inputWidth
	}
	m.progress.Width = inputWidth
	m.progress.Height = max(3, height-14)
}

func (m *newWizard) View() string {
	switch m.step {
	case newStepSummary:
		return m.summaryView()
	case newStepRunning:
		return titleStyle.Render("Creating project") + "\n\n" + m.progress.View() + "\n" +
			helpStyle.Render("ctrl+c cancel")
	case newStepDone:
		return m.doneView()
	case newStepFailed:
		return errorStyle.Render("Project generation failed") + "\n\n" + m.err + "\n\n" +
			helpStyle.Render("esc back · ctrl+c quit")
	default:
		return m.formView()
	}
}

func (m *newWizard) formView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("New project"))
	b.WriteByte('\n')
	b.WriteString(mutedStyle.Render("Choose a preset, or clear Profile and enter capabilities with --with."))
	b.WriteString("\n\n")
	for index := range m.inputs {
		line := m.inputs[index].View()
		if index == m.focus {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	if m.err != "" {
		b.WriteByte('\n')
		b.WriteString(errorStyle.Render(m.err))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(helpStyle.Render("tab next · shift+tab previous · enter review · esc back · ctrl+c quit"))
	return b.String()
}

func (m *newWizard) summaryView() string {
	input := m.input()
	var b strings.Builder
	b.WriteString(titleStyle.Render("Review new project"))
	b.WriteString("\n\n")
	for _, line := range []string{
		"directory   " + input.Dir,
		"module      " + valueOrDefault(input.Module, "derived from directory"),
		"profile     " + valueOrDefault(input.Profile, "minimal (default)"),
		"with        " + valueOrDefault(input.With, "none"),
		"shape       " + valueOrDefault(input.Shape, "app"),
		"db          " + valueOrDefault(input.DBDriver, "mysql"),
		"tests       " + valueOrDefault(input.Tests, "selected"),
		"tools       " + valueOrDefault(input.Tools, "auto"),
		"deploy      " + valueOrDefault(input.Deploy, "profile default"),
	} {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter create · esc edit · ctrl+c quit"))
	return b.String()
}

func (m *newWizard) doneView() string {
	var b strings.Builder
	b.WriteString(successStyle.Render("Project created"))
	b.WriteByte('\n')
	if m.result != nil {
		fmt.Fprintf(&b, "  directory   %s\n", m.result.Dir)
		fmt.Fprintf(&b, "  profile     %s\n", selectionLabel(m.result.Template))
		fmt.Fprintf(&b, "  support     %s\n", m.result.Support)
		fmt.Fprintf(&b, "  files       %d\n", m.result.FileCount)
	}
	b.WriteByte('\n')
	b.WriteString(m.progress.View())
	b.WriteByte('\n')
	b.WriteString(helpStyle.Render("esc menu · ctrl+c quit"))
	return b.String()
}

func valueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func selectionLabel(value string) string {
	if value == "" {
		return "minimal"
	}
	return value
}
