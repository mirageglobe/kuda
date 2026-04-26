package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirageglobe/kuda/engine"
	"github.com/mirageglobe/kuda/mapper"
	"github.com/mirageglobe/kuda/network"
	"github.com/mirageglobe/kuda/ui"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

// rootModel owns model transitions and network client creation.
// It is the only place allowed to import both ui and network.
type rootModel struct {
	current       tea.Model
	conn          ui.Connection
	engine        *engine.Engine
	mapper        *mapper.Mapper
	width, height int
}

func (m rootModel) Init() tea.Cmd { return m.current.Init() }

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case ui.SplashDoneMsg:
		next := ui.NewLaunchModel()
		m.current = next
		return m, tea.Batch(next.Init(), func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		})

	case ui.ReturnToLauncherMsg:
		if m.conn != nil {
			_ = m.conn.Close()
			m.conn = nil
		}
		m.engine = nil
		next := ui.NewLaunchModel()
		m.current = next
		return m, tea.Batch(next.Init(), func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		})

	case ui.ServerSelectedMsg:
		if msg.Address == ui.MockAddress {
			conn := newMockConnection()
			m.conn = conn
			m.engine = engine.NewEngine(newEngineAdapter(conn))
			go watchRooms(m.engine, m.mapper)
			next := ui.NewClientModel(conn, m.engine, m.mapper)
			m.current = next
			return m, tea.Batch(next.Init(), func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			})
		}
		client := network.NewClient()
		return m, dialCmd(client, msg.Address)

	case dialResultMsg:
		if msg.err != nil {
			var cmd tea.Cmd
			m.current, cmd = m.current.Update(ui.ConnectErrorMsg{Err: msg.err})
			return m, cmd
		}
		m.conn = msg.adapter
		m.engine = engine.NewEngine(newEngineAdapter(msg.adapter))
		go watchRooms(m.engine, m.mapper)
		next := ui.NewClientModel(msg.adapter, m.engine, m.mapper)
		m.current = next
		return m, tea.Batch(next.Init(), func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		})
	}

	var cmd tea.Cmd
	m.current, cmd = m.current.Update(msg)
	return m, cmd
}

func (m rootModel) View() string { return m.current.View() }

// dialResultMsg carries the result of an async TCP connect attempt.
type dialResultMsg struct {
	adapter *clientAdapter
	err     error
}

// dialCmd dials asynchronously so the event loop stays responsive.
func dialCmd(client *network.Client, address string) tea.Cmd {
	return func() tea.Msg {
		if err := client.Connect(address); err != nil {
			return dialResultMsg{err: err}
		}
		return dialResultMsg{adapter: newClientAdapter(client)}
	}
}

// ── clientAdapter ────────────────────────────────────────────────────────────

// clientAdapter converts *network.Client to ui.Connection by mapping event types.
type clientAdapter struct {
	client *network.Client
	events chan ui.Event
	engine chan engine.Event
}

var _ ui.Connection = (*clientAdapter)(nil)

func newClientAdapter(client *network.Client) *clientAdapter {
	a := &clientAdapter{
		client: client,
		events: make(chan ui.Event, 1024),
		engine: make(chan engine.Event, 1024),
	}
	go a.forward()
	return a
}

func (a *clientAdapter) forward() {
	for ev := range a.client.EventsCh() {
		uEv := mapToUIEvent(ev)
		eEv := mapToEngineEvent(ev)
		a.events <- uEv
		a.engine <- eEv
	}
	close(a.events)
	close(a.engine)
}

func (a *clientAdapter) Write(data []byte) error   { return a.client.Write(data) }
func (a *clientAdapter) EventsCh() <-chan ui.Event { return a.events }
func (a *clientAdapter) ErrorsCh() <-chan error    { return a.client.ErrorsCh() }
func (a *clientAdapter) Close() error              { return a.client.Close() }
func (a *clientAdapter) ConnStatus() ui.ConnStatusInfo {
	return ui.ConnStatusInfo{
		MCCPActive: a.client.IsMCCPActive(),
		GMCPActive: a.client.IsGMCPActive(),
		EchoActive: a.client.IsEchoActive(),
	}
}

