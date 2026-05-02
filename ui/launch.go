package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
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
	errMsg     string
	connecting bool
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
	l.Title = "Select a MUD"
	return LaunchModel{list: l}
}

func (m LaunchModel) Init() tea.Cmd { return nil }

func (m LaunchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.connecting {
			return m, nil
		}
		switch msg.String() {
		case "enter":
			i := m.list.SelectedItem().(item)
			m.connecting = true
			return m, func() tea.Msg { return ServerSelectedMsg{Name: i.title, Address: i.desc} }
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case ConnectErrorMsg:
		m.connecting = false
		m.errMsg = fmt.Sprintf("[ ERROR: %v ]", msg.Err)
		return m, nil
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-3)
		return m, nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m LaunchModel) View() string {
	m.list.SetShowHelp(false)
	lView := m.list.View()

	var status string
	if m.connecting {
		status = hintStyle.Render("Connecting...")
	} else if m.errMsg != "" {
		status = m.errMsg
	}

	// Always reserve 2 lines for status (one for content, one for spacing)
	// and 1 line for help. Total 3 lines reserved in SetSize.
	return fmt.Sprintf("%s\n\n%s\n%s", lView, status, m.list.Help.View(m.list))
}
