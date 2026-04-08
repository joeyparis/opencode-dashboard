package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/joeyparis/opencode-dashboard/internal/domain"
	"github.com/joeyparis/opencode-dashboard/internal/filter"
)

// FilterChangedMsg is emitted when the filter preset or search text changes.
type FilterChangedMsg struct {
	Filter filter.Filter
}

// FilterBarModel is the filter bar UI component rendered above the split panes.
type FilterBarModel struct {
	preset     domain.FilterPreset
	searchMode bool
	textInput  textinput.Model
	shownCount int
	totalCount int
}

// NewFilterBar creates a new FilterBarModel defaulting to NeedsAttention.
func NewFilterBar() FilterBarModel {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 100
	return FilterBarModel{
		preset:    domain.FilterNeedsAttention,
		textInput: ti,
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
		switch msg.String() {
		case "tab":
			m.preset = nextPreset(m.preset)
			return m, emitFilterChanged(m)
		case "/":
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
				Background(lipgloss.Color("#7D56F4"))
			presetStrs = append(presetStrs, s.Render("["+label+"]"))
		} else {
			s := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			presetStrs = append(presetStrs, s.Render(label))
		}
	}

	filterPart := "Filter: " + strings.Join(presetStrs, " | ")

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

	return strings.Join([]string{filterPart, searchPart, counterPart}, "  ")
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
	}
	return func() tea.Msg {
		return FilterChangedMsg{Filter: f}
	}
}
