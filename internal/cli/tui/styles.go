package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle    lipgloss.Style
	selectedStyle lipgloss.Style
	mutedStyle    lipgloss.Style
	helpStyle     lipgloss.Style
	errorStyle    lipgloss.Style
	successStyle  lipgloss.Style
	passStyle     lipgloss.Style
	warnStyle     lipgloss.Style
	failStyle     lipgloss.Style
)

func init() {
	configureStyles(false)
}

func configureStyles(noColor bool) {
	if noColor {
		titleStyle = lipgloss.NewStyle()
		selectedStyle = lipgloss.NewStyle()
		mutedStyle = lipgloss.NewStyle()
		helpStyle = lipgloss.NewStyle()
		errorStyle = lipgloss.NewStyle()
		successStyle = lipgloss.NewStyle()
		passStyle = lipgloss.NewStyle()
		warnStyle = lipgloss.NewStyle()
		failStyle = lipgloss.NewStyle()
		return
	}
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	successStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	passStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
	warnStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	failStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
}
