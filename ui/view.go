package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var statusHint = hintStyle.Render("[ ?: help · esc: return to launch ]")

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
		"  esc        return to server list",
		"  ctrl+c     quit",
		"",
		"  commands",
		"  " + sep,
		"  /clear      clear scrollback buffer",
		"  /help       show / hide this help",
		"  /lua        toggle lua scripting",
		"  /map        toggle map panel",
		"  /map reset  reset map (backs up current map file)",
		"  /quit       disconnect and return to server list",
		"  /raw        toggle raw debug mode",
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

	toggleStr := func(label string, active bool) string {
		if active {
			return toggleActiveStyle.Render(label)
		}
		return toggleInactiveStyle.Render(label)
	}
	toggles := toggleStr("map", m.showMap) + " " +
		toggleStr("raw", m.rawMode) + " " +
		toggleStr("lua", m.engine.LuaActive()) + " "

	vitals := fmt.Sprintf("%s │ ♥ %d/%d │ ◆ %d/%d │ ↑ %d/%d │ %s%s │ %s",
		logoStyle.Render(name),
		v.HP, v.MaxHP,
		v.Mana, v.MaxMana,
		v.Move, v.MaxMove,
		subtitleStyle.Render(room.Name),
		hintStyle.Render(connInfo),
		toggles,
	)

	now := time.Now()
	pageLabel := fmt.Sprintf("session │ %s │ %s", now.Format("2006-01-02"), now.Format("15:04:05"))
	topBar := renderTopBar(m.width, pageLabel, m.serverName, m.stats.String())

	if m.showMap && m.mapView != nil {
		mapRendered := viewportBorderStyle.Render(m.mapView.Render(mapPanelWidth, m.viewportHeight()))
		pane = lipgloss.JoinHorizontal(lipgloss.Top, pane, mapRendered)
	}
	var left string
	switch {
	case m.pendingConfirm == "esc":
		left = renderKudaLabel("", "return to launcher? y to confirm, any other key to cancel")
	case m.pendingConfirm == "quit":
		left = renderKudaLabel("", "quit? y to confirm, any other key to cancel")
	case m.confirmResetMap:
		left = renderKudaLabel("", "reset map? all rooms will be lost — y to confirm, any other key to cancel")
	default:
		left = renderKudaLabel("", m.warnMsg)
	}
	pad := m.width - lipgloss.Width(left) - lipgloss.Width(vitals)
	if pad < 0 {
		pad = 0
	}
	statusLine := left + strings.Repeat(" ", pad) + vitals

	bottom := statusHint
	if suggestion := completionMatch(m.input.Value()); suggestion != "" {
		bottom = hintStyle.Render("  tab → " + suggestion)
	}
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s", topBar, pane, statusLine, m.input.View(), bottom)
}
