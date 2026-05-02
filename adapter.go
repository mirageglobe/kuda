package main

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirageglobe/kuda/engine"
	"github.com/mirageglobe/kuda/mapper"
	"github.com/mirageglobe/kuda/network"
	"github.com/mirageglobe/kuda/ui"
)

type dialResultMsg struct {
	adapter *clientAdapter
	err     error
}

func dialCmd(client *network.Client, address string) tea.Cmd {
	return func() tea.Msg {
		if err := client.Connect(address); err != nil {
			return dialResultMsg{err: err}
		}
		return dialResultMsg{adapter: newClientAdapter(client)}
	}
}

// ── clientAdapter ─────────────────────────────────────────────────────────────

type clientAdapter struct {
	client *network.Client
	events chan ui.Event
	engine chan engine.Event
}

var _ ui.Connection = (*clientAdapter)(nil)
var _ ui.MapView = (*mapper.Mapper)(nil)

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
		a.events <- mapToUIEvent(ev)
		a.engine <- mapToEngineEvent(ev)
	}
	close(a.events)
	close(a.engine)
}

// engineEventsCh exposes the pre-mapped engine event channel,
// allowing newEngineAdapter to reuse it instead of double-forwarding.
func (a *clientAdapter) engineEventsCh() <-chan engine.Event { return a.engine }

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

// ── engineAdapter ─────────────────────────────────────────────────────────────

// engineChannelProvider is satisfied by adapters that pre-map engine events,
// avoiding a second forwarding goroutine.
type engineChannelProvider interface {
	engineEventsCh() <-chan engine.Event
}

type engineAdapter struct {
	conn ui.Connection
	ch   <-chan engine.Event
}

var _ engine.EventSource = (*engineAdapter)(nil)

func newEngineAdapter(conn ui.Connection) *engineAdapter {
	if p, ok := conn.(engineChannelProvider); ok {
		return &engineAdapter{conn: conn, ch: p.engineEventsCh()}
	}
	own := make(chan engine.Event, 1024)
	go func() {
		for ev := range conn.EventsCh() {
			own <- engine.Event{Type: engine.EventType(ev.Type), Data: ev.Data}
		}
		close(own)
	}()
	return &engineAdapter{conn: conn, ch: own}
}

func (e *engineAdapter) EventsCh() <-chan engine.Event { return e.ch }
func (e *engineAdapter) ErrorsCh() <-chan error        { return e.conn.ErrorsCh() }
func (e *engineAdapter) Write(data []byte) error       { return e.conn.Write(data) }

// ── mockConnection ────────────────────────────────────────────────────────────

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

// ── wiring helpers ────────────────────────────────────────────────────────────

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

func mapSavePath(serverName string) string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	safe := strings.ToLower(strings.ReplaceAll(serverName, " ", "_"))
	if safe == "" {
		safe = "default"
	}
	return filepath.Join(dir, "kuda", "maps", safe+".json")
}
