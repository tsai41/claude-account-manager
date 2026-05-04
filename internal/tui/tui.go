package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/tsai41/claude-account-manager/internal/jsonlscan"
	"github.com/tsai41/claude-account-manager/internal/profile"
	"github.com/tsai41/claude-account-manager/internal/switcher"
	"github.com/tsai41/claude-account-manager/internal/usage"
)

type tabID int

const (
	tabProfiles tabID = iota
	tabCosts
	tabActivity
)

var tabNames = []string{"Profiles", "Costs", "Activity"}

type viewMode int

const (
	modeTable viewMode = iota
	modeConfirmDelete
	modeEditNote
	modeEditUsage
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	tabStyle      = lipgloss.NewStyle().Padding(0, 2).Foreground(lipgloss.Color("245"))
	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(lipgloss.Color("99"))
	costAmountStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("46"))
	subStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
)

type Model struct {
	tab       tabID
	table     table.Model
	mode      viewMode
	status    string
	errMsg    string
	noteIn    textinput.Model
	noteFor   string
	usageIn   textinput.Model
	usageFor  string
	delFor    string
	current   string
	costs     jsonlscan.CostStats
	costsErr  error
	stats     jsonlscan.Activity
	statsErr  error
}

func New() (Model, error) {
	m := Model{}
	if err := m.reload(); err != nil {
		return m, err
	}
	ti := textinput.New()
	ti.Placeholder = "usage note..."
	ti.CharLimit = 200
	ti.Width = 60
	m.noteIn = ti

	ui := textinput.New()
	ui.Placeholder = "session 42%, weekly 68%"
	ui.CharLimit = 200
	ui.Width = 60
	m.usageIn = ui
	return m, nil
}

func (m *Model) reload() error {
	profs, err := profile.List()
	if err != nil {
		return err
	}
	st, _ := profile.LoadState()
	m.current = st.CurrentProfile

	cols := []table.Column{
		{Title: "", Width: 2},
		{Title: "Name", Width: 14},
		{Title: "Email", Width: 28},
		{Title: "Session Left", Width: 13},
		{Title: "Weekly Left", Width: 12},
		{Title: "Last Used", Width: 16},
		{Title: "Note", Width: 28},
	}
	rows := make([]table.Row, 0, len(profs))
	for _, p := range profs {
		mark := " "
		if p.Name == st.CurrentProfile {
			mark = "*"
		}
		u, _ := usage.Load(p.Name)
		session := usage.Remaining(u.Session.Display)
		weekly := usage.Remaining(u.Weekly.Display)
		last := "--"
		if !p.LastUsedAt.IsZero() {
			last = p.LastUsedAt.Format("2006-01-02 15:04")
		}
		email := p.Email
		if email == "" {
			email = "--"
		}
		rows = append(rows, table.Row{mark, p.Name, email, session, weekly, last, u.Note})
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(12),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true)
	t.SetStyles(s)
	m.table = t
	return nil
}

func (m *Model) loadCosts() {
	m.costs, m.costsErr = jsonlscan.ScanCosts()
}

func (m *Model) loadActivity() {
	m.stats, m.statsErr = jsonlscan.Scan()
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) currentRowName() string {
	row := m.table.SelectedRow()
	if len(row) < 2 {
		return ""
	}
	return row[1]
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Global tab switching keys (only outside text-input modes)
	if k, ok := msg.(tea.KeyMsg); ok && m.mode != modeEditNote && m.mode != modeEditUsage {
		switch k.String() {
		case "tab", "right", "l":
			if m.mode == modeTable {
				m.tab = (m.tab + 1) % tabID(len(tabNames))
				m.lazyLoadTab()
				return m, nil
			}
		case "shift+tab", "left", "h":
			if m.mode == modeTable {
				m.tab = (m.tab + tabID(len(tabNames)) - 1) % tabID(len(tabNames))
				m.lazyLoadTab()
				return m, nil
			}
		case "1":
			if m.mode == modeTable {
				m.tab = tabProfiles
				return m, nil
			}
		case "2":
			if m.mode == modeTable {
				m.tab = tabCosts
				m.lazyLoadTab()
				return m, nil
			}
		case "3":
			if m.mode == modeTable {
				m.tab = tabActivity
				m.lazyLoadTab()
				return m, nil
			}
		}
	}

	switch m.mode {
	case modeConfirmDelete:
		return m.updateConfirmDelete(msg)
	case modeEditNote:
		return m.updateEditNote(msg)
	case modeEditUsage:
		return m.updateEditUsage(msg)
	}

	switch m.tab {
	case tabProfiles:
		return m.updateProfilesTab(msg)
	case tabCosts:
		return m.updateCostsTab(msg)
	case tabActivity:
		return m.updateActivityTab(msg)
	}
	return m, nil
}

func (m *Model) lazyLoadTab() {
	switch m.tab {
	case tabCosts:
		if m.costs.Today.Window == "" && m.costsErr == nil {
			m.loadCosts()
		}
	case tabActivity:
		if m.stats.LastActive.IsZero() && m.statsErr == nil {
			m.loadActivity()
		}
	}
}

