package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// NetworkEventMsg wraps a network event for bubbletea.
type NetworkEventMsg struct {
	Event Event
}

// ErrorMsg wraps a network or runtime error for bubbletea.
type ErrorMsg struct {
	Err error
}

// ClientModel is the main connected-session view.
type ClientModel struct {
	client   Connection
	viewport viewport.Model
	input    textinput.Model
	history  *strings.Builder // pointer: strings.Builder must not be copied after first write
	width    int
	height   int
}

func NewClientModel(client Connection) ClientModel {
	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.Focus()

	vp := viewport.New(0, 0)
	vp.SetContent("Connected...")

	return ClientModel{
		client:   client,
		input:    ti,
		viewport: vp,
		history:  &strings.Builder{},
	}
}

func (m ClientModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.waitForNetworkEvent(),
	)
}

func (m ClientModel) waitForNetworkEvent() tea.Cmd {
	return func() tea.Msg {
		select {
		case ev, ok := <-m.client.EventsCh():
			if !ok {
				return ErrorMsg{Err: fmt.Errorf("network connection closed")}
			}
			return NetworkEventMsg{Event: ev}
		case err := <-m.client.ErrorsCh():
			return ErrorMsg{Err: err}
		}
	}
}

func (m ClientModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				if err := m.client.Write([]byte(cmd + "\n")); err != nil {
					fmt.Fprintf(m.history, "\n[ ERROR: %v ]\n", err)
					m.viewport.SetContent(m.history.String())
					m.viewport.GotoBottom()
				}
				m.input.SetValue("")
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 3
		m.input.Width = msg.Width

	case NetworkEventMsg:
		if msg.Event.Type == EventText {
			text := strings.ReplaceAll(string(msg.Event.Data), "\r\n", "\n")
			text = strings.ReplaceAll(text, "\r", "\n")
			m.history.WriteString(text)
			m.viewport.SetContent(m.history.String())
			m.viewport.GotoBottom()
		}
		cmds = append(cmds, m.waitForNetworkEvent())

	case ErrorMsg:
		fmt.Fprintf(m.history, "\n[ ERROR: %v ]\n", msg.Err)
		m.viewport.SetContent(m.history.String())
		m.viewport.GotoBottom()
		return m, tea.Quit
	}

	m.input, tiCmd = m.input.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, tiCmd, vpCmd)
	return m, tea.Batch(cmds...)
}

func (m ClientModel) View() string {
	content := m.viewport.View()
	if content == "" {
		content = "Waiting for data..."
	}
	return fmt.Sprintf("%s\n\n%s", content, m.input.View())
}
