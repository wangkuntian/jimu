package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Screen identifies a TUI page.
type Screen int

const (
	ScreenMenu Screen = iota
	ScreenNew
	ScreenCheck
	ScreenUpgrade
	ScreenAdd
)

type menuItem struct {
	label  string
	screen Screen
}

var menuItems = []menuItem{
	{label: "New project", screen: ScreenNew},
	{label: "Check project", screen: ScreenCheck},
	{label: "Upgrade project", screen: ScreenUpgrade},
	{label: "Add capability", screen: ScreenAdd},
}

type keyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Back  key.Binding
	Quit  key.Binding
}

var keys = keyMap{
	Up:    key.NewBinding(key.WithKeys("up", "k")),
	Down:  key.NewBinding(key.WithKeys("down", "j")),
	Enter: key.NewBinding(key.WithKeys("enter")),
	Back:  key.NewBinding(key.WithKeys("esc")),
	Quit:  key.NewBinding(key.WithKeys("q", "ctrl+c")),
}

type rootModel struct {
	opts          Options
	ctx           context.Context
	cursor        int
	screen        Screen
	quit          bool
	newWizard     *newWizard
	checkWizard   *checkWizard
	upgradeWizard *upgradeWizard
	addWizard     *addWizard
}

func newRootModel(opts Options) rootModel {
	return rootModel{opts: opts, ctx: context.Background()}
}

func (m rootModel) Init() tea.Cmd { return nil }

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if key.Matches(keyMsg, keys.Quit) {
		m.quit = true
		return m, tea.Quit
	}
	switch m.screen {
	case ScreenNew:
		if m.newWizard != nil {
			updated, cmd, back := m.newWizard.Update(msg)
			m.newWizard = &updated
			if back {
				m.screen = ScreenMenu
				m.newWizard = nil
			}
			return m, cmd
		}
	case ScreenCheck:
		if m.checkWizard != nil {
			cmd, back := m.checkWizard.Update(msg)
			if back {
				m.screen = ScreenMenu
				m.checkWizard = nil
			}
			return m, cmd
		}
	case ScreenUpgrade:
		if m.upgradeWizard != nil {
			cmd, back := m.upgradeWizard.Update(msg)
			if back {
				m.screen = ScreenMenu
				m.upgradeWizard = nil
			}
			return m, cmd
		}
	case ScreenAdd:
		if m.addWizard != nil {
			cmd, back := m.addWizard.Update(msg)
			if back {
				m.screen = ScreenMenu
				m.addWizard = nil
			}
			return m, cmd
		}
	}
	switch {
	case key.Matches(keyMsg, keys.Back):
		m.screen = ScreenMenu
		return m, nil
	case m.screen == ScreenMenu && key.Matches(keyMsg, keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case m.screen == ScreenMenu && key.Matches(keyMsg, keys.Down):
		if m.cursor < len(menuItems)-1 {
			m.cursor++
		}
		return m, nil
	case m.screen == ScreenMenu && key.Matches(keyMsg, keys.Enter):
		m.screen = menuItems[m.cursor].screen
		switch m.screen {
		case ScreenNew:
			m.newWizard = newNewWizard(m.opts, m.ctx, nil)
		case ScreenCheck:
			m.checkWizard = newCheckWizard(m.opts, m.ctx, nil)
		case ScreenUpgrade:
			m.upgradeWizard = newUpgradeWizard(m.opts, m.ctx, nil)
		case ScreenAdd:
			m.addWizard = newAddWizard(m.opts, m.ctx, nil)
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m rootModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Jimu CLI"))
	if m.opts.Version != "" {
		b.WriteString(" ")
		b.WriteString(mutedStyle.Render(m.opts.Version))
	}
	b.WriteByte('\n')
	if m.opts.ProjectRoot != "" {
		b.WriteString(mutedStyle.Render("project: " + m.opts.ProjectRoot))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	if m.screen != ScreenMenu {
		switch m.screen {
		case ScreenNew:
			if m.newWizard != nil {
				return m.newWizard.View()
			}
		case ScreenCheck:
			if m.checkWizard != nil {
				return m.checkWizard.View()
			}
		case ScreenUpgrade:
			if m.upgradeWizard != nil {
				return m.upgradeWizard.View()
			}
		case ScreenAdd:
			if m.addWizard != nil {
				return m.addWizard.View()
			}
		}
		b.WriteString(selectedStyle.Render(menuItems[m.screen-1].label))
		b.WriteByte('\n')
		b.WriteString(mutedStyle.Render("This screen is wired in the next task."))
		b.WriteByte('\n')
		b.WriteByte('\n')
		b.WriteString(helpStyle.Render("esc back · ctrl+c quit"))
		return b.String()
	}

	for index, item := range menuItems {
		line := "  " + item.label
		if index == m.cursor {
			line = "› " + item.label
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(helpStyle.Render("↑/↓ choose · enter select · q quit"))
	return b.String()
}
