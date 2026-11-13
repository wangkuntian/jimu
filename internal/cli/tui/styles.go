package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errorStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	successStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
)
