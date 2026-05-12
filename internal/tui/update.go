package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tsai41/claude-account-manager/internal/jsonlscan"
	"github.com/tsai41/claude-account-manager/internal/switcher"
	"github.com/tsai41/claude-account-manager/internal/usage"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		bodyH := msg.Height - 8
		if bodyH < 6 {
			bodyH = 6
		}
		m.bodyVP.Width = msg.Width
		m.bodyVP.Height = bodyH
		tableH := bodyH - 1
		if tableH < 4 {
			tableH = 4
		}
		m.table.SetHeight(tableH)
		m.refreshBodyVP()
		return m, nil
	case costsLoadedMsg:
		m.costs = msg.cs
		m.costsErr = msg.err
		m.costsLoading = false
		m.refreshBodyVP()
		return m, nil
	case statsLoadedMsg:
		m.stats = msg.a
		m.statsErr = msg.err
		m.statsLoading = false
		m.refreshBodyVP()
		return m, nil
	case oauthRefetchMsg:
		return m, tea.Batch(m.refetchAllOAuthCmd(), oauthTickCmd())
	case oauthUsageMsg:
		return m, nil
	case oauthBatchDoneMsg:
		ok, fail := 0, 0
		var lastErr string
		for _, r := range msg.results {
			if r.err != nil {
				fail++
				lastErr = fmt.Sprintf("%s: %v", r.profile, r.err)
				continue
			}
			if err := usage.ApplyOAuth(r.profile, r.u); err != nil {
				fail++
				lastErr = fmt.Sprintf("%s save: %v", r.profile, err)
				continue
			}
			ok++
		}
		_ = m.reload()
		if fail == 0 {
			m.status = fmt.Sprintf("OAuth usage updated: %d profile(s)", ok)
			m.errMsg = ""
		} else if ok == 0 {
			m.errMsg = "oauth fetch failed: " + lastErr
		} else {
			m.status = fmt.Sprintf("OAuth updated %d, %d failed", ok, fail)
			m.errMsg = lastErr
		}
		return m, nil
	}

	if k, ok := msg.(tea.KeyMsg); ok && m.mode != modeEditNote && m.mode != modeEditUsage {
		switch k.String() {
		case "tab", "right":
			if m.mode == modeTable {
				m.tab = (m.tab + 1) % tabID(len(tabNames))
				cmd := m.lazyLoadTab()
				if m.tab == tabHistory {
					m.loadHistory()
				}
				m.refreshBodyVP()
				return m, cmd
			}
		case "shift+tab", "left":
			if m.mode == modeTable {
				m.tab = (m.tab + tabID(len(tabNames)) - 1) % tabID(len(tabNames))
				cmd := m.lazyLoadTab()
				if m.tab == tabHistory {
					m.loadHistory()
				}
				m.refreshBodyVP()
				return m, cmd
			}
		case "1":
			if m.mode == modeTable {
				m.tab = tabProfiles
				m.refreshBodyVP()
				return m, nil
			}
		case "2":
			if m.mode == modeTable {
				m.tab = tabCosts
				cmd := m.lazyLoadTab()
				m.refreshBodyVP()
				return m, cmd
			}
		case "3":
			if m.mode == modeTable {
				m.tab = tabActivity
				cmd := m.lazyLoadTab()
				m.refreshBodyVP()
				return m, cmd
			}
		case "4":
			if m.mode == modeTable {
				m.tab = tabHistory
				m.loadHistory()
				m.refreshBodyVP()
				return m, nil
			}
		}
	}

	switch m.mode {
	case modeConfirmDelete:
		return m.updateConfirmDelete(msg)
	case modeConfirmSwitch:
		return m.updateConfirmSwitch(msg)
	case modeEditNote:
		return m.updateEditNote(msg)
	case modeEditUsage:
		return m.updateEditUsage(msg)
	case modeHelp, modeDetail:
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "esc", "q", "?", "i", "enter":
				m.mode = modeTable
				m.detailFor = ""
				return m, nil
			}
		}
		return m, nil
	}

	switch m.tab {
	case tabProfiles:
		return m.updateProfilesTab(msg)
	case tabCosts:
		return m.updateCostsTab(msg)
	case tabActivity:
		return m.updateActivityTab(msg)
	case tabHistory:
		return m.updateHistoryTab(msg)
	}
	return m, nil
}

