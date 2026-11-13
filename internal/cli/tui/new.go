package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

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

type newExitMsg struct{}

var newExitDelay = 1200 * time.Millisecond

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

const presetLabelWidth = 8

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
	{label: "Database", defaultValue: "mysql", help: "single choice · ←/→ change"},
	{label: "Tests", defaultValue: "selected", help: "single choice · ←/→ change"},
	{label: "Tools", defaultValue: "auto", help: "single choice · ←/→ change"},
	{label: "Deploy", help: "multi-select · ←/→ move · space toggle · blank = profile default"},
}

var (
	newDBOptions     = []string{"mysql", "postgres"}
	newTestsOptions  = []string{"selected", "all", "none"}
	newToolsOptions  = []string{"auto", "all", "none"}
	newDeployParts   = []string{"compose", "k8s", "helm", "observability"}
	newDeployOptions = []string{"compose", "k8s", "helm", "observability", "none", "all"}
)

type newWizard struct {
	opts         Options
	ctx          context.Context
	run          newRunFunc
	presetCursor int
	presetQuery  string
	inputs       []textinput.Model
	focus        int
	advanced     bool
	dbChoice     int
	testsChoice  int
	toolsChoice  int
	deployCursor int
	deploy       map[string]bool
	deployNone   bool
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
		configureTextInput(&input, opts.NoColor)
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
		deploy:   map[string]bool{},
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
		DBDriver:   newDBOptions[m.dbChoice],
		Tests:      newTestsOptions[m.testsChoice],
		Tools:      newToolsOptions[m.toolsChoice],
		Deploy:     m.deployValue(),
		SourceRoot: m.opts.SourceRoot,
		Framework:  m.opts.Framework,
	}
}

func (m *newWizard) deployValue() string {
	if m.deployNone {
		return "none"
	}
	var selected []string
	for _, part := range newDeployParts {
		if m.deploy[part] {
			selected = append(selected, part)
		}
	}
	if len(selected) == len(newDeployParts) {
		return "all"
	}
	return strings.Join(selected, ",")
}

func (m *newWizard) preset() newPreset {
	return newPresets[m.presetCursor]
}

func (m *newWizard) filteredPresets() []int {
	query := strings.ToLower(strings.TrimSpace(m.presetQuery))
	if query == "" {
		indexes := make([]int, len(newPresets))
		for index := range newPresets {
			indexes[index] = index
		}
		return indexes
	}
	var indexes []int
	for index, preset := range newPresets {
		haystack := strings.ToLower(preset.name + " " + preset.label + " " + preset.description + " " + preset.detail)
		if strings.Contains(haystack, query) {
			indexes = append(indexes, index)
		}
	}
	return indexes
}

func (m *newWizard) currentPreset() (newPreset, bool) {
	for _, index := range m.filteredPresets() {
		if index == m.presetCursor {
			return newPresets[index], true
		}
	}
	return newPreset{}, false
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
		m.progress.SetContent("")
		return *m, tea.Tick(newExitDelay, func(time.Time) tea.Msg {
			return newExitMsg{}
		}), false
	case newExitMsg:
		if m.step == newStepDone {
			return *m, tea.Quit, false
		}
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
	case "up":
		m.movePreset(-1)
	case "down":
		m.movePreset(1)
	case "backspace":
		runes := []rune(m.presetQuery)
		if len(runes) > 0 {
			m.presetQuery = string(runes[:len(runes)-1])
			m.clampPresetCursor()
		}
	case "enter":
		preset, ok := m.currentPreset()
		if !ok {
			return *m, nil, false
		}
		m.inputs[newFieldProfile].SetValue(preset.name)
		m.advanced = preset.name == ""
		m.err = ""
		m.step = newStepForm
		m.focus = 0
		m.presetQuery = ""
		return *m, m.setFocus(0), false
	default:
		if msg.Type == tea.KeyRunes {
			m.presetQuery += string(msg.Runes)
			m.clampPresetCursor()
		}
	}
	return *m, nil, false
}

