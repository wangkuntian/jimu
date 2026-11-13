package tui

import (
	"context"
	"fmt"
	"strings"

	"jimu/internal/cli/actions"
	"jimu/tools/generator"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type newStep int

const (
	newStepPreset newStep = iota
	newStepForm
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

type newPreset struct {
	name        string
	label       string
	badge       string
	description string
	detail      string
}

var newPresets = []newPreset{
	{
		name:        "minimal",
		label:       "Minimal",
		badge:       "lean",
		description: "A clean service with no external dependencies.",
		detail:      "fastest start · no database or Redis",
	},
	{
		name:        "auth",
		label:       "Auth",
		badge:       "ready",
		description: "Users, roles, sessions and access control.",
		detail:      "MySQL + Redis · compose included",
	},
	{
		name:        "full",
		label:       "Full",
		badge:       "complete",
		description: "Every capability for exploration and demos.",
		detail:      "all capabilities · validated support",
	},
	{
		label:       "Custom",
		badge:       "expert",
		description: "Choose capabilities yourself with --with.",
		detail:      "manual selection · experimental",
	},
}

const (
	newFieldDir = iota
	newFieldModule
	newFieldProfile
	newFieldWith
	newFieldShape
	newFieldDB
	newFieldTests
	newFieldTools
	newFieldDeploy
)

type newField struct {
	label        string
	defaultValue string
	help         string
}

var newFields = []newField{
	{label: "Directory", help: "e.g. my-service · required · the path is created atomically"},
	{label: "Go module", help: "e.g. example.com/my-service · optional · blank derives it from the directory"},
	{label: "Profile", defaultValue: "minimal"},
	{label: "Capabilities", help: "e.g. user,access,auth · custom preset only · comma-separated names"},
	{label: "Shape", help: "e.g. app · custom preset only · defaults to app"},
	{label: "Database", defaultValue: "mysql", help: "mysql or postgres"},
	{label: "Tests", defaultValue: "selected", help: "selected, all, or none"},
	{label: "Tools", defaultValue: "auto", help: "auto, all, or none"},
	{label: "Deploy", help: "e.g. compose,k8s · or all/none · blank uses the profile default"},
}

type newWizard struct {
	opts         Options
	ctx          context.Context
	run          newRunFunc
	presetCursor int
	inputs       []textinput.Model
	focus        int
	advanced     bool
	step         newStep
	err          string
	progress     viewport.Model
	spinner      spinner.Model
	result       *generator.Result
	submits      int
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
		input.PromptStyle = mutedStyle
		input.Cursor.Style = selectedStyle
		inputs = append(inputs, input)
	}
	spin := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	spin.Style = successStyle
	progress := viewport.New(60, 6)
	return &newWizard{
		opts:     opts,
		ctx:      ctx,
		run:      run,
		inputs:   inputs,
		progress: progress,
		spinner:  spin,
	}
}

func (m *newWizard) Init() tea.Cmd { return nil }

func (m *newWizard) input() actions.NewInput {
	value := func(index int) string { return strings.TrimSpace(m.inputs[index].Value()) }
	return actions.NewInput{
		Dir:        value(newFieldDir),
		Module:     value(newFieldModule),
		Profile:    value(newFieldProfile),
		With:       value(newFieldWith),
		Shape:      value(newFieldShape),
		DBDriver:   value(newFieldDB),
		Tests:      value(newFieldTests),
		Tools:      value(newFieldTools),
		Deploy:     value(newFieldDeploy),
		SourceRoot: m.opts.SourceRoot,
		Framework:  m.opts.Framework,
	}
}

func (m *newWizard) preset() newPreset {
	return newPresets[m.presetCursor]
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
		case newStepPreset:
			return m.updatePreset(msg)
		case newStepForm:
			return m.updateForm(msg)
		case newStepSummary:
			return m.updateSummary(msg)
		case newStepDone, newStepFailed:
			if msg.String() == "esc" {
				return *m, nil, true
			}
		}
		return *m, nil, false
	default:
		var cmds []tea.Cmd
		if index, ok := m.focusedInputIndex(); ok {
			var cmd tea.Cmd
			m.inputs[index], cmd = m.inputs[index].Update(msg)
			cmds = append(cmds, cmd)
		}
		var spinnerCmd tea.Cmd
		if m.step == newStepRunning {
			m.spinner, spinnerCmd = m.spinner.Update(msg)
			cmds = append(cmds, spinnerCmd)
		}
		return *m, tea.Batch(cmds...), false
	}
}

func (m *newWizard) updatePreset(msg tea.KeyMsg) (newWizard, tea.Cmd, bool) {
	switch msg.String() {
	case "esc":
		return *m, nil, true
	case "up", "k":
		if m.presetCursor > 0 {
			m.presetCursor--
		}
	case "down", "j":
		if m.presetCursor < len(newPresets)-1 {
			m.presetCursor++
		}
	case "enter", " ":
		preset := m.preset()
		m.inputs[newFieldProfile].SetValue(preset.name)
		m.advanced = preset.name == ""
		m.err = ""
		m.step = newStepForm
		m.focus = 0
		return *m, m.setFocus(0), false
	}
	return *m, nil, false
}

