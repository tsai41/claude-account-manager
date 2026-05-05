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
	"github.com/tsai41/claude-account-manager/internal/keychain"
	"github.com/tsai41/claude-account-manager/internal/profile"
	"github.com/tsai41/claude-account-manager/internal/snapshot"
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
	modeConfirmSwitch
	modeEditNote
	modeEditUsage
	modeHelp
	modeDetail
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
	dimStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	cardStyle       = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 2)
	cardLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	cardValue = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231"))
	todayRow  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	familyOpus   = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	familySonnet = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	familyHaiku  = lipgloss.NewStyle().Foreground(lipgloss.Color("84"))
	familyOther  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
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
	confirmSwitch string
	detailFor string
	current   string
	costs     jsonlscan.CostStats
	costsErr  error
	costsLoading bool
	stats     jsonlscan.Activity
	statsErr  error
	statsLoading bool
}

type costsLoadedMsg struct {
	cs  jsonlscan.CostStats
	err error
}
type statsLoadedMsg struct {
	a   jsonlscan.Activity
	err error
}

func loadCostsCmd() tea.Cmd {
	return func() tea.Msg {
		cs, err := jsonlscan.ScanCosts()
		return costsLoadedMsg{cs: cs, err: err}
	}
}
func loadStatsCmd() tea.Cmd {
	return func() tea.Msg {
		a, err := jsonlscan.Scan()
		return statsLoadedMsg{a: a, err: err}
	}
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

// loadCostsAsync returns a Cmd that triggers a background scan; results arrive via costsLoadedMsg.
func (m *Model) loadCostsAsync() tea.Cmd {
	if m.costsLoading {
		return nil
	}
	m.costsLoading = true
	return loadCostsCmd()
}

func (m *Model) loadStatsAsync() tea.Cmd {
	if m.statsLoading {
		return nil
	}
	m.statsLoading = true
	return loadStatsCmd()
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
	// Async scan results
	switch msg := msg.(type) {
	case costsLoadedMsg:
		m.costs = msg.cs
		m.costsErr = msg.err
		m.costsLoading = false
		return m, nil
	case statsLoadedMsg:
		m.stats = msg.a
		m.statsErr = msg.err
		m.statsLoading = false
		return m, nil
	}

	// Global tab switching keys (only outside text-input modes)
	if k, ok := msg.(tea.KeyMsg); ok && m.mode != modeEditNote && m.mode != modeEditUsage {
		switch k.String() {
		case "tab", "right", "l":
			if m.mode == modeTable {
				m.tab = (m.tab + 1) % tabID(len(tabNames))
				return m, m.lazyLoadTab()
			}
		case "shift+tab", "left", "h":
			if m.mode == modeTable {
				m.tab = (m.tab + tabID(len(tabNames)) - 1) % tabID(len(tabNames))
				return m, m.lazyLoadTab()
			}
		case "1":
			if m.mode == modeTable {
				m.tab = tabProfiles
				return m, nil
			}
		case "2":
			if m.mode == modeTable {
				m.tab = tabCosts
				return m, m.lazyLoadTab()
			}
		case "3":
			if m.mode == modeTable {
				m.tab = tabActivity
				return m, m.lazyLoadTab()
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
	}
	return m, nil
}

func (m *Model) lazyLoadTab() tea.Cmd {
	switch m.tab {
	case tabCosts:
		if m.costs.Today.Window == "" && m.costsErr == nil && !m.costsLoading {
			return m.loadCostsAsync()
		}
	case tabActivity:
		var cmds []tea.Cmd
		if m.stats.LastActive.IsZero() && m.statsErr == nil && !m.statsLoading {
			cmds = append(cmds, m.loadStatsAsync())
		}
		if m.costs.Today.Window == "" && m.costsErr == nil && !m.costsLoading {
			cmds = append(cmds, m.loadCostsAsync())
		}
		return tea.Batch(cmds...)
	}
	return nil
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
	return m, nil
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
	return m, nil
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
			b.WriteString(helpStyle.Render("? help  Tab/1-3 tab  j/k move  Enter switch  i info  e usage  u note  d delete  r refresh  q quit"))
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
	if m.costs.Today.Window == "" {
		return dimStyle.Render("Loading costs… (scanning ~/.claude/projects/*.jsonl)")
	}
	c := m.costs
	var b strings.Builder

	b.WriteString(dimStyle.Render("Scope: machine-wide · API-equivalent at list price · subscription bills are flat"))
	b.WriteString("\n\n")

	// Today card
	todayLines := []string{
		cardLabel.Render(fmt.Sprintf("Today's API-equivalent cost  ·  %s", time.Now().Format("Mon Jan 2"))),
		costAmountStyle.Render(fmt.Sprintf("$%.2f", c.Today.Cost)),
	}
	if len(c.Today.ByFamily) > 0 {
		todayLines = append(todayLines, dimStyle.Render(strings.Repeat("─", 38)))
		for _, fb := range c.Today.ByFamily {
			line := fmt.Sprintf("%s%s$%.2f",
				familyColor(fb.Family).Render(padRight(fb.Family, 10)),
				dimStyle.Render(padLeft(fmt.Sprintf("%s tokens", humanTokensTUI(fb.Tokens.Total())), 22)),
				fb.Cost)
			_ = line
			todayLines = append(todayLines,
				familyColor(fb.Family).Render(padRight(fb.Family, 10))+
					dimStyle.Render(padLeft(humanTokensTUI(fb.Tokens.Total())+" tok", 18))+
					cardValue.Render(fmt.Sprintf("  $%.2f", fb.Cost)))
		}
	}
	footer := fmt.Sprintf("%d sessions  ·  %s tokens  ·  active %s",
		c.Today.Sessions, humanTokensTUI(c.Today.Tokens.Total()), formatDurationTUI(c.Today.ActiveDur))
	todayLines = append(todayLines, dimStyle.Render(strings.Repeat("─", 38)), dimStyle.Render(footer))
	b.WriteString(cardStyle.Width(46).Render(strings.Join(todayLines, "\n")))
	b.WriteString("\n\n")

	// Two summary cards side-by-side
	mkCard := func(label string, r jsonlscan.CostReport) string {
		lines := []string{
			cardLabel.Render(label),
			cardValue.Render(fmt.Sprintf("$%.2f", r.Cost)),
			dimStyle.Render(fmt.Sprintf("%d turns · %s tok", r.Turns, humanTokensTUI(r.Tokens.Total()))),
		}
		return cardStyle.Width(22).Render(strings.Join(lines, "\n"))
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top,
		mkCard("Last 7 Days", c.Last7),
		"  ",
		mkCard("Last 30 Days", c.Last30),
	)
	b.WriteString(row)
	b.WriteString("\n\n")

	// Daily history
	if len(c.Last30.DailyTotals) > 0 {
		b.WriteString(subStyle.Render(fmt.Sprintf("Daily History  ·  Total: $%.2f", c.Last30.Cost)))
		b.WriteString("\n")
		max := 0.0
		for _, d := range c.Last30.DailyTotals {
			if d.Cost > max {
				max = d.Cost
			}
		}
		today := time.Now().Format("2006-01-02")
		shown := c.Last30.DailyTotals
		if len(shown) > 7 {
			shown = shown[:7]
		}
		for _, d := range shown {
			bar := bar20(d.Cost, max)
			date := strings.Replace(d.Date[5:], "-", "/", 1) // MM-DD
			fams := strings.Join(d.Families, ",")
			line := fmt.Sprintf("  %s   $%-8.2f  %s  %s", date, d.Cost, bar, dimStyle.Render(fams))
			if d.Date == today {
				line = todayRow.Render(fmt.Sprintf("▶ %s   $%-8.2f  ", date, d.Cost)) + bar + "  " + dimStyle.Render(fams)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString(helpStyle.Render("List price · Opus $15/$75 · Sonnet $3/$15 · Haiku $1/$5 per 1M (in/out) · Not an invoice"))
	return b.String()
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func padLeft(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return strings.Repeat(" ", w-len(s)) + s
}

func (m Model) viewHelp() string {
	rows := [][2]string{
		{"Navigation", ""},
		{"Tab / →", "next tab"},
		{"Shift+Tab / ←", "prev tab"},
		{"1 / 2 / 3", "jump to Profiles / Costs / Activity"},
		{"j / k / ↓ / ↑", "move row"},
		{"", ""},
		{"Profiles tab", ""},
		{"Enter", "switch to profile (Y/n confirm)"},
		{"i", "show profile detail (fp / snapshot / email)"},
		{"e", "edit usage value (parses session/weekly %)"},
		{"u", "edit note"},
		{"d", "delete profile (y/N confirm)"},
		{"r", "refresh"},
		{"", ""},
		{"Costs / Activity tabs", ""},
		{"r", "rescan jsonl transcripts"},
		{"", ""},
		{"Exit", ""},
		{"? / Esc / q / Enter", "close help / detail"},
		{"q / Ctrl+C", "quit"},
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("Keys"))
	b.WriteString("\n\n")
	for _, r := range rows {
		if r[1] == "" {
			b.WriteString(subStyle.Render(r[0]))
			b.WriteString("\n")
			continue
		}
		b.WriteString(fmt.Sprintf("  %-22s %s\n", cardValue.Render(r[0]), dimStyle.Render(r[1])))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Press any of: ? Esc q Enter to return"))
	return b.String()
}

func (m Model) viewDetail() string {
	var b strings.Builder
	p, err := profile.Load(m.detailFor)
	if err != nil {
		b.WriteString(errStyle.Render("load error: " + err.Error()))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("Esc / q / i / Enter to return"))
		return b.String()
	}
	u, _ := usage.Load(p.Name)
	snapDir, _ := snapshot.Latest(p.Name)
	bkTok, bkErr := keychain.ReadBackup(p.Name)

	b.WriteString(titleStyle.Render("Profile detail · " + p.Name))
	b.WriteString("\n\n")
	row := func(k, v string) {
		b.WriteString(fmt.Sprintf("  %-18s %s\n", cardLabel.Render(k), v))
	}
	row("Name", p.Name)
	row("Auth", p.AuthType)
	row("Email", p.Email)
	row("Account UUID", p.AccountUUID)
	row("Org", p.OrgName)
	row("Created", p.CreatedAt.Format("2006-01-02 15:04:05"))
	if !p.LastUsedAt.IsZero() {
		row("Last used", p.LastUsedAt.Format("2006-01-02 15:04:05"))
	}
	if snapDir != "" {
		row("Snapshot", snapDir)
	} else {
		row("Snapshot", errStyle.Render("(none)"))
	}
	if bkErr == nil {
		row("Keychain backup", "fp="+keychain.Fingerprint(bkTok))
	} else {
		row("Keychain backup", errStyle.Render(bkErr.Error()))
	}
	row("Usage left", "session "+usage.Remaining(u.Session.Display)+"  weekly "+usage.Remaining(u.Weekly.Display))
	if u.Manual != "" {
		row("Usage raw", u.Manual)
	}
	if u.Note != "" {
		row("Note", u.Note)
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Esc / q / i / Enter to return"))
	return b.String()
}

func (m Model) viewActivity() string {
	if m.statsErr != nil {
		return errStyle.Render("scan error: " + m.statsErr.Error())
	}
	if m.stats.LastActive.IsZero() && m.statsLoading {
		return dimStyle.Render("Loading activity… (scanning ~/.claude/projects/*.jsonl)")
	}
	s := m.stats
	c := m.costs
	var b strings.Builder
	b.WriteString(dimStyle.Render("Scope: machine-wide (jsonl has no account binding)"))
	b.WriteString("\n\n")

	// Today summary card
	todayLines := []string{
		cardLabel.Render(fmt.Sprintf("Today's Activity  ·  %s", time.Now().Format("Mon Jan 2"))),
		lipgloss.JoinHorizontal(lipgloss.Top,
			activityStat("Turns", fmt.Sprintf("%d", s.Today), 14),
			activityStat("Active", formatDurationTUI(c.Today.ActiveDur), 14),
			activityStat("Sessions", fmt.Sprintf("%d", s.Sessions), 14),
		),
	}
	if len(c.Today.ByFamily) > 0 {
		todayLines = append(todayLines, dimStyle.Render(strings.Repeat("─", 44)))
		var famLine []string
		for _, fb := range c.Today.ByFamily {
			famLine = append(famLine,
				familyColor(fb.Family).Render("● ")+
					cardValue.Render(fmt.Sprintf("%d", fb.Turns))+
					" "+dimStyle.Render(fb.Family))
		}
		todayLines = append(todayLines, strings.Join(famLine, "    "))
	}
	if !s.LastActive.IsZero() {
		todayLines = append(todayLines,
			dimStyle.Render(fmt.Sprintf("Last active: %s", s.LastActive.Format("2006-01-02 15:04:05"))))
	}
	b.WriteString(cardStyle.Width(50).Render(strings.Join(todayLines, "\n")))
	b.WriteString("\n\n")

	// Recent windows
	mkCard := func(label, val, sub string) string {
		lines := []string{
			cardLabel.Render(label),
			cardValue.Render(val),
			dimStyle.Render(sub),
		}
		return cardStyle.Width(22).Render(strings.Join(lines, "\n"))
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top,
		mkCard("Last 5 Hours", fmt.Sprintf("%d", s.Last5Hours), "turns"),
		"  ",
		mkCard("Last 7 Days", fmt.Sprintf("%d", s.Last7Days), "turns"),
	)
	b.WriteString(row)
	b.WriteString("\n\n")

	// Per-day turn bar (last 7 days from cost daily totals)
	if len(c.Last30.DailyTotals) > 0 {
		b.WriteString(subStyle.Render("Daily turns"))
		b.WriteString("\n")
		max := 0
		shown := c.Last30.DailyTotals
		if len(shown) > 7 {
			shown = shown[:7]
		}
		for _, d := range shown {
			if d.Turns > max {
				max = d.Turns
			}
		}
		today := time.Now().Format("2006-01-02")
		for _, d := range shown {
			bar := barInt(d.Turns, max, 24)
			line := fmt.Sprintf("  %s   %5d turns  %s", strings.Replace(d.Date[5:], "-", "/", 1), d.Turns, bar)
			if d.Date == today {
				line = todayRow.Render(fmt.Sprintf("▶ %s   %5d turns  ", strings.Replace(d.Date[5:], "-", "/", 1), d.Turns)) + bar
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Turn = user or assistant message in jsonl. Active = sum of <5min gaps."))
	return b.String()
}

func activityStat(label, value string, width int) string {
	v := cardValue.Render(value)
	l := dimStyle.Render(label)
	return lipgloss.NewStyle().Width(width).Render(v + "\n" + l)
}

func barInt(v, max, width int) string {
	if max <= 0 {
		return strings.Repeat(" ", width)
	}
	n := int(float64(v) / float64(max) * float64(width))
	if n < 0 {
		n = 0
	}
	if n > width {
		n = width
	}
	return strings.Repeat("█", n) + strings.Repeat(" ", width-n)
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
