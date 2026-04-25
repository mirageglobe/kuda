package main

import (
	"fmt"
	"os"

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
	if sel, ok := msg.(ui.ServerSelectedMsg); ok {
		client := network.NewClient()
		if err := client.Connect(sel.Address); err != nil {
			var cmd tea.Cmd
			m.current, cmd = m.current.Update(ui.ConnectErrorMsg{Err: err})
			return m, cmd
		}
		next := ui.NewClientModel(newClientAdapter(client))
		m.current = next
		return m, next.Init()
	}
	var cmd tea.Cmd
	m.current, cmd = m.current.Update(msg)
	return m, cmd
}

func (m rootModel) View() string { return m.current.View() }

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

func main() {
	model := rootModel{current: ui.NewLaunchModel()}
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("error running program: %v\n", err)
		os.Exit(1)
	}
}