func (m Model) updateProfilesTab(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "j", "down":
			m.table.MoveDown(1)
			return m, nil
		case "k", "up":
			m.table.MoveUp(1)
			return m, nil
		case "enter":
			name := m.currentRowName()
			if name == "" {
				return m, nil
			}
			if name == m.current {
				m.status = fmt.Sprintf("Already on %s", name)
				return m, nil
			}
			m.confirmSwitch = name
			m.mode = modeConfirmSwitch
			m.errMsg = ""
			m.status = ""
			return m, nil
		case "r":
			if err := m.reload(); err != nil {
				m.errMsg = err.Error()
			} else {
				m.status = "Refreshed"
				m.errMsg = ""
			}
			return m, nil
		case "R":
			m.status = "Refetching OAuth usage..."
			m.errMsg = ""
			return m, m.refetchAllOAuthCmd()
		case "u":
			name := m.currentRowName()
			if name == "" {
				return m, nil
			}
			u, _ := usage.Load(name)
			m.noteIn.SetValue(u.Note)
			m.noteIn.Focus()
			m.noteFor = name
			m.mode = modeEditNote
			m.errMsg = ""
			m.status = ""
			return m, textinput.Blink
		case "e":
			name := m.currentRowName()
			if name == "" {
				return m, nil
			}
			u, _ := usage.Load(name)
			m.usageIn.SetValue(u.Manual)
			m.usageIn.Focus()
			m.usageFor = name
			m.mode = modeEditUsage
			m.errMsg = ""
			m.status = ""
			return m, textinput.Blink
		case "d":
			name := m.currentRowName()
			if name == "" {
				return m, nil
			}
			m.delFor = name
			m.mode = modeConfirmDelete
			m.errMsg = ""
			m.status = ""
			return m, nil
		case "i":
			name := m.currentRowName()
			if name == "" {
				return m, nil
			}
			m.detailFor = name
			m.mode = modeDetail
			m.errMsg = ""
			m.status = ""
			return m, nil
		case "?":
			m.mode = modeHelp
			m.errMsg = ""
			m.status = ""
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) updateCostsTab(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			m.costs = jsonlscan.CostStats{}
			m.costsErr = nil
			cmd := m.loadCostsAsync()
			m.status = "Refreshing costs..."
			m.errMsg = ""
			return m, cmd
		}
	}
	var cmd tea.Cmd
	m.bodyVP, cmd = m.bodyVP.Update(msg)
	return m, cmd
}

func (m Model) updateActivityTab(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			m.stats = jsonlscan.Activity{}
			m.statsErr = nil
			m.costs = jsonlscan.CostStats{}
			m.costsErr = nil
			cmd := tea.Batch(m.loadStatsAsync(), m.loadCostsAsync())
			m.status = "Refreshing..."
			m.errMsg = ""
			return m, cmd
		}
	}
	var cmd tea.Cmd
	m.bodyVP, cmd = m.bodyVP.Update(msg)
	return m, cmd
}

func (m Model) updateHistoryTab(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			m.loadHistory()
			m.refreshBodyVP()
			m.status = "History refreshed"
			m.errMsg = ""
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.bodyVP, cmd = m.bodyVP.Update(msg)
	return m, cmd
}

func (m Model) updateConfirmSwitch(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch strings.ToLower(k.String()) {
		case "y", "enter":
			name := m.confirmSwitch
			m.confirmSwitch = ""
			m.mode = modeTable
			res, err := switcher.Switch(name)
			if err != nil {
				m.errMsg = err.Error()
				m.status = ""
				return m, nil
			}
			m.errMsg = ""
			m.status = fmt.Sprintf("Switched to %s (fp=%s, backup=%s)", res.Profile.Name, res.TokenFP, res.BackupDir)
			_ = m.reload()
			return m, nil
		case "n", "esc", "q":
			m.confirmSwitch = ""
			m.mode = modeTable
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateConfirmDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch strings.ToLower(k.String()) {
		case "y":
			err := switcher.Remove(m.delFor, false)
			if err != nil {
				m.errMsg = err.Error()
			} else {
				m.status = fmt.Sprintf("Removed %s", m.delFor)
			}
			m.delFor = ""
			m.mode = modeTable
			_ = m.reload()
			return m, nil
		case "n", "esc", "q":
			m.delFor = ""
			m.mode = modeTable
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateEditUsage(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "enter":
			if err := usage.SetManual(m.usageFor, m.usageIn.Value()); err != nil {
				m.errMsg = err.Error()
			} else {
				m.status = fmt.Sprintf("Usage saved for %s", m.usageFor)
			}
			m.usageIn.Blur()
			m.usageFor = ""
			m.mode = modeTable
			_ = m.reload()
			return m, nil
		case "esc":
			m.usageIn.Blur()
			m.usageFor = ""
			m.mode = modeTable
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.usageIn, cmd = m.usageIn.Update(msg)
	return m, cmd
}

func (m Model) updateEditNote(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "enter":
			if err := usage.SetNote(m.noteFor, m.noteIn.Value()); err != nil {
				m.errMsg = err.Error()
			} else {
				m.status = fmt.Sprintf("Note saved for %s", m.noteFor)
			}
			m.noteIn.Blur()
			m.noteFor = ""
			m.mode = modeTable
			_ = m.reload()
			return m, nil
		case "esc":
			m.noteIn.Blur()
			m.noteFor = ""
			m.mode = modeTable
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.noteIn, cmd = m.noteIn.Update(msg)
	return m, cmd
}