func (m *newWizard) movePreset(direction int) {
	indexes := m.filteredPresets()
	if len(indexes) == 0 {
		return
	}
	position := 0
	for index, value := range indexes {
		if value == m.presetCursor {
			position = index
			break
		}
	}
	position = (position + direction + len(indexes)) % len(indexes)
	m.presetCursor = indexes[position]
}

func (m *newWizard) clampPresetCursor() {
	indexes := m.filteredPresets()
	if len(indexes) == 0 {
		return
	}
	for _, index := range indexes {
		if index == m.presetCursor {
			return
		}
	}
	m.presetCursor = indexes[0]
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
	case "left", "right":
		index, ok := m.focusedFieldIndex()
		if !ok {
			return *m, nil, false
		}
		direction := 1
		if msg.String() == "left" {
			direction = -1
		}
		m.cycleChoice(index, direction)
		return *m, nil, false
	case " ":
		if m.focus == len(fields) {
			return *m, m.toggleAdvanced(), false
		}
		index, ok := m.focusedFieldIndex()
		if ok && m.isChoiceField(index) {
			if index == newFieldDeploy {
				m.toggleDeploy()
			} else {
				m.cycleChoice(index, 1)
			}
			return *m, nil, false
		}
	case "enter":
		if m.focus == len(fields) {
			return *m, m.toggleAdvanced(), false
		}
		index, ok := m.focusedFieldIndex()
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

func (m *newWizard) focusedFieldIndex() (int, bool) {
	fields := m.visibleFields()
	if m.focus < 0 || m.focus >= len(fields) {
		return 0, false
	}
	return fields[m.focus], true
}

func (m *newWizard) focusedInputIndex() (int, bool) {
	index, ok := m.focusedFieldIndex()
	if !ok || !m.isTextInput(index) {
		return 0, false
	}
	return index, true
}

func (m *newWizard) isTextInput(index int) bool {
	switch index {
	case newFieldDir, newFieldModule, newFieldWith, newFieldShape:
		return true
	default:
		return false
	}
}

func (m *newWizard) isChoiceField(index int) bool {
	switch index {
	case newFieldDB, newFieldTests, newFieldTools, newFieldDeploy:
		return true
	default:
		return false
	}
}

func (m *newWizard) cycleChoice(index, direction int) {
	switch index {
	case newFieldDB:
		m.dbChoice = cycleIndex(m.dbChoice, len(newDBOptions), direction)
	case newFieldTests:
		m.testsChoice = cycleIndex(m.testsChoice, len(newTestsOptions), direction)
	case newFieldTools:
		m.toolsChoice = cycleIndex(m.toolsChoice, len(newToolsOptions), direction)
	case newFieldDeploy:
		m.deployCursor = cycleIndex(m.deployCursor, len(newDeployOptions), direction)
	}
}

func cycleIndex(current, length, direction int) int {
	return (current + direction + length) % length
}

func (m *newWizard) toggleDeploy() {
	option := newDeployOptions[m.deployCursor]
	switch option {
	case "none":
		m.deployNone = !m.deployNone
		if m.deployNone {
			clear(m.deploy)
		}
	case "all":
		allSelected := true
		for _, part := range newDeployParts {
			allSelected = allSelected && m.deploy[part]
		}
		for _, part := range newDeployParts {
			m.deploy[part] = !allSelected
		}
		m.deployNone = false
	default:
		m.deploy[option] = !m.deploy[option]
		m.deployNone = false
	}
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
	b.WriteString(promptTitle("Create a Jimu project"))
	b.WriteString(promptLine(renderStepBar(0)))
	b.WriteString(promptBlank())
	b.WriteString(promptLine(mutedStyle.Render("Search: ") + m.presetQuery + selectedStyle.Render("▌")))
	b.WriteString(promptLine(mutedStyle.Render("↑↓ move, enter select, esc back")))
	b.WriteString(promptBlank())

	indexes := m.filteredPresets()
	if len(indexes) == 0 {
		b.WriteString(promptLine(mutedStyle.Render("No matches found")))
	} else {
		for _, index := range indexes {
			preset := newPresets[index]
			prefix := "  "
			radio := "◻"
			label := fmt.Sprintf("%-*s", presetLabelWidth, preset.label)
			if index == m.presetCursor {
				prefix = "❯ "
				radio = "◼"
				label = selectedStyle.Render(label)
			}
			b.WriteString(promptLine(prefix + radio + " " + label + "  " + presetBadge(preset)))
		}
	}

	b.WriteString(promptBlank())
	if preset, ok := m.currentPreset(); ok {
		b.WriteString(promptLine(mutedStyle.Render("Description")))
		b.WriteString(promptLine(mutedStyle.Render(preset.description)))
		b.WriteString(promptLine(mutedStyle.Render(preset.detail)))
		b.WriteString(promptBlank())
		b.WriteString(promptLine(mutedStyle.Render("Selected: ") + selectedStyle.Render(preset.label)))
	} else {
		b.WriteString(promptLine(mutedStyle.Render("Selected: (none)")))
	}
	b.WriteString(promptBlank())
	b.WriteString(promptLine(helpStyle.Render("type to search · enter continue · ctrl+c quit")))
	b.WriteString(promptBottom())
	return b.String()
}

func (m *newWizard) formView() string {
	var b strings.Builder
	b.WriteString(promptTitle("Project details"))
	b.WriteString(promptLine(renderStepBar(1)))
	b.WriteString(promptBlank())
	b.WriteString(promptLine(mutedStyle.Render("Template  ") + selectedStyle.Render(m.preset().label) + mutedStyle.Render("  ·  "+m.preset().detail)))
	b.WriteString(promptBlank())
	if m.advanced {
		b.WriteString(promptLine(mutedStyle.Render("Selectors: ←/→ single choice · Deploy: space toggles")))
		b.WriteString(promptBlank())
	}

	fields := m.visibleFields()
	for row, index := range fields {
		b.WriteString(promptLine(m.renderField(index, row == m.focus)))
		if row == m.focus {
			help := newFields[index].help
			b.WriteString(promptLine(mutedStyle.Render("  " + help)))
		}
	}
	b.WriteString(promptLine(m.advancedLine(len(fields))))
	if m.advanced {
		b.WriteString(promptLine(mutedStyle.Render("  defaults are fine; change only what you need")))
	}
	b.WriteString(promptBlank())
	if m.err != "" {
		b.WriteString(promptLine(errorStyle.Render(m.err)))
		b.WriteString(promptBlank())
	}
	b.WriteString(promptLine(helpStyle.Render("tab next · enter continue · esc back · ctrl+c quit")))
	b.WriteString(promptBottom())
	return b.String()
}

func (m *newWizard) renderField(index int, focused bool) string {
	switch index {
	case newFieldDB:
		return m.renderChoice(index, newDBOptions, m.dbChoice, focused)
	case newFieldTests:
		return m.renderChoice(index, newTestsOptions, m.testsChoice, focused)
	case newFieldTools:
		return m.renderChoice(index, newToolsOptions, m.toolsChoice, focused)
	case newFieldDeploy:
		return m.renderDeploy(focused)
	}
	input := m.inputs[index]
	prepareTextInput(&input, focused)
	if focused {
		input.Prompt = "❯ " + newFields[index].label + ": "
		input.PromptStyle = selectedStyle
	} else {
		input.Prompt = "  " + newFields[index].label + ": "
		input.PromptStyle = mutedStyle
	}
	input.Cursor.Style = selectedStyle
	return input.View()
}

func (m *newWizard) renderChoice(index int, options []string, selected int, focused bool) string {
	label := newFields[index].label
	if !focused {
		return "  " + label + "  " + options[selected]
	}
	var b strings.Builder
	b.WriteString("❯ " + label + "  ")
	for optionIndex, option := range options {
		if optionIndex > 0 {
			b.WriteString("  ")
		}
		marker := "○"
		item := marker + " " + option
		if optionIndex == selected {
			marker = "◉"
			item = selectedStyle.Render(marker + " " + option)
		} else {
			item = mutedStyle.Render(item)
		}
		b.WriteString(item)
	}
	return b.String()
}

func (m *newWizard) renderDeploy(focused bool) string {
	if !focused {
		return "  Deploy  " + valueOrDefault(m.deployValue(), "profile default")
	}
	var b strings.Builder
	b.WriteString("❯ Deploy  ")
	for optionIndex, option := range newDeployOptions {
		if optionIndex > 0 {
			b.WriteString("  ")
		}
		marker := "◻"
		if m.deployOptionSelected(option) {
			marker = "◼"
		}
		item := marker + " " + option
		if optionIndex == m.deployCursor {
			item = selectedStyle.Render(item)
		} else if m.deployOptionSelected(option) {
			item = successStyle.Render(item)
		} else {
			item = mutedStyle.Render(item)
		}
		b.WriteString(item)
	}
	return b.String()
}

func (m *newWizard) deployOptionSelected(option string) bool {
	switch option {
	case "none":
		return m.deployNone
	case "all":
		for _, part := range newDeployParts {
			if !m.deploy[part] {
				return false
			}
		}
		return true
	default:
		return m.deploy[option]
	}
}

func (m *newWizard) advancedLine(toggleRow int) string {
	check := "[ ]"
	if m.advanced {
		check = "[x]"
	}
	line := "  Advanced settings " + check
	if m.focus == toggleRow {
		return selectedStyle.Render("❯ Advanced settings " + check)
	}
	return mutedStyle.Render(line)
}

func (m *newWizard) summaryView() string {
	input := m.input()
	var b strings.Builder
	b.WriteString(promptTitle("Review and create"))
	b.WriteString(promptLine(renderStepBar(2)))
	b.WriteString(promptBlank())
	b.WriteString(promptLine(successStyle.Render("✓ Ready to create")))
	b.WriteString(promptBlank())
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
		b.WriteString(promptLine("  " + line))
	}
	b.WriteString(promptBlank())
	if m.err != "" {
		b.WriteString(promptLine(errorStyle.Render(m.err)))
		b.WriteString(promptBlank())
	}
	b.WriteString(promptLine(helpStyle.Render("enter create · esc edit · ctrl+c quit")))
	b.WriteString(promptBottom())
	return b.String()
}

