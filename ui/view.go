package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var statusHint = hintStyle.Render("[ ?: help · esc: back · ^l: lua · ^p: map · ^r: raw · ^x: reset map · ^c: quit ]")

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
		"  ctrl+x     reset map (backs up current map file)",
		"  esc        return to server list",
		"  ctrl+c     quit",
		"",
		"  client commands",
		"  " + sep,
		"  /clear     clear scrollback buffer",
		"  /help      show / hide this help",
		"  /map       toggle map panel",
		"  /quit      disconnect and return to server list",
		"  /raw       toggle raw debug mode",
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

	v := m.engine.Vitals()
	room := m.engine.Room()
	name := m.engine.CharName()

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
	hint := statusHint
	if m.confirmResetMap {
		hint = hintStyle.Render("[ reset map? all rooms will be lost — y to confirm, any other key to cancel ]")
	}
	suggestion := completionMatch(m.input.Value())
	suggestionLine := hintStyle.Render("  tab → " + suggestion)
	if suggestion == "" {
		suggestionLine = ""
	}
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s", topBar, pane, statusBar, m.input.View(), suggestionLine, hint)
}
