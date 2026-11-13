package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenMenu screen = iota
	screenNew
	screenCheck
	screenUpgrade
	screenAdd
)

type menuItem struct {
	label  string
	screen screen
}

var menuItems = []menuItem{
	{label: "New project", screen: screenNew},
	{label: "Check project", screen: screenCheck},
	{label: "Upgrade project", screen: screenUpgrade},
	{label: "Add capability", screen: screenAdd},
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
	opts   Options
	cursor int
	screen screen
	quit   bool
}

func newRootModel(opts Options) rootModel {
	return rootModel{opts: opts}
}

func (m rootModel) Init() tea.Cmd { return nil }

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(keyMsg, keys.Quit):
		m.quit = true
		return m, tea.Quit
	case key.Matches(keyMsg, keys.Back):
		m.screen = screenMenu
		return m, nil
	case m.screen == screenMenu && key.Matches(keyMsg, keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case m.screen == screenMenu && key.Matches(keyMsg, keys.Down):
		if m.cursor < len(menuItems)-1 {
			m.cursor++
		}
		return m, nil
	case m.screen == screenMenu && key.Matches(keyMsg, keys.Enter):
		m.screen = menuItems[m.cursor].screen
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

	if m.screen != screenMenu {
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
