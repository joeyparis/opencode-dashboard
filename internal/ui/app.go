package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var headerStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#7D56F4")).
	Padding(0, 1)

type AppModel struct {
	layout LayoutModel
	width  int
	height int
}

func NewApp() AppModel {
	return AppModel{
		layout: NewLayout(nil),
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.layout.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		layoutMsg := tea.WindowSizeMsg{
			Width:  msg.Width,
			Height: msg.Height - 1,
		}
		updated, cmd := m.layout.Update(layoutMsg)
		m.layout = updated.(LayoutModel)
		return m, cmd
	}
	updated, cmd := m.layout.Update(msg)
	m.layout = updated.(LayoutModel)
	return m, cmd
}

func (m AppModel) View() string {
	header := headerStyle.Render("OpenCode Dashboard")
	return header + "\n" + m.layout.View()
}
