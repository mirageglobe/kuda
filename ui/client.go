package ui

import (
	"fmt"
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

var clientCmds = []string{"/clear", "/help", "/lua", "/map", "/map reset", "/quit", "/raw"}
var launchCmds = []string{"/filter", "/quit"}

func matchCmd(cmds []string, input string) string {
	if !strings.HasPrefix(input, "/") || input == "/" {
		return ""
	}
	for _, c := range cmds {
		if strings.HasPrefix(c, input) && c != input {
			return c
		}
	}
	return ""
}

func completionMatch(input string) string       { return matchCmd(clientCmds, input) }
func launchCompletionMatch(input string) string { return matchCmd(launchCmds, input) }

type sysTick struct{}

func sysTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return sysTick{} })
}

// ClientModel is the main connected-session view.
type ClientModel struct {
	client          Connection
	engine          engine.GameState
	mapView         MapView
	serverName      string
	showMap         bool
	showHelp        bool
	rawMode         bool
	confirmResetMap bool
	pendingConfirm  string // "esc", "quit", or ""
	viewport        viewport.Model
	input           textinput.Model
	history         *strings.Builder
	cmdHistory      []string
	historyIdx      int
	inputDraft      string
	warnMsg         string
	width           int
	height          int
	stats           statsInfo
}

func NewClientModel(client Connection, state engine.GameState, mapView MapView, serverName string) ClientModel {
	ti := textinput.New()
	ti.Prompt = "❯ "
	ti.Placeholder = "type a command..."
	ti.Focus()

	vp := viewport.New(0, 0)
	vp.SetContent("Connected...")

	return ClientModel{
		client:     client,
		engine:     state,
		mapView:    mapView,
		serverName: serverName,
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
		if m.showHelp && msg.Type != tea.KeyCtrlC {
			m.showHelp = false
			return m, nil
		}
		if m.pendingConfirm != "" {
			if msg.String() == "y" || msg.String() == "Y" {
				action := m.pendingConfirm
				m.pendingConfirm = ""
				switch action {
				case "esc":
					return m, func() tea.Msg { return ReturnToLauncherMsg{} }
				case "quit":
					_ = m.client.Close()
					return m, tea.Quit
				}
			}
			m.pendingConfirm = ""
			return m, nil
		}
		if m.confirmResetMap {
			if msg.String() == "y" || msg.String() == "Y" {
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
			m.confirmResetMap = false
			return m, nil
		}
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyTab:
			if match := completionMatch(m.input.Value()); match != "" {
				m.input.SetValue(match)
				m.input.CursorEnd()
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
			m.pendingConfirm = "esc"
			return m, nil
		case tea.KeyEnter:
			cmd := m.input.Value()
			if cmd != "" && m.input.EchoMode != textinput.EchoPassword {
				m.cmdHistory = append(m.cmdHistory, cmd)
			}
			m.historyIdx = -1
			m.inputDraft = ""
			m.warnMsg = ""
			if strings.HasPrefix(cmd, "/") {
				m.handleClientCmd(strings.TrimSpace(cmd))
			} else {
				if err := m.engine.Execute(cmd); err != nil {
					fmt.Fprintf(m.history, "\n[ ERROR: %v ]\n", err)
					m.refreshViewport()
				}
			}
			m.input.SetValue("")
			return m, nil
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
		m.stats = m.stats.update()
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

func (m *ClientModel) handleClientCmd(cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}
	switch parts[0] {
	case "/help":
		m.showHelp = !m.showHelp
		m.refreshViewport()
	case "/lua":
		m.engine.ToggleLua()
	case "/map":
		m.showMap = !m.showMap
		m.viewport.Width = m.viewportInnerWidth()
		m.viewport.Height = m.viewportHeight()
		m.refreshViewport()
	case "/map reset":
		if m.mapView != nil {
			m.confirmResetMap = true
		}
	case "/raw":
		m.rawMode = !m.rawMode
	case "/quit":
		m.pendingConfirm = "quit"
	case "/clear":
		m.history.Reset()
		m.refreshViewport()
	default:
		m.warnMsg = "unknown command " + parts[0]
	}
}
