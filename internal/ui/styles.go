package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/joeyparis/opencode-dashboard/internal/config"
	"github.com/joeyparis/opencode-dashboard/internal/domain"
)

var sectionHeaderStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#7D56F4"))

func iconAndColor(signal domain.AttentionSignal, d config.DisplayConfig) (string, string) {
	switch signal {
	case domain.HasErrors:
		return d.IconError, d.ColorError
	case domain.WaitingForInput:
		return d.IconWaiting, d.ColorWaiting
	case domain.ActiveNow:
		return d.IconActive, d.ColorActive
	case domain.NeedsResponse:
		return d.IconReply, d.ColorReply
	case domain.StaleWork:
		return d.IconStale, d.ColorStale
	case domain.PendingTodos:
		return d.IconTodos, d.ColorTodos
	default:
		return "", ""
	}
}

func attentionIcon(signal domain.AttentionSignal, d config.DisplayConfig) string {
	icon, color := iconAndColor(signal, d)
	if icon == "" {
		return " "
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(icon)
}

func attentionIconPlain(signal domain.AttentionSignal, d config.DisplayConfig) string {
	icon, _ := iconAndColor(signal, d)
	if icon == "" {
		return " "
	}
	return icon
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
