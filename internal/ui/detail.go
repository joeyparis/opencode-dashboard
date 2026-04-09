package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/joeyparis/opencode-dashboard/internal/config"
	"github.com/joeyparis/opencode-dashboard/internal/domain"
)

type DetailModel struct {
	session      *domain.SessionView
	width        int
	height       int
	scrollOffset int
	display      config.DisplayConfig
}

func NewDetail() DetailModel {
	return DetailModel{
		display: config.Default().Display,
	}
}

func (m *DetailModel) SetSession(s *domain.SessionView) {
	m.session = s
	m.scrollOffset = 0
}

func (m *DetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m DetailModel) Init() tea.Cmd {
	return nil
}

func (m DetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.scrollOffset++
		case "k", "up":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		}
	}
	return m, nil
}

func (m DetailModel) View() string {
	if m.session == nil {
		placeholder := "Select a session"
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))
		if m.width > 0 && m.height > 0 {
			style = style.
				Width(m.width).
				Height(m.height).
				Align(lipgloss.Center, lipgloss.Center)
		}
		return style.Render(placeholder)
	}

	sections := []string{
		m.renderHeader(),
		m.renderMetadata(),
		m.renderStats(),
		m.renderTodos(),
		m.renderLastActivity(),
	}

	var parts []string
	for _, s := range sections {
		if s != "" {
			parts = append(parts, s)
		}
	}

	content := strings.Join(parts, "\n\n")
	allLines := strings.Split(content, "\n")

	if m.height > 0 {
		start := m.scrollOffset
		if start >= len(allLines) {
			start = max(0, len(allLines)-1)
		}
		end := start + m.height
		if end > len(allLines) {
			end = len(allLines)
		}
		allLines = allLines[start:end]
	}

	return strings.Join(allLines, "\n")
}

func (m DetailModel) renderHeader() string {
	if m.session == nil {
		return ""
	}
	title := m.session.Title
	if title == "" {
		title = m.session.Slug
	}
	badge := attentionIcon(m.session.AttentionSignal, m.display)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA"))
	return titleStyle.Render(title) + " " + badge
}

func (m DetailModel) renderMetadata() string {
	if m.session == nil {
		return ""
	}
	s := m.session
	header := sectionHeaderStyle.Render("Metadata")

	relTime := relativeTime(s.TimeUpdated)
	absTime := s.TimeUpdated.Format("Jan 2 15:04")

	lines := []string{
		header,
		fmt.Sprintf("  Project:   %s", s.ProjectName),
		fmt.Sprintf("  Directory: %s", s.Directory),
		fmt.Sprintf("  Slug:      %s", s.Slug),
		fmt.Sprintf("  Version:   %s", s.Version),
		fmt.Sprintf("  Updated:   %s (%s)", relTime, absTime),
	}
	return strings.Join(lines, "\n")
}

func (m DetailModel) renderStats() string {
	if m.session == nil {
		return ""
	}
	s := m.session
	header := sectionHeaderStyle.Render("Stats")

	lines := []string{
		header,
		fmt.Sprintf("  Messages:  %d", s.MessageCount),
	}

	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88"))
	delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))
	changes := addStyle.Render(fmt.Sprintf("+%d", s.SummaryAdditions)) +
		" " +
		delStyle.Render(fmt.Sprintf("-%d", s.SummaryDeletions)) +
		fmt.Sprintf(", %d files", s.SummaryFiles)
	lines = append(lines, fmt.Sprintf("  Changes:   %s", changes))

	if s.ChildCount > 0 {
		lines = append(lines, fmt.Sprintf("  Children:  %d", s.ChildCount))
	}
	if s.ErrorCount > 0 {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
		lines = append(lines, fmt.Sprintf("  Errors:    %s", errStyle.Render(fmt.Sprintf("%d", s.ErrorCount))))
	}

	return strings.Join(lines, "\n")
}

func (m DetailModel) renderTodos() string {
	if m.session == nil || len(m.session.Todos) == 0 {
		return ""
	}

	header := sectionHeaderStyle.Render("Todos")
	lines := []string{header}

	for _, todo := range m.session.Todos {
		icon := todoStatusIcon(todo.Status)
		priority := todoPriorityBadge(todo.Priority)
		lines = append(lines, fmt.Sprintf("  %s %s %s", icon, priority, todo.Content))
	}

	return strings.Join(lines, "\n")
}

func (m DetailModel) renderLastActivity() string {
	if m.session == nil {
		return ""
	}
	lm := m.session.LastMessage
	if lm.TimeCreated.IsZero() {
		return ""
	}

	header := sectionHeaderStyle.Render("Last Activity")
	lines := []string{header}

	if lm.Agent != "" {
		lines = append(lines, fmt.Sprintf("  Agent:     %s", lm.Agent))
	}
	if lm.ModelID != "" {
		lines = append(lines, fmt.Sprintf("  Model:     %s", lm.ModelID))
	}
	lines = append(lines, fmt.Sprintf("  Time:      %s", relativeTime(lm.TimeCreated)))

	return strings.Join(lines, "\n")
}

func todoStatusIcon(status string) string {
	switch status {
	case "completed":
		return "[x]"
	case "pending":
		return "[ ]"
	case "in_progress":
		return "[>]"
	case "cancelled":
		return "[-]"
	default:
		return "[ ]"
	}
}

func todoPriorityBadge(priority string) string {
	switch priority {
	case "high":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444")).Render("[H]")
	case "medium":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00")).Render("[M]")
	case "low":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")).Render("[L]")
	default:
		return "   "
	}
}
