package tui

import (
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

var blinkCursorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("15")).
	Background(lipgloss.Color("4"))

func configureTextInput(input *textinput.Model, noColor bool) {
	input.Cursor.Style = selectedStyle
	if noColor {
		input.Cursor.SetMode(cursor.CursorStatic)
		return
	}
	input.Cursor.SetMode(cursor.CursorBlink)
}

func prepareTextInput(input *textinput.Model, focused bool) {
	if focused {
		input.Cursor.TextStyle = blinkCursorStyle
		return
	}
	input.Cursor.TextStyle = lipgloss.NewStyle()
}
