package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/tsai41/claude-account-manager/internal/format"
)

func (m Model) tabSubtitle() string {
	switch m.tab {
	case tabProfiles:
		cur := m.current
		if cur == "" {
			cur = "none"
		}
		fetchInfo := ""
		if m.fetchingOAuth {
			fetchInfo = "  ·  fetching..."
		} else if !m.lastFetched.IsZero() {
			fetchInfo = "  ·  fetched " + format.RelTime(m.lastFetched)
		}
		return subtitleStyle.Render("Profiles") + mutedSubStyle.Render(fmt.Sprintf("  ·  current: %s%s", cur, fetchInfo))
	case tabCosts:
		return subtitleStyle.Render("Costs") + mutedSubStyle.Render("  ·  machine-wide  ·  last 30d  ·  API-equivalent at list price")
	case tabActivity:
		return subtitleStyle.Render("Activity") + mutedSubStyle.Render("  ·  machine-wide  ·  from ~/.claude/projects/*.jsonl")
	case tabHistory:
		return subtitleStyle.Render("History") + mutedSubStyle.Render(fmt.Sprintf("  ·  %d recent events", len(m.history)))
	case tabConfig:
		return subtitleStyle.Render("Config") + mutedSubStyle.Render("  ·  stored in ~/.ccm/config.json")
	}
	return ""
}

func (m Model) renderTabs() string {
	var parts []string
	for i, name := range tabNames {
		if tabID(i) == m.tab {
			parts = append(parts, activeTabStyle.Render(fmt.Sprintf("%d %s", i+1, name)))
		} else {
			parts = append(parts, tabStyle.Render(fmt.Sprintf("%d %s", i+1, name)))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("ccm"))
	b.WriteString(dimStyle.Render("  —  Claude account manager"))
	if m.current != "" {
		b.WriteString("  ")
		b.WriteString(statusStyle.Render("● " + m.current))
	}
	b.WriteString("\n")
	b.WriteString(m.renderTabs())
	b.WriteString("\n")
	if m.width > 0 {
		b.WriteString(dimStyle.Render(strings.Repeat("─", m.width)))
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
	}

	switch m.mode {
	case modeConfirmDelete:
		b.WriteString(m.table.View())
		b.WriteString("\n\n")
		b.WriteString(errStyle.Render(fmt.Sprintf("Delete profile %q? (y/N) ", m.delFor)))
	case modeConfirmSwitch:
		b.WriteString(m.table.View())
		b.WriteString("\n\n")
		b.WriteString(statusStyle.Render(fmt.Sprintf("Switch to profile %q? (Y/n) ", m.confirmSwitch)))
	case modeEditNote:
		b.WriteString(m.table.View())
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("Edit note for %s (Enter to save, Esc to cancel):\n", m.noteFor))
		b.WriteString(m.noteIn.View())
	case modeEditUsage:
		b.WriteString(m.table.View())
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("Edit usage for %s (Enter to save, Esc to cancel):\n", m.usageFor))
		b.WriteString(m.usageIn.View())
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("Tip: \"session 42%, weekly 68%\" parses both fields."))
	case modeHelp:
		b.WriteString(m.viewHelp())
	case modeDetail:
		b.WriteString(m.viewDetail())
	default:
		header := m.tabSubtitle()
		var body string
		switch m.tab {
		case tabProfiles:
			body = m.table.View()
		case tabConfig:
			body = m.viewConfig()
		case tabCosts, tabActivity, tabHistory:
			body = m.bodyVP.View()
		}
		panel := lipgloss.JoinVertical(lipgloss.Left,
			header,
			"",
			body,
		)
		b.WriteString(panelStyle.Render(panel))
		b.WriteString("\n")

		var footer string
		switch m.tab {
		case tabProfiles:
			footer = "? help  j/k move  Enter switch  r reload  R refetch  q quit"
		case tabConfig:
			footer = "j/k move  ←/→ cycle  s save  r reset  q quit"
			if m.configDirty {
				footer = errStyle.Render("⚠ unsaved") + helpStyle.Render("  —  press s to save  |  Tab/q to discard")
			}
		case tabCosts, tabActivity, tabHistory:
			scroll := ""
			if !(m.bodyVP.AtTop() && m.bodyVP.AtBottom()) {
				pct := int(m.bodyVP.ScrollPercent() * 100)
				marker := "↕"
				switch {
				case m.bodyVP.AtTop():
					marker = "↓"
				case m.bodyVP.AtBottom():
					marker = "↑"
				}
				scroll = fmt.Sprintf("  [%s %d%%]", marker, pct)
			}
			footer = "↑/↓ scroll  r refresh  q quit" + scroll
		}
		b.WriteString(helpStyle.Render(footer))
	}

	if m.status != "" {
		b.WriteString("\n")
		b.WriteString(statusStyle.Render(m.status))
	}
	if m.errMsg != "" {
		b.WriteString("\n")
		b.WriteString(errStyle.Render("error: " + m.errMsg))
	}
	return b.String()
}