func (m Model) updateProfilesTab(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
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
		case "r":
			if err := m.reload(); err != nil {
				m.errMsg = err.Error()
			} else {
				m.status = "Refreshed"
				m.errMsg = ""
			}
			return m, nil
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
			m.loadCosts()
			m.status = "Costs refreshed"
			m.errMsg = ""
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateActivityTab(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			m.loadActivity()
			m.status = "Activity refreshed"
			m.errMsg = ""
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
	default:
		switch m.tab {
		case tabProfiles:
			b.WriteString(m.table.View())
			b.WriteString("\n\n")
			b.WriteString(helpStyle.Render("Tab/1-3 switch tab  j/k move  Enter switch  e edit-usage  u edit-note  d delete  r refresh  q quit"))
		case tabCosts:
			b.WriteString(m.viewCosts())
			b.WriteString("\n")
			b.WriteString(helpStyle.Render("Tab cycle tabs  r refresh  q quit"))
		case tabActivity:
			b.WriteString(m.viewActivity())
			b.WriteString("\n")
			b.WriteString(helpStyle.Render("Tab cycle tabs  r refresh  q quit"))
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

func (m Model) viewCosts() string {
	if m.costsErr != nil {
		return errStyle.Render("scan error: " + m.costsErr.Error())
	}
	c := m.costs
	var b strings.Builder
	b.WriteString(subStyle.Render("Today (machine-wide, list-price estimate)"))
	b.WriteString("\n")
	b.WriteString(costAmountStyle.Render(fmt.Sprintf("$%.2f", c.Today.Cost)))
	b.WriteString(fmt.Sprintf("   %d turns   %s tokens   %d sessions   active %s",
		c.Today.Turns, humanTokensTUI(c.Today.Tokens.Total()),
		c.Today.Sessions, formatDurationTUI(c.Today.ActiveDur)))
	b.WriteString("\n")
	for _, fb := range c.Today.ByFamily {
		b.WriteString(fmt.Sprintf("  %-8s  %5d turns  %8s  $%.2f\n",
			fb.Family, fb.Turns, humanTokensTUI(fb.Tokens.Total()), fb.Cost))
	}
	b.WriteString("\n")
	b.WriteString(subStyle.Render("Last 7 days  : "))
	b.WriteString(fmt.Sprintf("$%.2f  (%d turns, %s tokens)\n",
		c.Last7.Cost, c.Last7.Turns, humanTokensTUI(c.Last7.Tokens.Total())))
	b.WriteString(subStyle.Render("Last 30 days : "))
	b.WriteString(fmt.Sprintf("$%.2f  (%d turns, %s tokens)\n",
		c.Last30.Cost, c.Last30.Turns, humanTokensTUI(c.Last30.Tokens.Total())))

	if len(c.Last30.DailyTotals) > 0 {
		b.WriteString("\n")
		b.WriteString(subStyle.Render("Daily history:\n"))
		max := 0.0
		for _, d := range c.Last30.DailyTotals {
			if d.Cost > max {
				max = d.Cost
			}
		}
		shown := c.Last30.DailyTotals
		if len(shown) > 10 {
			shown = shown[:10]
		}
		for _, d := range shown {
			bar := bar20(d.Cost, max)
			b.WriteString(fmt.Sprintf("  %s  $%-9.2f %s  %s\n",
				d.Date, d.Cost, bar, strings.Join(d.Families, ",")))
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Pricing: Opus $15/$75, Sonnet $3/$15, Haiku $1/$5 per 1M (in/out); cache adjusted. Not an invoice."))
	return b.String()
}

func (m Model) viewActivity() string {
	if m.statsErr != nil {
		return errStyle.Render("scan error: " + m.statsErr.Error())
	}
	s := m.stats
	var b strings.Builder
	b.WriteString(subStyle.Render("Local activity (machine-wide, from ~/.claude/projects/*.jsonl)"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Last 5 hours : %6d turns\n", s.Last5Hours))
	b.WriteString(fmt.Sprintf("Today (24h)  : %6d turns  (%d session(s))\n", s.Today, s.Sessions))
	b.WriteString(fmt.Sprintf("Last 7 days  : %6d turns\n", s.Last7Days))
	if !s.LastActive.IsZero() {
		b.WriteString(fmt.Sprintf("Last active  : %s\n", s.LastActive.Format("2006-01-02 15:04:05")))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Note: jsonl transcripts have no per-account binding;\nthese counts are machine-wide and not the official usage bar."))
	return b.String()
}

func humanTokensTUI(n int64) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d", n)
	case n < 1_000_000:
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	case n < 1_000_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	default:
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	}
}

func formatDurationTUI(d time.Duration) string {
	if d <= 0 {
		return "0m"
	}
	h := int(d / time.Hour)
	m := int((d % time.Hour) / time.Minute)
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

func bar20(v, max float64) string {
	if max <= 0 {
		return strings.Repeat(" ", 20)
	}
	n := int((v / max) * 20)
	if n < 0 {
		n = 0
	}
	if n > 20 {
		n = 20
	}
	return strings.Repeat("█", n) + strings.Repeat(" ", 20-n)
}

func Run() error {
	m, err := New()
	if err != nil {
		return err
	}
	if len(m.table.Rows()) == 0 {
		return fmt.Errorf("no profiles to display; run `ccm import-current <name>` first")
	}
	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
