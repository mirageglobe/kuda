package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// AppVersion is set by main before creating any model, injected from build-time ldflags.
var AppVersion = "dev"

var (
	logoStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	subtitleStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	hintStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errorStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	toggleActiveStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	toggleInactiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	viewportBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))
)

// renderKudaLabel renders "  kuda : <text>" — optionally prefixed with "[warn] " in red.
func renderKudaLabel(text, warn string) string {
	glyph := logoStyle.Render("⧫")
	if warn != "" {
		return glyph + " " + errorStyle.Render("[warn] ") + subtitleStyle.Render(warn)
	}
	if text == "" {
		return glyph
	}
	return glyph + " " + subtitleStyle.Render(text)
}

// renderTopBar returns a full-width top bar: left shows kuda/version/mud, right shows page name.
func renderTopBar(width int, page, mud, stats string) string {
	left := " " + logoStyle.Render("kuda") + "  " + hintStyle.Render(AppVersion)
	if mud != "" {
		left += "  " + hintStyle.Render(mud)
	}
	right := hintStyle.Render(stats + "  " + page + " ")
	pad := width - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 0 {
		pad = 0
	}
	return left + strings.Repeat(" ", pad) + right
}
