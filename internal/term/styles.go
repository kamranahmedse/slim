package term

import "charm.land/lipgloss/v2"

var (
	Green   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(2))
	Red     = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(1))
	Yellow  = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(3))
	Cyan    = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(6))
	Magenta = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(5))
	Dim     = lipgloss.NewStyle().Faint(true)
	Bold    = lipgloss.NewStyle().Bold(true)

	CheckMark = Green.Render("✓")
	CrossMark = Red.Render("✗")
	WarnMark  = Yellow.Render("!")
)

func StyleForStatus(code int) lipgloss.Style {
	switch {
	case code >= 500:
		return Red
	case code >= 400:
		return Yellow
	case code >= 300:
		return Cyan
	default:
		return Green
	}
}
