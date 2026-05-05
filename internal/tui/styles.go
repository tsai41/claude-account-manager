package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	helpStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	statusStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))
	errStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	tabStyle        = lipgloss.NewStyle().Padding(0, 2).Foreground(lipgloss.Color("245"))
	activeTabStyle  = lipgloss.NewStyle().Padding(0, 2).Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("99"))
	costAmountStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("46"))
	subStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	dimStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	cardStyle       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 2)
	cardLabel       = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	cardValue       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231"))
	todayRow        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	familyOpus      = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	familySonnet    = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	familyHaiku     = lipgloss.NewStyle().Foreground(lipgloss.Color("84"))
	familyOther     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

func familyColor(name string) lipgloss.Style {
	switch name {
	case "Opus":
		return familyOpus
	case "Sonnet":
		return familySonnet
	case "Haiku":
		return familyHaiku
	default:
		return familyOther
	}
}
