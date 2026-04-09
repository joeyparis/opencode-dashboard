package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	appcore "github.com/joeyparis/opencode-dashboard/internal/app"
	"github.com/joeyparis/opencode-dashboard/internal/config"
	"github.com/joeyparis/opencode-dashboard/internal/domain"
	"github.com/joeyparis/opencode-dashboard/internal/filter"
	"github.com/joeyparis/opencode-dashboard/internal/launcher"
)

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

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return refreshMsg{}
	})
}

type AppModel struct {
	cfg         config.Config
	headerStyle lipgloss.Style
	layout      LayoutModel
	allGroups   []domain.ProjectGroup
	width       int
	height      int
	aggregator  *appcore.Aggregator
	loading     bool
	refreshing  bool
	loadErr     error
}

func NewAppWithConfig(cfg config.Config, agg *appcore.Aggregator) AppModel {
	windowOptions := make([]domain.TimeWindowOption, 0, len(cfg.TimeWindows.Options)+1)
	for _, opt := range cfg.TimeWindows.Options {
		windowOptions = append(windowOptions, domain.TimeWindowOption{
			Label:    opt.Label,
			Duration: time.Duration(opt.Hours) * time.Hour,
		})
	}
	windowOptions = append(windowOptions, domain.TimeWindowOption{Label: "All", Duration: 0})

	defaultPreset := resolveDefaultPreset(cfg.Defaults.Filter)
	defaultWindowIdx := 0
	for i, opt := range windowOptions {
		if opt.Label == cfg.Defaults.TimeWindow {
			defaultWindowIdx = i
			break
		}
	}

	hs := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color(cfg.Display.ColorHeader)).
		Padding(0, 1)
	return AppModel{
		cfg:         cfg,
		headerStyle: hs,
		layout: NewLayout(
			nil,
			cfg.Keys,
			cfg.Display,
			cfg.Defaults.ListWidthRatio,
			windowOptions,
			defaultPreset,
			defaultWindowIdx,
		),
		aggregator: agg,
		loading:    agg != nil,
	}
}

func resolveDefaultPreset(filterStr string) domain.FilterPreset {
	switch filterStr {
	case "all_active":
		return domain.FilterAllActive
	case "archived":
		return domain.FilterArchived
	default:
		return domain.FilterNeedsAttention
	}
}

func NewApp() AppModel {
	return NewAppWithAggregator(nil)
}

func NewAppWithAggregator(agg *appcore.Aggregator) AppModel {
	return NewAppWithConfig(config.Default(), agg)
}

func (m AppModel) Init() tea.Cmd {
	interval := time.Duration(m.cfg.Defaults.RefreshInterval) * time.Second
	if m.aggregator != nil {
		return tea.Batch(m.layout.Init(), loadDataCmd(m.aggregator), tickCmd(interval))
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
		interval := time.Duration(m.cfg.Defaults.RefreshInterval) * time.Second
		if m.aggregator == nil {
			return m, tickCmd(interval)
		}
		m.refreshing = true
		return m, tea.Batch(refreshDataCmd(m.aggregator), tickCmd(interval))

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
		keyStr := msg.String()
		isQuit := config.Matches(keyStr, m.cfg.Keys.Quit)
		isCtrlC := keyStr == "ctrl+c"
		if isCtrlC || (isQuit && !m.layout.IsSearchActive()) {
			return m, tea.Quit
		}
		if config.Matches(keyStr, m.cfg.Keys.Refresh) && !m.layout.IsSearchActive() && m.aggregator != nil && !m.refreshing {
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

	filtered := filter.Apply(allSessions, f, time.Now())
	filteredGroups := regroupSessions(m.allGroups, filtered)

	m.layout.SetGroups(filteredGroups)
	m.layout.SetFilterCounts(len(filtered), len(allSessions))
}

func regroupSessions(allGroups []domain.ProjectGroup, filtered []domain.SessionView) []domain.ProjectGroup {
	filteredSet := make(map[string]struct{}, len(filtered))
	for _, sv := range filtered {
		filteredSet[sv.ID] = struct{}{}
	}

	grouped := make([]domain.ProjectGroup, 0, len(allGroups))
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
		grouped = append(grouped, domain.ProjectGroup{
			Project:        g.Project,
			Sessions:       sessions,
			AttentionCount: attentionCount,
		})
	}
	return grouped
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

func formatKeys(bindings []string) string {
	return strings.Join(bindings, "/")
}

func (m AppModel) View() string {
	headerText := "OpenCode Dashboard"
	if m.refreshing {
		headerText += "  [Refreshing...]"
	}
	header := m.headerStyle.Render(headerText)
	parts := []string{header}
	if m.loadErr != nil {
		parts = append(parts, errorBannerStyle.Render(fmt.Sprintf("Error: %v", m.loadErr)))
	}
	if m.loading {
		parts = append(parts, loadingStyle.Render("Loading..."))
	}
	parts = append(parts, m.layout.View())
	footerText := fmt.Sprintf(
		"%s up/down  %s/%s pane  \u2190 jump/collapse  %s filter  %s time  %s search  %s refresh  Enter launch  %s quit",
		formatKeys(m.cfg.Keys.Down),
		formatKeys(m.cfg.Keys.PaneLeft),
		formatKeys(m.cfg.Keys.PaneRight),
		formatKeys(m.cfg.Keys.CycleFilter),
		formatKeys(m.cfg.Keys.CycleTime),
		formatKeys(m.cfg.Keys.Search),
		formatKeys(m.cfg.Keys.Refresh),
		formatKeys(m.cfg.Keys.Quit),
	)
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Render(footerText)
	parts = append(parts, footer)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
