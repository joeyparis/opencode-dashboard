package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/joeyparis/opencode-dashboard/internal/domain"
)

type rowKind int

const (
	rowProject rowKind = iota
	rowSession
)

type visibleRow struct {
	kind       rowKind
	projectID  string
	groupIdx   int // index into groups slice
	sessionIdx int // index within ProjectGroup.Sessions (-1 for project rows)
}

// SessionListModel is a Bubbletea sub-model for the left-pane project/session tree.
type SessionListModel struct {
	groups    []domain.ProjectGroup
	cursor    int             // index into visible rows
	collapsed map[string]bool // project ID -> collapsed
	width     int
	height    int
}

// NewSessionList constructs a SessionListModel with the given data.
func NewSessionList(groups []domain.ProjectGroup) SessionListModel {
	return SessionListModel{
		groups:    groups,
		cursor:    0,
		collapsed: make(map[string]bool),
	}
}

// SetGroups replaces the data without resetting cursor or collapse state.
func (m *SessionListModel) SetGroups(groups []domain.ProjectGroup) {
	m.groups = groups
	rows := m.buildVisibleRows()
	if m.cursor >= len(rows) {
		m.cursor = max(0, len(rows)-1)
	}
}

// SetSize sets the available width and height for rendering.
func (m *SessionListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SelectedSession returns the currently highlighted session, or nil if a project
// header is selected or there are no rows.
func (m *SessionListModel) SelectedSession() *domain.SessionView {
	rows := m.buildVisibleRows()
	if len(rows) == 0 || m.cursor >= len(rows) {
		return nil
	}
	row := rows[m.cursor]
	if row.kind != rowSession {
		return nil
	}
	g := m.groups[row.groupIdx]
	if row.sessionIdx < 0 || row.sessionIdx >= len(g.Sessions) {
		return nil
	}
	sv := g.Sessions[row.sessionIdx]
	return &sv
}

func (m SessionListModel) Init() tea.Cmd {
	return nil
}

func (m SessionListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		rows := m.buildVisibleRows()
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(rows)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if len(rows) > 0 && m.cursor < len(rows) {
				row := rows[m.cursor]
				if row.kind == rowProject {
					m.collapsed[row.projectID] = !m.collapsed[row.projectID]
				}
			}
		}
	}
	return m, nil
}

func (m SessionListModel) View() string {
	rows := m.buildVisibleRows()
	if len(rows) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Render("  No sessions")
	}

	lines := make([]string, 0, len(rows))
	for i, row := range rows {
		selected := i == m.cursor
		switch row.kind {
		case rowProject:
			lines = append(lines, m.renderProjectRow(row, selected))
		case rowSession:
			lines = append(lines, m.renderSessionRow(row, selected))
		}
	}

	if m.height > 0 {
		return m.scrollWindow(lines)
	}
	return strings.Join(lines, "\n")
}

func (m SessionListModel) buildVisibleRows() []visibleRow {
	rows := make([]visibleRow, 0, len(m.groups)*4)
	for gi, g := range m.groups {
		rows = append(rows, visibleRow{
			kind:       rowProject,
			projectID:  g.Project.ID,
			groupIdx:   gi,
			sessionIdx: -1,
		})
		if !m.collapsed[g.Project.ID] {
			for si := range g.Sessions {
				rows = append(rows, visibleRow{
					kind:       rowSession,
					projectID:  g.Project.ID,
					groupIdx:   gi,
					sessionIdx: si,
				})
			}
		}
	}
	return rows
}

func (m SessionListModel) renderProjectRow(row visibleRow, selected bool) string {
	g := m.groups[row.groupIdx]
	name := g.Project.DisplayName()

	arrow := "v"
	if m.collapsed[row.projectID] {
		arrow = ">"
	}

	label := fmt.Sprintf("%s %s (%d sessions", arrow, name, len(g.Sessions))
	if g.AttentionCount > 0 {
		label += fmt.Sprintf(", %d need attention", g.AttentionCount)
	}
	label += ")"

	style := lipgloss.NewStyle().Bold(true)
	if selected {
		style = style.
			Background(lipgloss.Color("#3A3A5C")).
			Foreground(lipgloss.Color("#FFFFFF"))
	} else {
		style = style.Foreground(lipgloss.Color("#AAAAFF"))
	}

	if m.width > 0 {
		style = style.Width(m.width)
	}
	return style.Render(label)
}

func (m SessionListModel) renderSessionRow(row visibleRow, selected bool) string {
	sv := m.groups[row.groupIdx].Sessions[row.sessionIdx]

	icon := attentionIcon(sv.AttentionSignal)

	slug := sv.Slug
	if slug == "" {
		slug = sv.ID[:min(8, len(sv.ID))]
	}

	relTime := relativeTime(sv.TimeUpdated)

	var todoStr string
	if sv.TotalTodoCount > 0 {
		todoStr = fmt.Sprintf("  %d/%d", sv.PendingTodoCount, sv.TotalTodoCount)
	}

	marker := "  "
	if selected {
		marker = "> "
	}

	inner := fmt.Sprintf("%s[%s] %-24s  %-8s%s", marker, icon, slug, relTime, todoStr)

	var style lipgloss.Style
	if selected {
		style = lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1E3A")).
			Foreground(lipgloss.Color("#E0E0FF"))
	} else {
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC"))
	}

	if m.width > 0 {
		style = style.Width(m.width)
	}
	return style.Render(inner)
}

func (m SessionListModel) scrollWindow(lines []string) string {
	total := len(lines)
	if total <= m.height {
		return strings.Join(lines, "\n")
	}
	half := m.height / 2
	start := m.cursor - half
	if start < 0 {
		start = 0
	}
	end := start + m.height
	if end > total {
		end = total
		start = end - m.height
		if start < 0 {
			start = 0
		}
	}
	return strings.Join(lines[start:end], "\n")
}
