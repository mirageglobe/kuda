package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SplashDoneMsg signals that the splash screen has been dismissed.
type SplashDoneMsg struct{}

const asciiLogo = ` _  ___   _ ___   _
| |/ / | | |   \ /_\
|   <| |_| | |) / _ \
|_|\_\\__,_|___/_/ \_\`

// SplashModel is the opening screen shown on launch.
type SplashModel struct {
	width  int
	height int
	stats  statsInfo
}

func NewSplashModel() SplashModel { return SplashModel{} }

func (m SplashModel) Init() tea.Cmd { return sysTickCmd() }

func (m SplashModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case sysTick:
		m.stats = m.stats.update()
		return m, sysTickCmd()
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, func() tea.Msg { return SplashDoneMsg{} }
	}
	return m, nil
}

func (m SplashModel) View() string {
	topBar := renderTopBar(m.width, "splash", "", m.stats.String())
	content := logoStyle.Render(asciiLogo) +
		"\n\n" + subtitleStyle.Render("a modern mud client") +
		"\n\n" + hintStyle.Render("press any key")

	var body string
	if m.width > 0 && m.height > 0 {
		body = lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, content)
	} else {
		body = content
	}
	return topBar + "\n" + body
}
