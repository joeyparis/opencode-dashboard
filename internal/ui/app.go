package ui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	appcore "github.com/joeyparis/opencode-dashboard/internal/app"
	"github.com/joeyparis/opencode-dashboard/internal/domain"
	"github.com/joeyparis/opencode-dashboard/internal/filter"
	"github.com/joeyparis/opencode-dashboard/internal/launcher"
)

var headerStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#7D56F4")).
	Padding(0, 1)

var errorBannerStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#B00020")).
	Padding(0, 1)

var loadingStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#AAAAAA")).
	Padding(0, 1)

type dataLoadedMsg struct {
	groups []domain.ProjectGroup
}

type dataLoadErrorMsg struct {
	err error
}

type refreshMsg struct{}

type refreshDataLoadedMsg struct {
	groups []domain.ProjectGroup
}

type launchSessionMsg struct {
	session *domain.SessionView
}

type sessionResumeMsg struct {
	err error
}

func loadDataCmd(agg *appcore.Aggregator) tea.Cmd {
	return func() tea.Msg {
		groups, err := agg.LoadAll(context.Background())
		if err != nil {
			return dataLoadErrorMsg{err: err}
		}
		return dataLoadedMsg{groups: groups}
	}
}

func refreshDataCmd(agg *appcore.Aggregator) tea.Cmd {
	return func() tea.Msg {
		groups, err := agg.Refresh(context.Background())
		if err != nil {
			return dataLoadErrorMsg{err: err}
		}
		return refreshDataLoadedMsg{groups: groups}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return refreshMsg{}
	})
}

type AppModel struct {
	layout     LayoutModel
	allGroups  []domain.ProjectGroup
	width      int
	height     int
	aggregator *appcore.Aggregator
	loading    bool
	refreshing bool
	loadErr    error
}

func NewApp() AppModel {
	return NewAppWithAggregator(nil)
}

func NewAppWithAggregator(agg *appcore.Aggregator) AppModel {
	return AppModel{
		layout:     NewLayout(nil),
		aggregator: agg,
		loading:    agg != nil,
	}
}

func (m AppModel) Init() tea.Cmd {
	if m.aggregator != nil {
		return tea.Batch(m.layout.Init(), loadDataCmd(m.aggregator), tickCmd())
	}
	return m.layout.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dataLoadedMsg:
		m.allGroups = msg.groups
		m.loading = false
		m.loadErr = nil
		m.applyFilter()
		m.syncLayoutSize()
		return m, nil

	case dataLoadErrorMsg:
		m.loading = false
		m.refreshing = false
		m.loadErr = msg.err
		m.syncLayoutSize()
		return m, nil

	case refreshMsg:
		if m.aggregator == nil {
			return m, tickCmd()
		}
		m.refreshing = true
		return m, tea.Batch(refreshDataCmd(m.aggregator), tickCmd())

	case refreshDataLoadedMsg:
		selectedID := m.layout.SelectedSessionID()
		m.allGroups = msg.groups
		m.applyFilter()
		if selectedID != "" {
			m.layout.RestoreSelection(selectedID)
		}
		m.refreshing = false
		m.loadErr = nil
		m.syncLayoutSize()
		return m, nil

	case FilterChangedMsg:
		m.applyFilter()
		return m, nil

	case launchSessionMsg:
		cmd, err := launcher.LaunchCmd(msg.session.ID, msg.session.Directory)
		if err != nil {
			m.loadErr = fmt.Errorf("launch failed: %w", err)
			return m, nil
		}
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return sessionResumeMsg{err: err}
		})

	case sessionResumeMsg:
		if msg.err != nil {
			m.loadErr = msg.err
		}
		if m.aggregator != nil {
			m.refreshing = true
			return m, refreshDataCmd(m.aggregator)
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "q" && !m.layout.IsSearchActive() {
			return m, tea.Quit
		}
		if msg.String() == "r" && !m.layout.IsSearchActive() && m.aggregator != nil && !m.refreshing {
			m.refreshing = true
			return m, refreshDataCmd(m.aggregator)
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncLayoutSize()
		return m, nil
	}
	updated, cmd := m.layout.Update(msg)
	m.layout = updated.(LayoutModel)
	return m, cmd
}

func (m *AppModel) applyFilter() {
	f := m.layout.CurrentFilter()

	allSessions := make([]domain.SessionView, 0, len(m.allGroups)*4)
	for _, g := range m.allGroups {
		allSessions = append(allSessions, g.Sessions...)
	}

	filtered := filter.Apply(allSessions, f)
	filteredGroups := regroupSessions(m.allGroups, filtered)

	m.layout.SetGroups(filteredGroups)
	m.layout.SetFilterCounts(len(filtered), len(allSessions))
}

func regroupSessions(allGroups []domain.ProjectGroup, filtered []domain.SessionView) []domain.ProjectGroup {
	filteredSet := make(map[string]struct{}, len(filtered))
	for _, sv := range filtered {
		filteredSet[sv.ID] = struct{}{}
	}

	result := make([]domain.ProjectGroup, 0, len(allGroups))
	for _, g := range allGroups {
		var sessions []domain.SessionView
		for _, sv := range g.Sessions {
			if _, ok := filteredSet[sv.ID]; ok {
				sessions = append(sessions, sv)
			}
		}
		if len(sessions) == 0 {
			continue
		}
		attentionCount := 0
		for _, sv := range sessions {
			if sv.AttentionSignal != domain.None {
				attentionCount++
			}
		}
		result = append(result, domain.ProjectGroup{
			Project:        g.Project,
			Sessions:       sessions,
			AttentionCount: attentionCount,
		})
	}
	return result
}

func (m *AppModel) syncLayoutSize() {
	if m.width == 0 || m.height == 0 {
		return
	}
	layoutHeight := m.height - m.chromeHeight()
	updated, _ := m.layout.Update(tea.WindowSizeMsg{
		Width:  m.width,
		Height: layoutHeight,
	})
	m.layout = updated.(LayoutModel)
}

func (m AppModel) chromeHeight() int {
	height := 2 // header + footer
	if m.loadErr != nil {
		height++
	}
	if m.loading {
		height++
	}
	return height
}

func (m AppModel) View() string {
	headerText := "OpenCode Dashboard"
	if m.refreshing {
		headerText += "  [Refreshing...]"
	}
	header := headerStyle.Render(headerText)
	parts := []string{header}
	if m.loadErr != nil {
		parts = append(parts, errorBannerStyle.Render(fmt.Sprintf("Error: %v", m.loadErr)))
	}
	if m.loading {
		parts = append(parts, loadingStyle.Render("Loading..."))
	}
	parts = append(parts, m.layout.View())
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Render("j/k up/down  h/l pane  ← jump/collapse  Tab filter  t time  / search  r refresh  Enter launch  q quit")
	parts = append(parts, footer)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
