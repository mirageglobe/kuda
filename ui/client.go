package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mirageglobe/kuda/engine"
	"github.com/muesli/reflow/wordwrap"
)

// ReturnToLauncherMsg signals that the user wants to return to the server list.
type ReturnToLauncherMsg struct{}

// NetworkEventMsg wraps a network event for bubbletea.
type NetworkEventMsg struct {
	Event Event
}

// ErrorMsg wraps a network or runtime error for bubbletea.
type ErrorMsg struct {
	Err error
}

const mapPanelWidth = 35 // inner width of the right-side map panel

var statusHint = hintStyle.Render("[ esc: server list  ctrl+l: toggle lua  ctrl+p: toggle map  ctrl+c: quit ]")

// ClientModel is the main connected-session view.
type ClientModel struct {
	client   Connection
	engine   engine.GameState
	mapView  MapView
	showMap  bool
	viewport viewport.Model
	input    textinput.Model
	history  *strings.Builder // raw history with ANSI codes
	width    int
	height   int
}

func NewClientModel(client Connection, state engine.GameState, mapView MapView) ClientModel {
	ti := textinput.New()
	ti.Placeholder = "type a command..."
	ti.Focus()

	vp := viewport.New(0, 0)
	vp.SetContent("Connected...")

	return ClientModel{
		client:   client,
		engine:   state,
		mapView:  mapView,
		showMap:  false,
		input:    ti,
		viewport: vp,
		history:  &strings.Builder{},
	}
}

func (m *ClientModel) viewportHeight() int {
	h := m.height - 7 // 5 for status/input/hint + 2 for border top/bottom
	if h < 1 {
		h = 1
	}
	return h
}

func (m *ClientModel) viewportInnerWidth() int {
	w := m.width - 2 // subtract viewport border
	if m.showMap {
		w -= mapPanelWidth + 2 // subtract map panel content + its border
	}
	if w < 1 {
		w = 1
	}
	return w
}

func (m *ClientModel) refreshViewport() {
	if m.width <= 0 {
		return
	}
	// wordwrap.String is ANSI-aware. wrapping ensures colors don't bleed
	// and text doesn't overflow horizontally.
	wrapped := wordwrap.String(m.history.String(), m.viewportInnerWidth())
	m.viewport.SetContent(wrapped)
	m.viewport.GotoBottom()
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
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyCtrlL:
			m.engine.ToggleLua()
			return m, nil
		case tea.KeyCtrlP:
			m.showMap = !m.showMap
			m.viewport.Width = m.viewportInnerWidth()
			m.viewport.Height = m.viewportHeight()
			m.refreshViewport()
			return m, nil
		case tea.KeyEsc:
			return m, func() tea.Msg { return ReturnToLauncherMsg{} }
		case tea.KeyEnter:
			cmd := m.input.Value()
			// Process command through the engine (aliases/scripting)
			if err := m.engine.Execute(cmd); err != nil {
				fmt.Fprintf(m.history, "\n[ ERROR: %v ]\n", err)
				m.refreshViewport()
			}
			m.input.SetValue("")
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = m.viewportInnerWidth()
		m.viewport.Height = m.viewportHeight()
		m.input.Width = msg.Width
		m.refreshViewport()

	case NetworkEventMsg:
		switch msg.Event.Type {
		case EventText:
			text := string(msg.Event.Data)
			// Most MUDs send \r\n for newlines. Some send \r\x00 for prompts.
			// By stripping \r and \x00 entirely, we preserve only the \n
			// which prevents the "double spacing" gap issue caused by
			// treating \r as a separate newline.
			text = strings.ReplaceAll(text, "\r", "")
			text = strings.ReplaceAll(text, "\x00", "")
			m.history.WriteString(text)
			m.refreshViewport()
		case EventTelnetCommand:
			if len(msg.Event.Data) >= 2 {
				cmd := msg.Event.Data[0]
				opt := msg.Event.Data[1]
				if opt == TelnetOptEcho {
					if cmd == TelnetCmdWILL {
						m.input.EchoMode = textinput.EchoPassword
					} else if cmd == TelnetCmdWONT {
						m.input.EchoMode = textinput.EchoNormal
					}
				}
			}
		}
		cmds = append(cmds, m.waitForNetworkEvent())

	case ErrorMsg:
		fmt.Fprintf(m.history, "\n[ ERROR: %v ]\n", msg.Err)
		m.refreshViewport()
		return m, tea.Quit
	}

	m.input, tiCmd = m.input.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, tiCmd, vpCmd)
	return m, tea.Batch(cmds...)
}

func (m ClientModel) View() string {
	content := viewportBorderStyle.Render(m.viewport.View())

	// Render Status Bar
	v := m.engine.Vitals()
	room := m.engine.Room()
	name := m.engine.CharName()
	if name == "" {
		name = "Connecting..."
	}

	statusBar := fmt.Sprintf(" %s | HP %d/%d | MN %d/%d | MV %d/%d | %s",
		logoStyle.Render(name),
		v.HP, v.MaxHP,
		v.Mana, v.MaxMana,
		v.Move, v.MaxMove,
		subtitleStyle.Render(room.Name),
	)

	if m.showMap && m.mapView != nil {
		mapRendered := viewportBorderStyle.Render(m.mapView.Render(mapPanelWidth, m.viewportHeight()))
		content = lipgloss.JoinHorizontal(lipgloss.Top, content, mapRendered)
	}
	return fmt.Sprintf("%s\n%s\n%s\n%s", content, statusBar, m.input.View(), statusHint)
}
