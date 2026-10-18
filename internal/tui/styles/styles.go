package styles

import "charm.land/lipgloss/v2"

var (
	Title    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	Selected = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	Error    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	Success  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	Help     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)
