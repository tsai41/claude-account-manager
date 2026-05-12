package tui

import "github.com/charmbracelet/lipgloss"

// Palette — change these constants to retheme the whole TUI.
const (
	clrAccent   = lipgloss.Color("99")  // purple: title, active tab
	clrSubtitle = lipgloss.Color("213") // pink: tab subtitles
	clrSelBg    = lipgloss.Color("57")  // purple: selected row background
	clrSelFg    = lipgloss.Color("229") // cream: selected row foreground
	clrCursor   = lipgloss.Color("212") // pink: config cursor ▸
	clrBorder   = lipgloss.Color("240") // border / header separator
	clrDim      = lipgloss.Color("244") // dim / muted text
	clrMuted    = lipgloss.Color("245") // secondary text
	clrSub      = lipgloss.Color("250") // slightly brighter secondary
	clrBright   = lipgloss.Color("231") // bright white: values
	clrStatus   = lipgloss.Color("36")  // teal: status / current profile indicator
	clrErr      = lipgloss.Color("196") // red: errors
	clrHelp     = lipgloss.Color("241") // dark gray: footer hints
	clrToday    = lipgloss.Color("214") // orange: today row
	clrCost     = lipgloss.Color("46")  // green: cost amounts
	clrOpus     = lipgloss.Color("208")
	clrSonnet   = lipgloss.Color("75")
	clrHaiku    = lipgloss.Color("84")
)

var (
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(clrAccent)
	helpStyle       = lipgloss.NewStyle().Foreground(clrHelp)
	statusStyle     = lipgloss.NewStyle().Foreground(clrStatus)
	errStyle        = lipgloss.NewStyle().Foreground(clrErr)
	tabStyle        = lipgloss.NewStyle().Padding(0, 2).Foreground(clrMuted)
	activeTabStyle  = lipgloss.NewStyle().Padding(0, 2).Bold(true).Foreground(clrBright).Background(clrAccent)
	costAmountStyle = lipgloss.NewStyle().Bold(true).Foreground(clrCost)
	subStyle        = lipgloss.NewStyle().Foreground(clrSub)
	dimStyle        = lipgloss.NewStyle().Foreground(clrDim)
	cardStyle       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(clrBorder).Padding(0, 2)
	cardLabel       = lipgloss.NewStyle().Foreground(clrMuted)
	cardValue       = lipgloss.NewStyle().Bold(true).Foreground(clrBright)
	todayRow        = lipgloss.NewStyle().Bold(true).Foreground(clrToday)
	familyOpus      = lipgloss.NewStyle().Foreground(clrOpus)
	familySonnet    = lipgloss.NewStyle().Foreground(clrSonnet)
	familyHaiku     = lipgloss.NewStyle().Foreground(clrHaiku)
	familyOther     = lipgloss.NewStyle().Foreground(clrMuted)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clrBorder).
			Padding(0, 2)
	subtitleStyle = lipgloss.NewStyle().Bold(true).Foreground(clrSubtitle)
	mutedSubStyle = lipgloss.NewStyle().Foreground(clrDim)
	cfgKeyCol     = lipgloss.NewStyle().Width(22).Foreground(clrSub)
	cfgValCol     = lipgloss.NewStyle().Width(14).Bold(true).Foreground(clrBright)
	cfgHintCol    = lipgloss.NewStyle().Width(42).Foreground(clrDim).Italic(true)
	cfgRowSel     = lipgloss.NewStyle().Background(clrSelBg).Foreground(clrBright)
	cfgCursor     = lipgloss.NewStyle().Width(2).Bold(true).Foreground(clrCursor)
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
