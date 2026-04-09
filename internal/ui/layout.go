package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/joeyparis/opencode-dashboard/internal/config"
	"github.com/joeyparis/opencode-dashboard/internal/domain"
	"github.com/joeyparis/opencode-dashboard/internal/filter"
)

const (
	paneLeft  = 0
	paneRight = 1

	filterBarHeight = 1
)

type LayoutModel struct {
	filterBar  FilterBarModel
	list       SessionListModel
	detail     DetailModel
	activePane int
	width      int
	height     int
}

func NewLayout(groups []domain.ProjectGroup) LayoutModel {
	defaultOptions := []domain.TimeWindowOption{
		{Label: "1d", Duration: 24 * time.Hour},
		{Label: "3d", Duration: 72 * time.Hour},
		{Label: "7d", Duration: 168 * time.Hour},
		{Label: "All", Duration: 0},
	}
	return LayoutModel{
		filterBar: NewFilterBar(config.Default().Keys, config.Default().Display, defaultOptions, domain.FilterNeedsAttention, 1),
		list:      NewSessionList(groups, config.Default().Keys, config.Default().Display),
		detail:    NewDetail(),
	}
}

func (m *LayoutModel) SetGroups(groups []domain.ProjectGroup) {
	m.list.SetGroups(groups)
}

func (m *LayoutModel) SelectedSessionID() string {
	sel := m.list.SelectedSession()
	if sel == nil {
		return ""
	}
	return sel.ID
}

func (m *LayoutModel) RestoreSelection(sessionID string) {
	m.list.RestoreSelection(sessionID)
}

// CurrentFilter returns the active filter from the filter bar.
func (m LayoutModel) CurrentFilter() filter.Filter {
	return m.filterBar.CurrentFilter()
}

// SetFilterCounts updates the shown/total counters displayed in the filter bar.
func (m *LayoutModel) SetFilterCounts(shown, total int) {
	m.filterBar.SetCounts(shown, total)
}

// IsSearchActive returns true when the filter bar is accepting text input.
func (m LayoutModel) IsSearchActive() bool {
	return m.filterBar.IsSearchMode()
}

func (m LayoutModel) Init() tea.Cmd {
	return tea.Batch(m.list.Init(), m.detail.Init(), m.filterBar.Init())
}

func (m LayoutModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		leftWidth, rightWidth := m.paneWidths()
		borderAndTitle := 3
		contentH := max(0, m.height-filterBarHeight-borderAndTitle)
		leftInnerW := max(0, leftWidth-2)
		rightInnerW := max(0, rightWidth-2)
		m.list.SetSize(leftInnerW, contentH)
		m.detail.SetSize(rightInnerW, contentH)
		m.filterBar.SetWidth(m.width)
		return m, nil

	case tea.KeyMsg:
		// Tab always cycles filter presets.
		if msg.String() == "tab" {
			updated, cmd := m.filterBar.Update(msg)
			m.filterBar = updated.(FilterBarModel)
			return m, cmd
		}

		if msg.String() == "t" && !m.filterBar.IsSearchMode() {
			updated, cmd := m.filterBar.Update(msg)
			m.filterBar = updated.(FilterBarModel)
			return m, cmd
		}

		// While in search mode all keys route to the filter bar.
		if m.filterBar.IsSearchMode() {
			updated, cmd := m.filterBar.Update(msg)
			m.filterBar = updated.(FilterBarModel)
			return m, cmd
		}

		// '/' activates search.
		if msg.String() == "/" {
			updated, cmd := m.filterBar.Update(msg)
			m.filterBar = updated.(FilterBarModel)
			return m, cmd
		}

		// Left arrow: switch pane from right→left, or tree-nav within left pane.
		if msg.String() == "left" {
			if m.activePane == paneRight {
				m.activePane = paneLeft
				return m, nil
			}
			updated, cmd := m.list.Update(msg)
			m.list = updated.(SessionListModel)
			m.detail.SetSession(m.list.SelectedSession())
			return m, cmd
		}

		// Pane switching.
		switch msg.String() {
		case "h":
			m.activePane = paneLeft
			return m, nil
		case "l", "right":
			m.activePane = paneRight
			return m, nil
		case "enter":
			if m.activePane == paneLeft {
				if sel := m.list.SelectedSession(); sel != nil {
					return m, func() tea.Msg { return launchSessionMsg{session: sel} }
				}
			}
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

	default:
		// Forward non-key messages to filterBar (textinput cursor blink etc.).
		updated, cmd := m.filterBar.Update(msg)
		m.filterBar = updated.(FilterBarModel)
		return m, cmd
	}
}

func (m LayoutModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Waiting for terminal size..."
	}

	filterBarView := m.filterBar.View()

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

	leftInnerW := max(0, leftWidth-2)
	rightInnerW := max(0, rightWidth-2)
	paneInnerH := max(1, m.height-filterBarHeight-2)

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

	panesView := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	return lipgloss.JoinVertical(lipgloss.Left, filterBarView, panesView)
}

func (m LayoutModel) paneWidths() (left, right int) {
	left = m.width / 2
	right = max(0, m.width-left-1)
	return left, right
}
