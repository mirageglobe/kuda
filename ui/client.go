package ui

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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

const mapPanelWidth = 35

type sysTick struct{}

func sysTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return sysTick{} })
}

// ClientModel is the main connected-session view.
type ClientModel struct {
	client     Connection
	engine     engine.GameState
	mapView    MapView
	showMap    bool
	showHelp   bool
	rawMode    bool
	viewport   viewport.Model
	input      textinput.Model
	history    *strings.Builder
	cmdHistory []string
	historyIdx int
	inputDraft string
	width      int
	height     int
	memMB      uint64
}

func NewClientModel(client Connection, state engine.GameState, mapView MapView) ClientModel {
	ti := textinput.New()
	ti.Prompt = "kuda > "
	ti.Placeholder = "type a command..."
	ti.Focus()

	vp := viewport.New(0, 0)
	vp.SetContent("Connected...")

	return ClientModel{
		client:     client,
		engine:     state,
		mapView:    mapView,
		input:      ti,
		viewport:   vp,
		history:    &strings.Builder{},
		historyIdx: -1,
	}
}

func (m *ClientModel) viewportHeight() int {
	h := m.height - 6
	if h < 1 {
		h = 1
	}
	return h
}

func (m *ClientModel) viewportInnerWidth() int {
	w := m.width - 2
	if m.showMap {
		w -= mapPanelWidth + 2
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
	wrapped := wordwrap.String(m.history.String(), m.viewportInnerWidth())
	m.viewport.SetContent(wrapped)
	m.viewport.GotoBottom()
}

func (m ClientModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.waitForNetworkEvent(),
		sysTickCmd(),
	)
}

func (m ClientModel) waitForNetworkEvent() tea.Cmd {
	return func() tea.Msg {
		select {
		case ev, ok := <-m.client.EventsCh():
			if !ok {
				return ReturnToLauncherMsg{}
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
		case tea.KeyCtrlR:
			m.rawMode = !m.rawMode
			return m, nil
		case tea.KeyCtrlX:
			if m.mapView != nil {
				backup, err := m.mapView.Reset()
				if err != nil {
					fmt.Fprintf(m.history, "\n[ MAP RESET FAILED: %v ]\n", err)
				} else if backup != "" {
					fmt.Fprintf(m.history, "\n[ MAP RESET — backup: %s ]\n", backup)
				} else {
					fmt.Fprintf(m.history, "\n[ MAP RESET ]\n")
				}
				m.refreshViewport()
			}
			return m, nil
		case tea.KeyRunes:
			if msg.String() == "?" && m.input.Value() == "" {
				m.showHelp = !m.showHelp
				return m, nil
			}
		case tea.KeyUp:
			if len(m.cmdHistory) == 0 {
				return m, nil
			}
			if m.historyIdx == -1 {
				m.inputDraft = m.input.Value()
				m.historyIdx = len(m.cmdHistory) - 1
			} else if m.historyIdx > 0 {
				m.historyIdx--
			}
			m.input.SetValue(m.cmdHistory[m.historyIdx])
			return m, nil
		case tea.KeyDown:
			if m.historyIdx == -1 {
				return m, nil
			}
			m.historyIdx++
			if m.historyIdx >= len(m.cmdHistory) {
				m.historyIdx = -1
				m.input.SetValue(m.inputDraft)
			} else {
				m.input.SetValue(m.cmdHistory[m.historyIdx])
			}
			return m, nil
		case tea.KeyEsc:
			return m, func() tea.Msg { return ReturnToLauncherMsg{} }
		case tea.KeyEnter:
			cmd := m.input.Value()
			if cmd != "" && m.input.EchoMode != textinput.EchoPassword {
				m.cmdHistory = append(m.cmdHistory, cmd)
			}
			m.historyIdx = -1
			m.inputDraft = ""
			if strings.EqualFold(strings.TrimSpace(cmd), "quit") {
				_ = m.client.Close()
			}
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
		if msg.Event.Type == EventTelnetCommand && len(msg.Event.Data) >= 2 {
			cmd, opt := msg.Event.Data[0], msg.Event.Data[1]
			if opt == TelnetOptEcho {
				if cmd == TelnetCmdWILL {
					m.input.EchoMode = textinput.EchoPassword
				} else if cmd == TelnetCmdWONT {
					m.input.EchoMode = textinput.EchoNormal
				}
			}
		}
		if m.rawMode {
			fmt.Fprintf(m.history, "%s\n", formatRawEvent(msg.Event))
		} else if msg.Event.Type == EventText {
			text := strings.ReplaceAll(string(msg.Event.Data), "\r", "")
			text = strings.ReplaceAll(text, "\x00", "")
			m.history.WriteString(text)
		}
		m.refreshViewport()
		cmds = append(cmds, m.waitForNetworkEvent())

	case sysTick:
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		m.memMB = ms.Alloc / 1024 / 1024
		return m, sysTickCmd()

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
