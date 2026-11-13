package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestRootModelQuitsOnQAndCtrlC(t *testing.T) {
	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
		{Type: tea.KeyCtrlC},
	} {
		updated, cmd := newRootModel(Options{}).Update(msg)
		model := updated.(rootModel)
		require.True(t, model.quit)
		require.NotNil(t, cmd)
	}
}

func TestRootModelMenuTransitionsAndBack(t *testing.T) {
	model := newRootModel(Options{})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(rootModel)
	require.Equal(t, 1, model.cursor)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(rootModel)
	require.Equal(t, ScreenCheck, model.screen)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(rootModel)
	require.Equal(t, ScreenMenu, model.screen)
}

func TestRootModelViewListsEveryAction(t *testing.T) {
	view := newRootModel(Options{Version: "test"}).View()
	for _, item := range menuItems {
		require.Contains(t, view, item.label)
	}
}

func TestRootModelOpensNewWizard(t *testing.T) {
	updated, _ := newRootModel(Options{}).Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(rootModel)
	require.Equal(t, ScreenNew, model.screen)
	require.NotNil(t, model.newWizard)
	require.Contains(t, model.View(), "New project")
	require.Contains(t, model.View(), "Directory:")
}