func (m *newWizard) updateForm(msg tea.KeyMsg) (newWizard, tea.Cmd, bool) {
	fields := m.visibleFields()
	switch msg.String() {
	case "esc":
		m.blurInputs()
		m.step = newStepPreset
		return *m, nil, false
	case "tab", "down":
		return *m, m.setFocus((m.focus + 1) % (len(fields) + 1)), false
	case "shift+tab", "up":
		return *m, m.setFocus((m.focus - 1 + len(fields) + 1) % (len(fields) + 1)), false
	case " ":
		if m.focus == len(fields) {
			return *m, m.toggleAdvanced(), false
		}
	case "enter":
		if m.focus == len(fields) {
			return *m, m.toggleAdvanced(), false
		}
		index, ok := m.focusedInputIndex()
		if !ok {
			return *m, nil, false
		}
		if index == newFieldDir && strings.TrimSpace(m.inputs[index].Value()) == "" {
			m.err = "directory is required"
			return *m, nil, false
		}
		if m.focus == len(fields)-1 {
			return m.submitDetails()
		}
		m.err = ""
		return *m, m.setFocus(m.focus + 1), false
	}
	index, ok := m.focusedInputIndex()
	if !ok {
		return *m, nil, false
	}
	var cmd tea.Cmd
	m.inputs[index], cmd = m.inputs[index].Update(msg)
	if msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete {
		m.err = ""
	}
	return *m, cmd, false
}

func (m *newWizard) updateSummary(msg tea.KeyMsg) (newWizard, tea.Cmd, bool) {
	switch msg.String() {
	case "esc":
		m.step = newStepForm
		return *m, m.setFocus(0), false
	case "enter":
		if m.submits > 0 {
			return *m, nil, false
		}
		if err := m.validate(); err != nil {
			m.err = err.Error()
			m.step = newStepForm
			return *m, m.setFocusForError(), false
		}
		m.err = ""
		m.submits++
		m.step = newStepRunning
		m.progress.SetContent("validating inputs...\ngenerating project...\n")
		return *m, tea.Batch(m.runCmd(), m.spinner.Tick), false
	}
	return *m, nil, false
}

func (m *newWizard) submitDetails() (newWizard, tea.Cmd, bool) {
	if err := m.validate(); err != nil {
		m.err = err.Error()
		return *m, nil, false
	}
	m.err = ""
	m.blurInputs()
	m.step = newStepSummary
	return *m, nil, false
}

func (m *newWizard) validate() error {
	if strings.TrimSpace(m.inputs[newFieldDir].Value()) == "" {
		return fmt.Errorf("directory is required")
	}
	if m.preset().name == "" && strings.TrimSpace(m.inputs[newFieldWith].Value()) == "" {
		return fmt.Errorf("custom preset needs at least one capability")
	}
	return actions.ValidateNew(m.input())
}

func (m *newWizard) visibleFields() []int {
	fields := []int{newFieldDir, newFieldModule}
	if !m.advanced {
		return fields
	}
	if m.preset().name == "" {
		fields = append(fields, newFieldWith, newFieldShape)
	}
	return append(fields, newFieldDB, newFieldTests, newFieldTools, newFieldDeploy)
}

func (m *newWizard) focusedInputIndex() (int, bool) {
	fields := m.visibleFields()
	if m.focus < 0 || m.focus >= len(fields) {
		return 0, false
	}
	return fields[m.focus], true
}

func (m *newWizard) setFocus(row int) tea.Cmd {
	rows := len(m.visibleFields()) + 1
	if row < 0 {
		row = 0
	}
	m.focus = row % rows
	m.blurInputs()
	if index, ok := m.focusedInputIndex(); ok {
		return m.inputs[index].Focus()
	}
	return nil
}

func (m *newWizard) setFocusForError() tea.Cmd {
	if m.preset().name == "" && strings.TrimSpace(m.inputs[newFieldWith].Value()) == "" {
		m.advanced = true
		return m.setFocus(m.rowForField(newFieldWith))
	}
	return m.setFocus(m.rowForField(newFieldDir))
}

func (m *newWizard) rowForField(field int) int {
	for row, index := range m.visibleFields() {
		if index == field {
			return row
		}
	}
	return 0
}

func (m *newWizard) blurInputs() {
	for index := range m.inputs {
		m.inputs[index].Blur()
	}
}

func (m *newWizard) toggleAdvanced() tea.Cmd {
	m.advanced = !m.advanced
	m.err = ""
	if m.focus > len(m.visibleFields()) {
		m.focus = len(m.visibleFields())
	}
	return m.setFocus(m.focus)
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
	case newStepPreset:
		return m.presetView()
	case newStepSummary:
		return m.summaryView()
	case newStepRunning:
		return m.runningView()
	case newStepDone:
		return m.doneView()
	case newStepFailed:
		return errorStyle.Render("✗ Project generation failed") + "\n\n" + m.err + "\n\n" +
			helpStyle.Render("esc back · ctrl+c quit")
	default:
		return m.formView()
	}
}

