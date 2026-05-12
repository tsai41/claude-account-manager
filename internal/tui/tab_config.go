package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/tsai41/claude-account-manager/internal/config"
)

// configFieldCount must match the number of cases handled in cycleConfigValue.
const configFieldCount = 4

func (m Model) viewConfig() string {
	rows := []struct {
		key, value, hint string
	}{
		{"Usage display", m.settings.UsageDisplay, otherOpt(m.settings.UsageDisplay, "left", "used")},
		{"Reset time", m.settings.ResetDisplay, otherOpt(m.settings.ResetDisplay, "relative", "absolute")},
		{"OAuth fetch interval", fmt.Sprintf("%ds", m.settings.RefetchSeconds), "60 / 120 / 300 / 600 / 1200 / 1800 / 3600"},
		{"Per-profile delay", fmt.Sprintf("%ds", m.settings.FetchSpacingSeconds), "1 / 2 / 3 / 5 / 10 / 20"},
	}

	rendered := make([]string, 0, len(rows))
	for i, r := range rows {
		selected := i == m.configCursor
		rendered = append(rendered, renderConfigRow(r.key, r.value, r.hint, selected))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, rendered...)

	var note strings.Builder
	if envOverride := strings.TrimSpace(os.Getenv("CCM_USAGE_DISPLAY")); envOverride != "" {
		note.WriteString("\n")
		note.WriteString(errStyle.Render("CCM_USAGE_DISPLAY=" + envOverride + " is overriding Usage display."))
	}
	return lipgloss.JoinVertical(lipgloss.Left, body, note.String())
}

func renderConfigRow(key, value, hint string, selected bool) string {
	keyCol := cfgKeyCol
	valCol := cfgValCol
	hintCol := cfgHintCol
	cursorCol := cfgCursor
	cursorGlyph := " "
	if selected {
		keyCol = keyCol.Inherit(cfgRowSel)
		valCol = valCol.Inherit(cfgRowSel)
		hintCol = hintCol.Inherit(cfgRowSel)
		cursorCol = cursorCol.Inherit(cfgRowSel)
		cursorGlyph = "▸"
	}
	inner := lipgloss.JoinHorizontal(lipgloss.Top,
		cursorCol.Render(cursorGlyph),
		keyCol.Render(key),
		valCol.Render(value),
		hintCol.Render(hint),
	)
	if selected {
		return cfgRowSel.Render(inner)
	}
	return inner
}

func (m Model) updateConfigTab(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch k.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "j", "down":
		m.configCursor = (m.configCursor + 1) % configFieldCount
		m.refreshBodyVP()
		return m, nil
	case "k", "up":
		m.configCursor = (m.configCursor + configFieldCount - 1) % configFieldCount
		m.refreshBodyVP()
		return m, nil
	case "enter", " ", "l", "right":
		m.settings = cycleConfigValue(m.settings, m.configCursor, +1)
		m.refreshBodyVP()
		return m, nil
	case "h", "left":
		m.settings = cycleConfigValue(m.settings, m.configCursor, -1)
		m.refreshBodyVP()
		return m, nil
	case "s":
		if err := config.Save(m.settings); err != nil {
			m.errMsg = "save config: " + err.Error()
		} else {
			m.status = "Config saved"
			m.errMsg = ""
			_ = m.reload()
		}
		m.refreshBodyVP()
		// Restart refetch ticker with new cadence.
		return m, oauthTickCmd(m.settings.RefetchInterval())
	case "r":
		m.settings = config.Defaults()
		m.status = "Reset to defaults (press s to save)"
		m.errMsg = ""
		m.refreshBodyVP()
		return m, nil
	}
	return m, nil
}

// cycleConfigValue advances/rewinds a single field through its allowed values.
func cycleConfigValue(s config.Settings, field, dir int) config.Settings {
	switch field {
	case 0:
		if s.UsageDisplay == config.DisplayUsed {
			s.UsageDisplay = config.DisplayLeft
		} else {
			s.UsageDisplay = config.DisplayUsed
		}
	case 1:
		if s.ResetDisplay == config.ResetAbsolute {
			s.ResetDisplay = config.ResetRelative
		} else {
			s.ResetDisplay = config.ResetAbsolute
		}
	case 2:
		s.RefetchSeconds = cycleInt(s.RefetchSeconds, []int{60, 120, 300, 600, 1200, 1800, 3600}, dir)
	case 3:
		s.FetchSpacingSeconds = cycleInt(s.FetchSpacingSeconds, []int{1, 2, 3, 5, 10, 20}, dir)
	}
	return s
}

func cycleInt(cur int, opts []int, dir int) int {
	idx := 0
	for i, v := range opts {
		if v == cur {
			idx = i
			break
		}
	}
	if dir >= 0 {
		idx = (idx + 1) % len(opts)
	} else {
		idx = (idx + len(opts) - 1) % len(opts)
	}
	return opts[idx]
}

func otherOpt(cur, a, b string) string {
	if cur == a {
		return "→ " + b
	}
	return "→ " + a
}
