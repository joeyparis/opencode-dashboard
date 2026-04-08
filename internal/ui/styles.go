package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/joeyparis/opencode-dashboard/internal/domain"
)

var sectionHeaderStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#7D56F4"))

func attentionIcon(signal domain.AttentionSignal) string {
	switch signal {
	case domain.HasErrors:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("!")
	case domain.WaitingForInput:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF44FF")).Render("@")
	case domain.ActiveNow:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("*")
	case domain.NeedsResponse:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Render("?")
	case domain.StaleWork:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("~")
	case domain.PendingTodos:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#0088FF")).Render(".")
	default:
		return " "
	}
}

func attentionIconPlain(signal domain.AttentionSignal) string {
	switch signal {
	case domain.HasErrors:
		return "!"
	case domain.WaitingForInput:
		return "@"
	case domain.ActiveNow:
		return "*"
	case domain.NeedsResponse:
		return "?"
	case domain.StaleWork:
		return "~"
	case domain.PendingTodos:
		return "."
	default:
		return " "
	}
}

func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
