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

type addStep int

const (
	addStepForm addStep = iota
	addStepRunning
	addStepDone
	addStepFailed
)

type addRunMsg struct {
	result *generator.Result
	err    error
}

type addRunFunc func(context.Context, actions.AddInput) (*generator.Result, error)

type addWizard struct {
	opts   Options
	ctx    context.Context
	run    addRunFunc
	inputs []textinput.Model
	focus  int
	force  bool
	dryRun bool
	step   addStep
	err    string
	result *generator.Result
}

var addFieldLabels = []string{"Directory", "Capability", "Driver"}

func newAddWizard(opts Options, ctx context.Context, run addRunFunc) *addWizard {
	if ctx == nil {
		ctx = context.Background()
	}
	if run == nil {
		run = actions.RunAddCapability
	}
	inputs := make([]textinput.Model, len(addFieldLabels))
	for index, label := range addFieldLabels {
		input := textinput.New()
		input.Prompt = label + ": "
		input.CharLimit = 512
		input.Width = 60
		input.Cursor.SetMode(cursor.CursorStatic)
		inputs[index] = input
	}
	inputs[0].SetValue(".")
	inputs[0].Focus()
	return &addWizard{opts: opts, ctx: ctx, run: run, inputs: inputs}
}

func (m *addWizard) Init() tea.Cmd { return m.inputs[0].Focus() }

func (m *addWizard) input() actions.AddInput {
	return actions.AddInput{
		Name:       strings.TrimSpace(m.inputs[1].Value()),
		Driver:     strings.TrimSpace(m.inputs[2].Value()),
		Dir:        strings.TrimSpace(m.inputs[0].Value()),
		SourceRoot: m.opts.SourceRoot,
		Framework:  m.opts.Framework,
		Force:      m.force,
		DryRun:     m.dryRun,
	}
}

func (m *addWizard) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		width := max(20, msg.Width-4)
		for index := range m.inputs {
			m.inputs[index].Width = width
		}
		return nil, false
	case addRunMsg:
		if msg.err != nil {
			m.step = addStepFailed
			m.err = msg.err.Error()
			return nil, false
		}
		m.step = addStepDone
		m.result = msg.result
		return nil, false
	case tea.KeyMsg:
		switch m.step {
		case addStepForm:
			return m.updateForm(msg)
		case addStepDone, addStepFailed:
			return nil, msg.String() == "esc"
		}
	default:
		if m.focus < len(m.inputs) {
			var cmd tea.Cmd
			m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
			return cmd, false
		}
	}
	return nil, false
}

func (m *addWizard) updateForm(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "esc":
		return nil, true
	case "tab", "down":
		m.focusNext(1)
		return nil, false
	case "shift+tab", "up":
		m.focusNext(-1)
		return nil, false
	case " ":
		switch m.focus {
		case 3:
			m.force = !m.force
			return nil, false
		case 4:
			m.dryRun = !m.dryRun
			return nil, false
		}
	case "enter":
		if m.input().Name == "" {
			m.err = "capability name is required"
			return nil, false
		}
		m.err = ""
		m.step = addStepRunning
		input := m.input()
		return func() tea.Msg {
			result, err := m.run(m.ctx, input)
			return addRunMsg{result: result, err: err}
		}, false
	}
	if m.focus < len(m.inputs) {
		var cmd tea.Cmd
		m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
		return cmd, false
	}
	return nil, false
}

func (m *addWizard) focusNext(direction int) {
	if m.focus < len(m.inputs) {
		m.inputs[m.focus].Blur()
	}
	m.focus = (m.focus + direction + 5) % 5
	if m.focus < len(m.inputs) {
		m.inputs[m.focus].Focus()
	}
}

func (m *addWizard) View() string {
	switch m.step {
	case addStepRunning:
		return titleStyle.Render("Adding capability") + "\n\n" + m.input().Name + " → " + m.input().Dir + "\n\n" +
			helpStyle.Render("ctrl+c cancel")
	case addStepFailed:
		return failStyle.Render("✗ FAILED") + "\n\n" + m.err + "\n\n" +
			helpStyle.Render("esc back · ctrl+c quit")
	case addStepDone:
		return m.resultView()
	default:
		return m.formView()
	}
}

func (m *addWizard) formView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Add capability"))
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
	b.WriteString(m.toggleLine(3, "Force", m.force))
	b.WriteString(m.toggleLine(4, "Dry run", m.dryRun))
	if m.err != "" {
		b.WriteByte('\n')
		b.WriteString(errorStyle.Render(m.err))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(helpStyle.Render("tab next · space toggle · enter run · esc back · ctrl+c quit"))
	return b.String()
}

func (m *addWizard) toggleLine(index int, label string, value bool) string {
	check := "[ ]"
	if value {
		check = "[x]"
	}
	prefix := "  "
	if m.focus == index {
		prefix = "› "
	}
	return prefix + label + " " + check + "\n"
}

func (m *addWizard) resultView() string {
	var b strings.Builder
	b.WriteString(successStyle.Render("Capability added"))
	b.WriteByte('\n')
	if m.result != nil {
		fmt.Fprintf(&b, "  directory   %s\n", m.result.Dir)
		fmt.Fprintf(&b, "  changed     %d file(s)\n", len(m.result.Changed))
		for _, path := range m.result.Changed {
			b.WriteString("  " + path + "\n")
		}
	}
	b.WriteByte('\n')
	b.WriteString(helpStyle.Render("esc back · ctrl+c quit"))
	return b.String()
}
