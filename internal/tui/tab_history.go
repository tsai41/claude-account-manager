package tui

import (
	"fmt"
	"strings"

	"github.com/tsai41/claude-account-manager/internal/logger"
)

func (m Model) viewHistory() string {
	if m.historyErr != nil {
		return errStyle.Render("read log error: " + m.historyErr.Error())
	}
	var b strings.Builder
	b.WriteString(dimStyle.Render("Source: " + logger.LogPath()))
	b.WriteString("\n\n")
	if len(m.history) == 0 {
		b.WriteString(dimStyle.Render("No log entries yet. Run `ccm import-current` or `ccm use` to populate."))
		return b.String()
	}
	relevant := make([]logger.Entry, 0, len(m.history))
	for _, e := range m.history {
		switch e.Event {
		case "switch.start", "switch.done", "switch.email_mismatch", "import-current", "remove", "rollback":
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
		evt := e.Event
		switch evt {
		case "switch.done":
			evt = statusStyle.Render(evt)
		case "switch.email_mismatch", "remove":
			evt = errStyle.Render(evt)
		case "import-current":
			evt = cardValue.Render(evt)
		default:
			evt = subStyle.Render(evt)
		}
		prof := e.Profile
		if prof == "" {
			prof = "-"
		}
		b.WriteString(fmt.Sprintf("  %s  %-26s %-16s %s\n",
			dimStyle.Render(ts), evt, cardValue.Render(prof), dimStyle.Render(e.Message)))
	}
	return b.String()
}
