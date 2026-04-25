package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirageglobe/kuda/network"
)

// --- Launch State ---

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type LaunchModel struct {
	list list.Model
}

func NewLaunchModel() LaunchModel {
	items := []list.Item{
		item{title: "Aardwolf", desc: "aardwolf.com:4000"},
		item{title: "TorilMUD", desc: "torilmud.com:9999"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select a MUD"
	return LaunchModel{list: l}
}

func (m LaunchModel) Init() tea.Cmd { return nil }

func (m LaunchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			i := m.list.SelectedItem().(item)
			client := network.NewClient()
			err := client.Connect(i.desc)
			if err != nil {
				return m, nil
			}
			newModel := NewModel(client)
			// Trigger a resize to initialize the viewport/input dimensions
			return newModel, func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.list.Width(), Height: m.list.Height()}
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m LaunchModel) View() string { return m.list.View() }

// --- Connected State ---

// NetworkEventMsg wraps a network event for Bubbletea.
type NetworkEventMsg struct {
	Event network.Event
}

// ErrorMsg wraps an error for Bubbletea.
type ErrorMsg struct {
	Err error
}

// Model represents the TUI state.
type Model struct {
	client   *network.Client
	viewport viewport.Model
	input    textinput.Model
	history  strings.Builder
	width    int
	height   int
}

// NewModel creates a new TUI model.
func NewModel(client *network.Client) Model {
	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.Focus()

	vp := viewport.New(0, 0)
	vp.SetContent("Connected...")

	return Model{
		client:   client,
		input:    ti,
		viewport: vp,
	}
}

// Init initializes the TUI.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.waitForNetworkEvent(),
	)
}

// waitForNetworkEvent waits for the next event from the network client.
func (m Model) waitForNetworkEvent() tea.Cmd {
	return func() tea.Msg {
		select {
		case ev, ok := <-m.client.Events:
			if !ok {
				return ErrorMsg{Err: fmt.Errorf("network connection closed")}
			}
			return NetworkEventMsg{Event: ev}
		case err := <-m.client.Errors:
			return ErrorMsg{Err: err}
		}
	}
}

// Update handles UI updates.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		cmds  []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			cmd := m.input.Value()
			if cmd != "" {
				// Send to server with newline
				m.client.Write([]byte(cmd + "\n"))
				m.input.SetValue("")
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 3 // Leave space for input
		m.input.Width = msg.Width

	case NetworkEventMsg:
		if msg.Event.Type == network.EventText {
			// Normalize \r\n to \n for the viewport
			text := string(msg.Event.Data)
			text = strings.ReplaceAll(text, "\r\n", "\n")
			text = strings.ReplaceAll(text, "\r", "\n")
			m.history.WriteString(text)
			
			m.viewport.SetContent(m.history.String())
			m.viewport.GotoBottom()
		}
		cmds = append(cmds, m.waitForNetworkEvent())

	case ErrorMsg:
		m.history.WriteString(fmt.Sprintf("\n[ ERROR: %v ]\n", msg.Err))
		m.viewport.SetContent(m.history.String())
		m.viewport.GotoBottom()
		return m, tea.Quit
	}

	m.input, tiCmd = m.input.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	cmds = append(cmds, tiCmd, vpCmd)
	return m, tea.Batch(cmds...)
}

// View renders the TUI.
func (m Model) View() string {
	content := m.viewport.View()
	if content == "" {
		content = "Waiting for data..."
	}
	return fmt.Sprintf(
		"%s\n\n%s",
		content,
		m.input.View(),
	)
}