func (m *newWizard) presetView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Create a Jimu project"))
	b.WriteByte('\n')
	b.WriteString(renderStepBar(0))
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("Pick a starting point. You can add capabilities after generation."))
	b.WriteString("\n\n")
	for index, preset := range newPresets {
		marker := "  "
		line := preset.label + "  " + presetBadge(preset)
		if index == m.presetCursor {
			marker = "› "
			b.WriteString(selectedStyle.Render(marker + line))
		} else {
			b.WriteString(marker + line)
		}
		b.WriteByte('\n')
		b.WriteString(mutedStyle.Render("   " + preset.description))
		b.WriteByte('\n')
		if index == m.presetCursor {
			b.WriteString(mutedStyle.Render("   " + preset.detail))
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	b.WriteString(helpStyle.Render("↑/↓ choose · enter continue · esc back · ctrl+c quit"))
	return b.String()
}

func (m *newWizard) formView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Project details"))
	b.WriteByte('\n')
	b.WriteString(renderStepBar(1))
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("Template  ") + selectedStyle.Render(m.preset().label))
	b.WriteString(mutedStyle.Render("  ·  " + m.preset().detail))
	b.WriteString("\n\n")

	fields := m.visibleFields()
	for row, index := range fields {
		b.WriteString(m.renderField(index, row == m.focus))
		b.WriteByte('\n')
		if row == m.focus {
			b.WriteString(mutedStyle.Render("   " + newFields[index].help))
			b.WriteByte('\n')
		}
	}
	b.WriteString(m.advancedLine(len(fields)))
	if m.advanced {
		b.WriteString(mutedStyle.Render("   defaults are fine; change only what you need"))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err))
		b.WriteString("\n\n")
	}
	b.WriteString(helpStyle.Render("tab next · enter continue · esc back · ctrl+c quit"))
	return b.String()
}

func (m *newWizard) renderField(index int, focused bool) string {
	input := m.inputs[index]
	if focused {
		input.Prompt = "▌ " + newFields[index].label + ": "
		input.PromptStyle = selectedStyle
	} else {
		input.Prompt = "  " + newFields[index].label + ": "
		input.PromptStyle = mutedStyle
	}
	input.Cursor.Style = selectedStyle
	return input.View()
}

func (m *newWizard) advancedLine(toggleRow int) string {
	check := "[ ]"
	if m.advanced {
		check = "[x]"
	}
	line := "  Advanced settings " + check
	if m.focus == toggleRow {
		return selectedStyle.Render("▌ Advanced settings " + check)
	}
	return mutedStyle.Render(line)
}

func (m *newWizard) summaryView() string {
	input := m.input()
	var b strings.Builder
	b.WriteString(titleStyle.Render("Review and create"))
	b.WriteByte('\n')
	b.WriteString(renderStepBar(2))
	b.WriteString("\n\n")
	b.WriteString(successStyle.Render("✓ Ready to create"))
	b.WriteString("\n\n")
	for _, line := range []string{
		"template    " + m.preset().label,
		"directory   " + input.Dir,
		"module      " + valueOrDefault(input.Module, "derived from directory"),
		"with        " + valueOrDefault(input.With, "preset capabilities"),
		"database    " + valueOrDefault(input.DBDriver, "mysql"),
		"tests       " + valueOrDefault(input.Tests, "selected"),
		"tools       " + valueOrDefault(input.Tools, "auto"),
		"deploy      " + valueOrDefault(input.Deploy, "profile default"),
	} {
		b.WriteString("  " + line + "\n")
	}
	b.WriteByte('\n')
	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err))
		b.WriteString("\n\n")
	}
	b.WriteString(helpStyle.Render("enter create · esc edit · ctrl+c quit"))
	return b.String()
}

func (m *newWizard) runningView() string {
	return titleStyle.Render("Creating project") + "\n" +
		renderStepBar(2) + "\n\n" +
		m.spinner.View() + mutedStyle.Render(" generating files, checking the project...") + "\n\n" +
		m.progress.View() + "\n" +
		helpStyle.Render("ctrl+c cancel")
}

func (m *newWizard) doneView() string {
	var b strings.Builder
	b.WriteString(successStyle.Render("✓ Project created"))
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

func renderStepBar(active int) string {
	labels := []string{"1 Template", "2 Details", "3 Review"}
	var b strings.Builder
	for index, label := range labels {
		if index > 0 {
			b.WriteString(mutedStyle.Render("  ›  "))
		}
		switch {
		case index < active:
			b.WriteString(passStyle.Render("✓ " + label))
		case index == active:
			b.WriteString(titleStyle.Render("● " + label))
		default:
			b.WriteString(mutedStyle.Render("○ " + label))
		}
	}
	return b.String()
}

func presetBadge(preset newPreset) string {
	switch preset.badge {
	case "lean":
		return passStyle.Render(preset.badge)
	case "ready":
		return successStyle.Render(preset.badge)
	case "complete":
		return warnStyle.Render(preset.badge)
	default:
		return mutedStyle.Render(preset.badge)
	}
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
