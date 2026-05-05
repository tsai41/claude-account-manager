package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
	b.WriteString(titleStyle.Render("ccm — Claude account manager"))
	b.WriteString("\n")
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

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
		switch m.tab {
		case tabProfiles:
			b.WriteString(m.table.View())
			b.WriteString("\n\n")
			b.WriteString(helpStyle.Render("? help  Tab/1-4 tab  j/k move  Enter switch  i info  e usage  u note  d delete  r refresh  q quit"))
		case tabCosts, tabActivity, tabHistory:
			b.WriteString(m.bodyVP.View())
			b.WriteString("\n")
			b.WriteString(helpStyle.Render("Tab cycle tabs  ↑/↓ scroll  r refresh  q quit"))
		}
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
