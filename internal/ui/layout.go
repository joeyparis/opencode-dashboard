package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/joeyparis/opencode-dashboard/internal/domain"
)

const (
	paneLeft  = 0
	paneRight = 1
)

type LayoutModel struct {
	list       SessionListModel
	detail     DetailModel
	activePane int
	width      int
	height     int
}

func NewLayout(groups []domain.ProjectGroup) LayoutModel {
	return LayoutModel{
		list:   NewSessionList(groups),
		detail: NewDetail(),
	}
}

func (m *LayoutModel) SetGroups(groups []domain.ProjectGroup) {
	m.list.SetGroups(groups)
}

func (m LayoutModel) Init() tea.Cmd {
	return tea.Batch(m.list.Init(), m.detail.Init())
}

func (m LayoutModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		leftWidth, rightWidth := m.paneWidths()
		borderAndTitle := 3
		contentH := m.height - borderAndTitle
		if contentH < 0 {
			contentH = 0
		}
		leftInnerW := leftWidth - 2
		if leftInnerW < 0 {
			leftInnerW = 0
		}
		rightInnerW := rightWidth - 2
		if rightInnerW < 0 {
			rightInnerW = 0
		}
		m.list.SetSize(leftInnerW, contentH)
		m.detail.SetSize(rightInnerW, contentH)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "h", "left":
			m.activePane = paneLeft
			return m, nil
		case "l", "right":
			m.activePane = paneRight
			return m, nil
		}

		if m.activePane == paneLeft {
			updated, cmd := m.list.Update(msg)
			m.list = updated.(SessionListModel)
			m.detail.SetSession(m.list.SelectedSession())
			return m, cmd
		}
		updated, cmd := m.detail.Update(msg)
		m.detail = updated.(DetailModel)
		return m, cmd
	}

	return m, nil
}

func (m LayoutModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Waiting for terminal size..."
	}

	leftWidth, rightWidth := m.paneWidths()

	leftTitle := "[ ] Sessions"
	rightTitle := "[ ] Details"
	if m.activePane == paneLeft {
		leftTitle = "[*] Sessions"
	} else {
		rightTitle = "[*] Details"
	}

	activeColor := lipgloss.Color("#7D56F4")
	inactiveColor := lipgloss.Color("#555555")

	leftBorderColor := inactiveColor
	rightBorderColor := inactiveColor
	if m.activePane == paneLeft {
		leftBorderColor = activeColor
	} else {
		rightBorderColor = activeColor
	}

	leftInnerW := leftWidth - 2
	if leftInnerW < 0 {
		leftInnerW = 0
	}
	rightInnerW := rightWidth - 2
	if rightInnerW < 0 {
		rightInnerW = 0
	}
	paneInnerH := m.height - 2
	if paneInnerH < 1 {
		paneInnerH = 1
	}

	leftPane := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(leftBorderColor).
		Width(leftInnerW).
		Height(paneInnerH).
		Render(leftTitle + "\n" + m.list.View())

	rightPane := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(rightBorderColor).
		Width(rightInnerW).
		Height(paneInnerH).
		Render(rightTitle + "\n" + m.detail.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}

func (m LayoutModel) paneWidths() (left, right int) {
	left = m.width * 2 / 5
	right = m.width - left - 1
	if right < 0 {
		right = 0
	}
	return left, right
}
