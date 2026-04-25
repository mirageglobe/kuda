package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirageglobe/kuda/network"
	"github.com/mirageglobe/kuda/ui"
)

// rootModel owns model transitions and network client creation.
// It is the only place allowed to import both ui and network.
type rootModel struct {
	current tea.Model
}

func (m rootModel) Init() tea.Cmd { return m.current.Init() }

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.ServerSelectedMsg:
		if msg.Address == ui.MockAddress {
			next := ui.NewClientModel(newMockConnection())
			m.current = next
			return m, next.Init()
		}
		client := network.NewClient()
		return m, dialCmd(client, msg.Address)

	case dialResultMsg:
		if msg.err != nil {
			var cmd tea.Cmd
			m.current, cmd = m.current.Update(ui.ConnectErrorMsg{Err: msg.err})
			return m, cmd
		}
		next := ui.NewClientModel(msg.adapter)
		m.current = next
		return m, next.Init()
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
}

var _ ui.Connection = (*clientAdapter)(nil)

func newClientAdapter(client *network.Client) *clientAdapter {
	a := &clientAdapter{
		client: client,
		events: make(chan ui.Event, 100),
	}
	go a.forward()
	return a
}

func (a *clientAdapter) forward() {
	for ev := range a.client.EventsCh() {
		a.events <- mapEvent(ev)
	}
	close(a.events)
}

func (a *clientAdapter) Write(data []byte) error   { return a.client.Write(data) }
func (a *clientAdapter) EventsCh() <-chan ui.Event { return a.events }
func (a *clientAdapter) ErrorsCh() <-chan error    { return a.client.ErrorsCh() }

func mapEvent(ev network.Event) ui.Event {
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

func (m *mockConnection) EventsCh() <-chan ui.Event { return m.events }
func (m *mockConnection) ErrorsCh() <-chan error    { return m.errors }

// ── entry point ──────────────────────────────────────────────────────────────

func main() {
	model := rootModel{current: ui.NewLaunchModel()}
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("error running program: %v\n", err)
		os.Exit(1)
	}
}
