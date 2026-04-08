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

// RestoreSelection moves the cursor to the session with the given ID.
// If the session is not visible or not found, cursor is unchanged.
func (m *SessionListModel) RestoreSelection(sessionID string) {
	if sessionID == "" {
		return
	}
	rows := m.buildVisibleRows()
	for i, row := range rows {
		if row.kind != rowSession {
			continue
		}
		g := m.groups[row.groupIdx]
		if row.sessionIdx >= 0 && row.sessionIdx < len(g.Sessions) {
			if g.Sessions[row.sessionIdx].ID == sessionID {
				m.cursor = i
				return
			}
		}
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
		case " ":
			if len(rows) > 0 && m.cursor < len(rows) {
				row := rows[m.cursor]
				if row.kind == rowProject {
					m.collapsed[row.projectID] = !m.collapsed[row.projectID]
				}
			}
		case "left":
			if len(rows) > 0 && m.cursor < len(rows) {
				row := rows[m.cursor]
				if row.kind == rowSession {
					for i := m.cursor - 1; i >= 0; i-- {
						if rows[i].kind == rowProject && rows[i].projectID == row.projectID {
							m.cursor = i
							break
						}
					}
				} else if row.kind == rowProject {
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

	// Truncate to pane width to prevent line wrapping
	if m.width > 0 {
		if runes := []rune(label); len(runes) > m.width {
			label = string(runes[:m.width-1]) + "…"
		}
	}

	style := lipgloss.NewStyle().Bold(true)
	if selected {
		style = style.Reverse(true)
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

	icon := attentionIconPlain(sv.AttentionSignal)
	if !selected {
		icon = attentionIcon(sv.AttentionSignal)
	}

	// Prefer the session title; fall back to slug for default/empty titles
	label := sv.Title
	if isDefaultSessionTitle(label) {
		label = sv.Slug
	}
	if label == "" {
		label = sv.ID[:min(8, len(sv.ID))]
	}

	relTime := relativeTime(sv.TimeUpdated)

	var todoRaw string
	if sv.TotalTodoCount > 0 {
		todoRaw = fmt.Sprintf("%d/%d", sv.CompletedTodoCount, sv.TotalTodoCount)
	}

	marker := "  "
	if selected {
		marker = "> "
	}

	// Fixed visible cols: marker(2) + [icon](3) + space(1) + sep(2) + todo(7) + sep(2) + age(8) = 25
	labelWidth := 24
	if m.width > 26 {
		labelWidth = m.width - 25
	}
	if runes := []rune(label); len(runes) > labelWidth {
		label = string(runes[:labelWidth-1]) + "…"
	}

	// Order: label | todo (right-aligned, fixed 7) | age (right-aligned, fixed 8)
	inner := fmt.Sprintf("%s[%s] %-*s  %7s  %8s", marker, icon, labelWidth, label, todoRaw, relTime)

	var style lipgloss.Style
	if selected {
		style = lipgloss.NewStyle().Reverse(true)
	} else {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))
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

func isDefaultSessionTitle(title string) bool {
	return title == "" ||
		strings.HasPrefix(title, "New session - ") ||
		strings.HasPrefix(title, "Child session - ")
}
