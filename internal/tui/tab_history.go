package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/tsai41/claude-account-manager/internal/logger"
)

var (
	histTimeCol  = lipgloss.NewStyle().Width(20).Foreground(lipgloss.Color("244"))
	histEventCol = lipgloss.NewStyle().Width(22)
	histProfCol  = lipgloss.NewStyle().Width(14).Bold(true).Foreground(lipgloss.Color("231"))
	histMsgCol   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

func (m Model) viewHistory() string {
	if m.historyErr != nil {
		return errStyle.Render("read log error: " + m.historyErr.Error())
	}
	var b strings.Builder
	if len(m.history) == 0 {
		b.WriteString(dimStyle.Render("No log entries yet. Run `ccm import-current` or `ccm use` to populate."))
		return b.String()
	}
	relevant := make([]logger.Entry, 0, len(m.history))
	for _, e := range m.history {
		switch e.Event {
		// switch.start is omitted: switch.done already records the completed
		// switch a beat later, so showing both creates pure duplication.
		case "switch.done", "switch.email_mismatch", "import-current", "remove", "rollback":
			relevant = append(relevant, e)
		}
	}
	if len(relevant) == 0 {
		b.WriteString(dimStyle.Render("No switch / import / remove events in the recent log."))
		return b.String()
	}
	for i := len(relevant) - 1; i >= 0; i-- {
		e := relevant[i]
		ts := e.Time.Format("2006-01-02 15:04:05")
		evt := histEventCol.Render(e.Event)
		switch e.Event {
		case "switch.done":
			evt = histEventCol.Foreground(lipgloss.Color("36")).Render(e.Event)
		case "switch.email_mismatch", "remove":
			evt = histEventCol.Foreground(lipgloss.Color("196")).Render(e.Event)
		case "import-current":
			evt = histEventCol.Foreground(lipgloss.Color("231")).Bold(true).Render(e.Event)
		default:
			evt = histEventCol.Foreground(lipgloss.Color("250")).Render(e.Event)
		}
		prof := e.Profile
		if prof == "" {
			prof = "-"
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top,
			histTimeCol.Render(ts),
			evt,
			histProfCol.Render(prof),
			histMsgCol.Render(e.Message),
		)
		b.WriteString(row)
		b.WriteString("\n")
	}
	return b.String()
}
