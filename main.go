package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirageglobe/kuda/engine"
	"github.com/mirageglobe/kuda/mapper"
	"github.com/mirageglobe/kuda/network"
	"github.com/mirageglobe/kuda/ui"
)

var (
	version   = "dev"
	buildTime = "unknown" //nolint:unused
)

// rootModel owns model transitions and network client creation.
// It is the only place allowed to import both ui and network.
type rootModel struct {
	current           tea.Model
	conn              ui.Connection
	engine            *engine.Engine
	mapper            *mapper.Mapper
	pendingServerName string
	width, height     int
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
		m.pendingServerName = msg.Name
		if msg.Address == ui.MockAddress {
			m.mapper = mapper.Load(mapSavePath(msg.Name))
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
		m.mapper = mapper.Load(mapSavePath(m.pendingServerName))
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

func main() {
	ui.AppVersion = version
	model := rootModel{
		current: ui.NewSplashModel(),
	}
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("error running program: %v\n", err)
		os.Exit(1)
	}
}
