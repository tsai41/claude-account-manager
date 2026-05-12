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

	// panelStyle is the single rounded container every tab uses. Padding is
	// uniform so tabs do not jiggle when switching.
	panelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2)
	subtitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213"))
	mutedSubStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	cfgKeyCol  = lipgloss.NewStyle().Width(22).Foreground(lipgloss.Color("250"))
	cfgValCol  = lipgloss.NewStyle().Width(14).Bold(true).Foreground(lipgloss.Color("231"))
	cfgHintCol = lipgloss.NewStyle().Width(42).Foreground(lipgloss.Color("244")).Italic(true)
	cfgRowSel  = lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("231"))
	cfgCursor  = lipgloss.NewStyle().Width(2).Bold(true).Foreground(lipgloss.Color("212"))
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
