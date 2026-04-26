package ui

import "github.com/charmbracelet/lipgloss"

// AppVersion is set by main before creating any model, injected from build-time ldflags.
var AppVersion = "dev"

var (
	logoStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	subtitleStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	hintStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	viewportBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))
)
