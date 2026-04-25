package ui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mirageglobe/kuda/network"
)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type LaunchModel struct {
	list list.Model
}

func NewLaunchModel() LaunchModel {
	items := []list.Item{
		item{title: "Aardwolf", desc: "aardwolf.com:4000"},
		item{title: "TorilMUD", desc: "torilmud.com:9999"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select a MUD"
	return LaunchModel{list: l}
}

func (m LaunchModel) Init() tea.Cmd { return nil }

func (m LaunchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			i := m.list.SelectedItem().(item)
			client := network.NewClient()
			err := client.Connect(i.desc)
			if err != nil {
				return m, nil
			}
			newModel := NewClientModel(client)
			return newModel, func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.list.Width(), Height: m.list.Height()}
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m LaunchModel) View() string { return m.list.View() }
