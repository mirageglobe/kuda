package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ServerSelectedMsg is emitted when the user confirms a server choice.
// main.go handles this by creating the network client and transitioning to ClientModel.
type ServerSelectedMsg struct {
	Name    string
	Address string
}

// ConnectErrorMsg is sent back to LaunchModel when the connection attempt fails.
type ConnectErrorMsg struct {
	Err error
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type LaunchModel struct {
	list       list.Model
	input      textinput.Model
	errMsg     string
	connecting bool
	width      int
}

// MockAddress is the sentinel address that triggers the in-process echo connection.
const MockAddress = "mock://echo"

func NewLaunchModel() LaunchModel {
	servers := GetServers()
	items := make([]list.Item, len(servers))
	for i, s := range servers {
		items[i] = item{title: s.Name, desc: s.Address}
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.SetShowTitle(false)
	l.SetFilteringEnabled(false)

	ti := textinput.New()
	ti.Prompt = "❯ "
	ti.Placeholder = "/filter  /quit"
	ti.Focus()

	return LaunchModel{list: l, input: ti}
}

func (m LaunchModel) Init() tea.Cmd { return textinput.Blink }

func (m LaunchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.connecting {
			return m, nil
		}
		switch msg.String() {
		case "up", "down":
			var listCmd tea.Cmd
			m.list, listCmd = m.list.Update(msg)
			return m, listCmd
		case "tab":
			if match := launchCompletionMatch(m.input.Value()); match != "" {
				m.input.SetValue(match)
				m.input.CursorEnd()
			}
			return m, nil
		case "enter":
			cmd := strings.TrimSpace(m.input.Value())
			if strings.HasPrefix(cmd, "/") {
				m.input.SetValue("")
				m.errMsg = ""
				parts := strings.Fields(cmd)
				switch parts[0] {
				case "/quit":
					return m, tea.Quit
				case "/filter":
					term := ""
					if len(parts) > 1 {
						term = strings.ToLower(strings.Join(parts[1:], " "))
					}
					all := GetServers()
					filtered := make([]list.Item, 0, len(all))
					for _, s := range all {
						if term == "" || strings.Contains(strings.ToLower(s.Name), term) || strings.Contains(strings.ToLower(s.Address), term) {
							filtered = append(filtered, item{title: s.Name, desc: s.Address})
						}
					}
					m.list.SetItems(filtered)
				default:
					m.errMsg = fmt.Sprintf("unknown command: %s", parts[0])
				}
				return m, nil
			}
			i := m.list.SelectedItem().(item)
			m.connecting = true
			return m, func() tea.Msg { return ServerSelectedMsg{Name: i.title, Address: i.desc} }
		case "ctrl+c":
			return m, tea.Quit
		}
	case ConnectErrorMsg:
		m.connecting = false
		m.errMsg = fmt.Sprintf("error: %v", msg.Err)
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.input.Width = msg.Width
		// reserve: topbar + border(2) + label + input + hint = 6
		m.list.SetSize(msg.Width-2, msg.Height-6)
		return m, nil
	}
	var tiCmd tea.Cmd
	m.input, tiCmd = m.input.Update(msg)
	return m, tiCmd
}

var launchHint = hintStyle.Render("[ ↑↓: navigate · enter: connect · /quit · ^c: quit ]")

func (m LaunchModel) View() string {
	m.list.SetShowHelp(false)
	topBar := renderTopBar(m.width, "launch", "")

	var labelText, labelWarn string
	switch {
	case m.connecting:
		labelText = "connecting..."
	case m.errMsg != "":
		labelWarn = m.errMsg
	default:
		labelText = "select a mud"
	}
	label := renderKudaLabel(labelText, labelWarn)

	bottom := launchHint
	if suggestion := launchCompletionMatch(m.input.Value()); suggestion != "" {
		bottom = hintStyle.Render("  tab → " + suggestion)
	}

	body := viewportBorderStyle.Width(m.width - 2).Render(m.list.View())
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s", topBar, body, label, m.input.View(), bottom)
}
