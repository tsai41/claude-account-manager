// Package tui implements ccm's terminal UI: tabbed Profiles / Costs / Activity /
// History views plus modal overlays (help, profile detail, confirmations, edits).
//
// File layout in this package:
//   - styles.go      lipgloss styles and family color picker
//   - model.go       Model struct, async msg types, New, reload, lazyLoadTab
//   - update.go      Update routing, per-tab and per-overlay updaters
//   - view.go        top-level View() and tab bar rendering
//   - tab_costs.go   Costs tab body
//   - tab_activity.go Activity tab body
//   - tab_history.go History tab body
//   - overlays.go    Help and profile detail overlays
//   - tui.go         this file: Run() entry point
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Run launches the TUI program. Errors when no profiles exist yet so the user
// gets a clear bootstrap hint instead of an empty table.
func Run() error {
	m, err := New()
	if err != nil {
		return err
	}
	if len(m.profileRows) == 0 {
		return fmt.Errorf("no profiles to display; run `ccm import-current <name>` first")
	}
	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