func mapToUIEvent(ev network.Event) ui.Event {
	var t ui.EventType
	switch ev.Type {
	case network.EventText:
		t = ui.EventText
	case network.EventGMCP:
		t = ui.EventGMCP
	case network.EventTelnetCommand:
		t = ui.EventTelnetCommand
	}
	return ui.Event{Type: t, Data: ev.Data}
}

func mapToEngineEvent(ev network.Event) engine.Event {
	var t engine.EventType
	switch ev.Type {
	case network.EventText:
		t = engine.EventText
	case network.EventGMCP:
		t = engine.EventGMCP
	case network.EventTelnetCommand:
		t = engine.EventTelnetCommand
	}
	return engine.Event{Type: t, Data: ev.Data}
}

// ── engineAdapter ────────────────────────────────────────────────────────────

type engineAdapter struct {
	conn ui.Connection
	ch   chan engine.Event
}

var _ engine.EventSource = (*engineAdapter)(nil)

func newEngineAdapter(conn ui.Connection) *engineAdapter {
	ea := &engineAdapter{
		conn: conn,
		ch:   make(chan engine.Event, 1024),
	}
	// We need to listen to the connection and map to engine events.
	// But clientAdapter already does this and provides a channel.
	// Let's optimize: if conn is clientAdapter, use its engine channel.
	if a, ok := conn.(*clientAdapter); ok {
		return &engineAdapter{conn: conn, ch: a.engine}
	}
	// Otherwise (like mock), we need to forward
	go func() {
		for ev := range conn.EventsCh() {
			ea.ch <- engine.Event{
				Type: engine.EventType(ev.Type),
				Data: ev.Data,
			}
		}
		close(ea.ch)
	}()
	return ea
}

func (e *engineAdapter) EventsCh() <-chan engine.Event { return e.ch }
func (e *engineAdapter) ErrorsCh() <-chan error        { return e.conn.ErrorsCh() }
func (e *engineAdapter) Write(data []byte) error       { return e.conn.Write(data) }

// ── mockConnection ───────────────────────────────────────────────────────────

// mockConnection is an in-process echo connection for local UI development.
// It requires no network; typed input is echoed back as server text.
type mockConnection struct {
	events chan ui.Event
	errors chan error
}

var _ ui.Connection = (*mockConnection)(nil)

func newMockConnection() *mockConnection {
	m := &mockConnection{
		events: make(chan ui.Event, 100),
		errors: make(chan error, 10),
	}
	m.events <- ui.Event{
		Type: ui.EventText,
		Data: []byte("[ kuda dev — local echo mode ]\r\nType anything and press Enter.\r\n\r\n"),
	}
	return m
}

func (m *mockConnection) Write(data []byte) error {
	text := strings.TrimRight(string(data), "\r\n")
	m.events <- ui.Event{Type: ui.EventText, Data: []byte("> " + text + "\r\n")}
	return nil
}

func (m *mockConnection) EventsCh() <-chan ui.Event     { return m.events }
func (m *mockConnection) ErrorsCh() <-chan error        { return m.errors }
func (m *mockConnection) Close() error                  { return nil }
func (m *mockConnection) ConnStatus() ui.ConnStatusInfo { return ui.ConnStatusInfo{} }

// watchRooms forwards room change events from an engine to the mapper.
// It exits when the engine's RoomCh is closed (on disconnect) and saves.
func watchRooms(eng *engine.Engine, mp *mapper.Mapper) {
	for room := range eng.RoomCh() {
		mp.Update(mapper.RoomData{
			Vnum:      room.Vnum,
			Name:      room.Name,
			Exits:     room.Exits,
			X:         room.X,
			Y:         room.Y,
			Z:         room.Z,
			HasCoords: room.HasCoords,
		})
	}
	_ = mp.Save()
}

// mapSavePath returns the platform config directory path for the map file.
func mapSavePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "kuda", "map.json")
}

// ── entry point ──────────────────────────────────────────────────────────────

func main() {
	model := rootModel{
		current: ui.NewSplashModel(),
		mapper:  mapper.Load(mapSavePath()),
	}
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("error running program: %v\n", err)
		os.Exit(1)
	}
}