func (m *newWizard) runningView() string {
	var b strings.Builder
	b.WriteString(promptTitle("Creating project"))
	b.WriteString(promptLine(renderStepBar(2)))
	b.WriteString(promptBlank())
	b.WriteString(promptLine(m.spinner.View() + mutedStyle.Render(" generating files, checking the project...")))
	b.WriteString(promptBlank())
	b.WriteString(promptBlock(m.progress.View()))
	b.WriteString(promptLine(helpStyle.Render("ctrl+c cancel")))
	b.WriteString(promptBottom())
	return b.String()
}

func (m *newWizard) doneView() string {
	var b strings.Builder
	b.WriteString(promptSuccessTitle("Project created"))
	if m.result != nil {
		b.WriteString(promptBlank())
		b.WriteString(promptLine(fmt.Sprintf("directory   %s", m.result.Dir)))
		b.WriteString(promptLine(fmt.Sprintf("profile     %s", selectionLabel(m.result.Template))))
		b.WriteString(promptLine(fmt.Sprintf("support     %s", m.result.Support)))
		b.WriteString(promptLine(fmt.Sprintf("files       %d", m.result.FileCount)))
	}
	b.WriteString(promptBlank())
	b.WriteString(promptLine(mutedStyle.Render("returning to shell...")))
	b.WriteString(promptBottom())
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

func promptTitle(title string) string {
	return selectedStyle.Render("◆") + "  " + titleStyle.Render(title) + "\n"
}

func promptSuccessTitle(title string) string {
	return successStyle.Render("◇") + "  " + successStyle.Render(title) + "\n"
}

func promptLine(line string) string {
	return mutedStyle.Render("│") + "  " + line + "\n"
}

func promptBlank() string {
	return mutedStyle.Render("│") + "\n"
}

func promptBlock(block string) string {
	var b strings.Builder
	for _, line := range strings.Split(strings.TrimRight(block, "\n"), "\n") {
		b.WriteString(promptLine(line))
	}
	return b.String()
}

func promptBottom() string {
	return mutedStyle.Render("└") + "\n"
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
