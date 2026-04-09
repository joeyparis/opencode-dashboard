package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/joeyparis/opencode-dashboard/internal/config"
	"github.com/joeyparis/opencode-dashboard/internal/domain"
	"github.com/joeyparis/opencode-dashboard/internal/filter"
)

// FilterChangedMsg is emitted when the filter preset or search text changes.
type FilterChangedMsg struct {
	Filter filter.Filter
}

// FilterBarModel is the filter bar UI component rendered above the split panes.
type FilterBarModel struct {
	keys          config.KeysConfig
	display       config.DisplayConfig
	preset        domain.FilterPreset
	windowOptions []domain.TimeWindowOption
	windowIdx     int
	searchMode    bool
	textInput     textinput.Model
	shownCount    int
	totalCount    int
	width         int
}

func (m *FilterBarModel) SetWidth(w int) { m.width = w }

// NewFilterBar creates a new FilterBarModel with the given config and defaults.
func NewFilterBar(keys config.KeysConfig, display config.DisplayConfig, windowOptions []domain.TimeWindowOption, defaultPreset domain.FilterPreset, defaultWindowIdx int) FilterBarModel {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 100
	if len(windowOptions) == 0 {
		windowOptions = []domain.TimeWindowOption{{Label: "All", Duration: 0}}
	}
	if defaultWindowIdx < 0 || defaultWindowIdx >= len(windowOptions) {
		defaultWindowIdx = 0
	}
	return FilterBarModel{
		keys:          keys,
		display:       display,
		preset:        defaultPreset,
		windowOptions: windowOptions,
		windowIdx:     defaultWindowIdx,
		textInput:     ti,
	}
}

// IsSearchMode returns true when the filter bar is accepting text input.
func (m FilterBarModel) IsSearchMode() bool {
	return m.searchMode
}

// CurrentFilter returns the active filter state.
func (m FilterBarModel) CurrentFilter() filter.Filter {
	return filter.Filter{
		Preset:     m.preset,
		SearchText: m.textInput.Value(),
		Window:     m.windowOptions[m.windowIdx].Duration,
	}
}

// SetCounts updates the shown/total session counters.
func (m *FilterBarModel) SetCounts(shown, total int) {
	m.shownCount = shown
	m.totalCount = total
}

func (m FilterBarModel) Init() tea.Cmd {
	return nil
}

func (m FilterBarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searchMode {
			switch msg.String() {
			case "esc":
				m.searchMode = false
				m.textInput.Blur()
				m.textInput.SetValue("")
				return m, emitFilterChanged(m)
			case "enter":
				m.searchMode = false
				m.textInput.Blur()
				return m, emitFilterChanged(m)
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}
		switch {
		case config.Matches(msg.String(), m.keys.CycleFilter):
			m.preset = nextPreset(m.preset)
			return m, emitFilterChanged(m)
		case config.Matches(msg.String(), m.keys.CycleTime):
			m.windowIdx = (m.windowIdx + 1) % len(m.windowOptions)
			return m, emitFilterChanged(m)
		case config.Matches(msg.String(), m.keys.Search):
			m.searchMode = true
			cmd := m.textInput.Focus()
			return m, cmd
		}
	default:
		// Forward non-key messages to textInput for cursor blink etc.
		if m.searchMode {
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m FilterBarModel) View() string {
	presets := []domain.FilterPreset{
		domain.FilterNeedsAttention,
		domain.FilterAllActive,
		domain.FilterArchived,
	}

	presetStrs := make([]string, 0, len(presets))
	for _, p := range presets {
		label := p.String()
		if p == m.preset {
			s := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color(m.display.ColorHeader))
			presetStrs = append(presetStrs, s.Render("["+label+"]"))
		} else {
			s := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			presetStrs = append(presetStrs, s.Render(label))
		}
	}

	filterPart := "Filter: " + strings.Join(presetStrs, " | ")

	windowPart := "Time: " + m.windowOptions[m.windowIdx].Label

	var searchPart string
	switch {
	case m.searchMode:
		searchPart = "Search: " + m.textInput.View()
	case m.textInput.Value() != "":
		searchPart = "Search: [" + m.textInput.Value() + "]"
	default:
		searchPart = "Search: ___"
	}

	counterPart := fmt.Sprintf("Showing %d of %d", m.shownCount, m.totalCount)

	color := func(c, text string) string {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(text)
	}
	legend := strings.Join([]string{
		color(m.display.ColorError, m.display.IconError) + " err",
		color(m.display.ColorWaiting, m.display.IconWaiting) + " waiting",
		color(m.display.ColorActive, m.display.IconActive) + " active",
		color(m.display.ColorReply, m.display.IconReply) + " reply",
		color(m.display.ColorStale, m.display.IconStale) + " stale",
		color(m.display.ColorTodos, m.display.IconTodos) + " todos",
	}, "  ")
	left := strings.Join([]string{filterPart, windowPart, searchPart, counterPart}, "  ")
	if m.width > 0 {
		gap := m.width - lipgloss.Width(left) - lipgloss.Width(legend)
		if gap > 0 {
			return left + strings.Repeat(" ", gap) + legend
		}
	}
	return left + "  " + legend
}

func nextPreset(p domain.FilterPreset) domain.FilterPreset {
	switch p {
	case domain.FilterNeedsAttention:
		return domain.FilterAllActive
	case domain.FilterAllActive:
		return domain.FilterArchived
	default:
		return domain.FilterNeedsAttention
	}
}

func emitFilterChanged(m FilterBarModel) tea.Cmd {
	f := filter.Filter{
		Preset:     m.preset,
		SearchText: m.textInput.Value(),
		Window:     m.windowOptions[m.windowIdx].Duration,
	}
	return func() tea.Msg {
		return FilterChangedMsg{Filter: f}
	}
}
