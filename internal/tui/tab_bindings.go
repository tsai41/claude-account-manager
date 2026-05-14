package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// viewBindings renders the Bindings tab: a table of directory -> profile rows
// with a cursor for navigation and unbind (d).
func (m Model) viewBindings() string {
	var b strings.Builder

	const (
		cursorW  = 2
		profileW = 14
	)

	hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(clrBright)
	b.WriteString(strings.Repeat(" ", cursorW))
	b.WriteString(hdrStyle.Render(padRight("Profile", profileW)) + " ")
	b.WriteString(hdrStyle.Render("Directory") + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", 78)) + "\n")

	if len(m.bindings) == 0 {
		b.WriteString(dimStyle.Render("  (no bindings — use `ccm bind <profile> <dir>` to add one)") + "\n")
		return b.String()
	}

	cursorStyle := lipgloss.NewStyle().Width(cursorW).Bold(true).Foreground(clrCursor)
	profileBase := lipgloss.NewStyle().Width(profileW).Foreground(clrStatus)
	pathBase := lipgloss.NewStyle().Foreground(clrSub)

	for i, bind := range m.bindings {
		selected := i == m.bindingsCursor
		glyph := " "
		profileStyle := profileBase
		pathStyle := pathBase
		if selected {
			glyph = "▸"
			profileStyle = profileBase.Foreground(clrCursor).Bold(true)
			pathStyle = pathBase.Foreground(clrBright)
		}
		b.WriteString(cursorStyle.Render(glyph) +
			profileStyle.Render(bind.Profile) + " " +
			pathStyle.Render(collapseHome(bind.Pattern)) + "\n")
	}
	return b.String()
}

// clampBindingsCursor keeps bindingsCursor inside the current slice bounds.
func (m *Model) clampBindingsCursor() {
	switch {
	case len(m.bindings) == 0:
		m.bindingsCursor = 0
	case m.bindingsCursor < 0:
		m.bindingsCursor = 0
	case m.bindingsCursor >= len(m.bindings):
		m.bindingsCursor = len(m.bindings) - 1
	}
}
