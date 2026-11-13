package tui

import (
	"context"
	"fmt"
	"strings"

	"jimu/internal/cli/actions"
	"jimu/tools/generator"
	upgradeplan "jimu/tools/generator/upgrade"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type upgradeMode int

const (
	upgradeModeApply upgradeMode = iota
	upgradeModeDryRun
	upgradeModeCheck
)

var upgradeModeLabels = []string{"apply", "dry-run", "check"}

type upgradeStep int

const (
	upgradeStepForm upgradeStep = iota
	upgradeStepConfirm
	upgradeStepRunning
	upgradeStepDone
	upgradeStepFailed
)

type upgradeRunMsg struct {
	plan    *generator.UpgradeResult
	applied *generator.UpgradeApplyResult
	err     error
}

type upgradeRunFunc func(context.Context, actions.UpgradeInput) (*generator.UpgradeResult, *generator.UpgradeApplyResult, error)

type upgradeWizard struct {
	opts    Options
	ctx     context.Context
	run     upgradeRunFunc
	dir     textinput.Model
	focus   int
	mode    upgradeMode
	step    upgradeStep
	err     string
	plan    *generator.UpgradeResult
	applied *generator.UpgradeApplyResult
}

func newUpgradeWizard(opts Options, ctx context.Context, run upgradeRunFunc) *upgradeWizard {
	if ctx == nil {
		ctx = context.Background()
	}
	if run == nil {
		run = actions.RunUpgrade
	}
	dir := textinput.New()
	dir.Prompt = "Directory: "
	dir.SetValue(".")
	dir.CharLimit = 512
	dir.Width = 60
	configureTextInput(&dir, opts.NoColor)
	dir.Focus()
	return &upgradeWizard{opts: opts, ctx: ctx, run: run, dir: dir}
}

func (m *upgradeWizard) Init() tea.Cmd { return m.dir.Focus() }

func (m *upgradeWizard) input() actions.UpgradeInput {
	return actions.UpgradeInput{
		Dir:        strings.TrimSpace(m.dir.Value()),
		SourceRoot: m.opts.SourceRoot,
		Framework:  m.opts.Framework,
		DryRun:     m.mode == upgradeModeDryRun,
		Check:      m.mode == upgradeModeCheck,
	}
}

func (m *upgradeWizard) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.dir.Width = max(20, msg.Width-4)
		return nil, false
	case upgradeRunMsg:
		if msg.err != nil && msg.plan == nil && msg.applied == nil {
			m.step = upgradeStepFailed
			m.err = msg.err.Error()
			return nil, false
		}
		m.step = upgradeStepDone
		m.plan = msg.plan
		m.applied = msg.applied
		if msg.err != nil {
			m.err = msg.err.Error()
		}
		return nil, false
	case tea.KeyMsg:
		switch m.step {
		case upgradeStepForm:
			return m.updateForm(msg)
		case upgradeStepConfirm:
			return m.updateConfirm(msg)
		case upgradeStepDone, upgradeStepFailed:
			return nil, msg.String() == "esc"
		}
	default:
		var cmd tea.Cmd
		m.dir, cmd = m.dir.Update(msg)
		return cmd, false
	}
	return nil, false
}

func (m *upgradeWizard) updateForm(msg tea.KeyMsg) (tea.Cmd, bool) {
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
	case "left", "right", " ":
		if m.focus == 1 {
			if msg.String() == "left" {
				m.mode = (m.mode + 2) % 3
			} else {
				m.mode = (m.mode + 1) % 3
			}
			return nil, false
		}
	case "enter":
		if m.mode == upgradeModeApply {
			m.step = upgradeStepConfirm
			return nil, false
		}
		return m.start(), false
	}
	var cmd tea.Cmd
	m.dir, cmd = m.dir.Update(msg)
	return cmd, false
}

func (m *upgradeWizard) updateConfirm(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "esc":
		m.step = upgradeStepForm
		return nil, false
	case "enter":
		return m.start(), false
	}
	return nil, false
}

func (m *upgradeWizard) start() tea.Cmd {
	m.step = upgradeStepRunning
	input := m.input()
	return func() tea.Msg {
		plan, applied, err := m.run(m.ctx, input)
		return upgradeRunMsg{plan: plan, applied: applied, err: err}
	}
}

func (m *upgradeWizard) View() string {
	switch m.step {
	case upgradeStepConfirm:
		return titleStyle.Render("Confirm upgrade") + "\n\n" +
			"Apply the upgrade to " + m.input().Dir + "?\n" +
			"Changed files are backed up and the project is verified; failures roll back.\n\n" +
			helpStyle.Render("enter apply · esc back · ctrl+c quit")
	case upgradeStepRunning:
		return titleStyle.Render("Upgrading project") + "\n\n" + m.input().Dir + "\n\n" +
			helpStyle.Render("ctrl+c cancel")
	case upgradeStepFailed:
		return failStyle.Render("✗ FAILED") + "\n\n" + m.err + "\n\n" +
			helpStyle.Render("esc back · ctrl+c quit")
	case upgradeStepDone:
		return m.resultView()
	default:
		return m.formView()
	}
}

func (m *upgradeWizard) formView() string {
	prefix := "  "
	if m.focus == 1 {
		prefix = "› "
	}
	dir := m.dir
	prepareTextInput(&dir, m.focus == 0)
	return titleStyle.Render("Upgrade project") + "\n\n" +
		dir.View() + "\n" +
		prefix + "Mode " + upgradeModeLabels[m.mode] + "\n\n" +
		helpStyle.Render("tab next · left/right mode · enter continue · esc back · ctrl+c quit")
}

func (m *upgradeWizard) resultView() string {
	report, dir := m.report()
	status, symbol, style := upgradeStatusLabels(report.Status())
	var b strings.Builder
	b.WriteString(style.Render(symbol + " " + status))
	b.WriteByte('\n')
	fmt.Fprintf(&b, "  directory   %s\n", dir)
	fmt.Fprintf(&b, "  conflicts   %d\n", report.Count(upgradeplan.Conflict))
	for _, entry := range report.Entries {
		if entry.Category == upgradeplan.Conflict {
			fmt.Fprintf(&b, "  conflict    %s\n", entry.Path)
		}
	}
	if m.applied != nil {
		fmt.Fprintf(&b, "  changed     %d file(s)\n", len(m.applied.Changed))
		for _, path := range m.applied.Changed {
			b.WriteString("  install     " + path + "\n")
		}
	}
	if m.err != "" {
		b.WriteByte('\n')
		b.WriteString(warnStyle.Render(m.err))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(helpStyle.Render("esc back · ctrl+c quit"))
	return b.String()
}

func (m *upgradeWizard) report() (upgradeplan.Report, string) {
	if m.applied != nil {
		return m.applied.Report, m.applied.Dir
	}
	if m.plan != nil {
		return m.plan.Report, m.plan.Dir
	}
	return upgradeplan.Report{}, m.input().Dir
}

func upgradeStatusLabels(status upgradeplan.Status) (string, string, interface{ Render(...string) string }) {
	switch status {
	case upgradeplan.StatusConflict:
		return "CONFLICT", "!", failStyle
	case upgradeplan.StatusUpgradeAvailable:
		return "UPGRADE AVAILABLE", "!", warnStyle
	default:
		return "UP-TO-DATE", "✓", passStyle
	}
}
