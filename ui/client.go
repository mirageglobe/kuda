package ui

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mirageglobe/kuda/engine"
	"github.com/muesli/reflow/wordwrap"
)

// connStatusStr returns a compact status string for active telnet protocol features.
func connStatusStr(cs ConnStatusInfo) string {
	var parts []string
	if cs.GMCPActive {
		parts = append(parts, "GMCP")
	}
	if cs.MCCPActive {
		parts = append(parts, "MCCP")
	}
	if cs.EchoActive {
		parts = append(parts, "ECHO")
	}
	if len(parts) == 0 {
		return ""
	}
	return " │ " + strings.Join(parts, " ")
}

// formatRawEvent renders a network event as a tagged escaped-byte string for raw mode.
func formatRawEvent(ev Event) string {
	var tag string
	switch ev.Type {
	case EventText:
		tag = "[TEXT  ]"
	case EventGMCP:
		tag = "[GMCP  ]"
	case EventTelnetCommand:
		tag = "[TELNET]"
	default:
		tag = "[OTHER ]"
	}
	var sb strings.Builder
	sb.WriteString(tag)
	sb.WriteByte(' ')
	for _, b := range ev.Data {
		if b >= 0x20 && b < 0x7f {
			sb.WriteByte(b)
		} else {
			fmt.Fprintf(&sb, "\\x%02x", b)
		}
	}
	return sb.String()
}

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

var statusHint = hintStyle.Render("[ ?: help · esc: back · ^l: lua · ^p: map · ^r: raw · ^c: quit ]")

// sysTick is the message fired by the 1-second system info ticker.
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
	history    *strings.Builder // raw history with ANSI codes
	cmdHistory []string         // previously entered commands
	historyIdx int              // current position in cmdHistory; -1 = not navigating
	inputDraft string           // saved input before history navigation began
	width      int
	height     int
	memMB      uint64 // heap alloc in MB, updated each sysTick
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
		showMap:    false,
		input:      ti,
		viewport:   vp,
		history:    &strings.Builder{},
		historyIdx: -1,
	}
}

func (m *ClientModel) viewportHeight() int {
	h := m.height - 6 // topbar(1) + border top+bottom(2) + status(1) + input(1) + hint(1)
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
			if cmd != "" {
				m.cmdHistory = append(m.cmdHistory, cmd)
			}
			m.historyIdx = -1
			m.inputDraft = ""
			// Signal clean disconnect before the server closes on "quit".
			if strings.EqualFold(strings.TrimSpace(cmd), "quit") {
				_ = m.client.Close()
			}
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
		// Telnet echo handling applies regardless of raw mode.
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
			// Most MUDs send \r\n for newlines. Some send \r\x00 for prompts.
			// By stripping \r and \x00 entirely, we preserve only the \n
			// which prevents the "double spacing" gap issue caused by
			// treating \r as a separate newline.
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

func (m ClientModel) helpView() string {
	w := m.viewportInnerWidth()
	sep := strings.Repeat("─", w-2)
	lines := []string{
		"",
		"  keybindings",
		"  " + sep,
		"  ?          show / hide this help",
		"  up / down  command history",
		"  ctrl+l     toggle lua scripting",
		"  ctrl+p     toggle map panel",
		"  ctrl+r     toggle raw mode (debug)",
		"  esc        return to server list",
		"  ctrl+c     quit",
	}
	h := m.viewportHeight()
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines[:h], "\n")
}

func (m ClientModel) View() string {
	var pane string
	if m.showHelp {
		pane = viewportBorderStyle.Render(m.helpView())
	} else {
		pane = viewportBorderStyle.Render(m.viewport.View())
	}

	// Render Status Bar
	v := m.engine.Vitals()
	room := m.engine.Room()
	name := m.engine.CharName()
	if name == "" {
		name = "Connecting..."
	}

	cs := m.client.ConnStatus()
	connInfo := connStatusStr(cs)
	statusBar := fmt.Sprintf(" %s │ ♥ %d/%d │ ◆ %d/%d │ ↑ %d/%d │ %s%s",
		logoStyle.Render(name),
		v.HP, v.MaxHP,
		v.Mana, v.MaxMana,
		v.Move, v.MaxMove,
		subtitleStyle.Render(room.Name),
		hintStyle.Render(connInfo),
	)

	now := time.Now()
	topLeft := fmt.Sprintf(" %s %s  %s",
		logoStyle.Render("kuda"),
		hintStyle.Render("v"+AppVersion),
		hintStyle.Render("github.com/mirageglobe/kuda"),
	)
	topRight := hintStyle.Render(fmt.Sprintf("%s │ %s │ mem %d MB ",
		now.Format("2006-01-02"),
		now.Format("15:04:05"),
		m.memMB,
	))
	topPad := m.width - lipgloss.Width(topLeft) - lipgloss.Width(topRight)
	if topPad < 0 {
		topPad = 0
	}
	topBar := topLeft + strings.Repeat(" ", topPad) + topRight

	if m.showMap && m.mapView != nil {
		mapRendered := viewportBorderStyle.Render(m.mapView.Render(mapPanelWidth, m.viewportHeight()))
		pane = lipgloss.JoinHorizontal(lipgloss.Top, pane, mapRendered)
	}
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s", topBar, pane, statusBar, m.input.View(), statusHint)
}
