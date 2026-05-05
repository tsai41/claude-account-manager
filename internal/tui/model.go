package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/tsai41/claude-account-manager/internal/format"
	"github.com/tsai41/claude-account-manager/internal/jsonlscan"
	"github.com/tsai41/claude-account-manager/internal/logger"
	"github.com/tsai41/claude-account-manager/internal/profile"
	"github.com/tsai41/claude-account-manager/internal/usage"
)

type tabID int

const (
	tabProfiles tabID = iota
	tabCosts
	tabActivity
	tabHistory
)

var tabNames = []string{"Profiles", "Costs", "Activity", "History"}

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

type Model struct {
	tab           tabID
	table         table.Model
	mode          viewMode
	status        string
	errMsg        string
	noteIn        textinput.Model
	noteFor       string
	usageIn       textinput.Model
	usageFor      string
	delFor        string
	confirmSwitch string
	detailFor     string
	current       string
	costs         jsonlscan.CostStats
	costsErr      error
	costsLoading  bool
	stats         jsonlscan.Activity
	statsErr      error
	statsLoading  bool
	history       []logger.Entry
	historyErr    error
	width, height int
	bodyVP        viewport.Model
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

	m.bodyVP = viewport.New(80, 16)
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
		email := format.MaskEmail(p.Email)
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

func (m *Model) loadHistory() {
	m.history, m.historyErr = logger.Tail(200)
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

// refreshBodyVP rerenders the viewport content for the active body tab.
func (m *Model) refreshBodyVP() {
	var content string
	switch m.tab {
	case tabCosts:
		content = m.viewCosts()
	case tabActivity:
		content = m.viewActivity()
	case tabHistory:
		content = m.viewHistory()
	default:
		return
	}
	m.bodyVP.SetContent(content)
}

func (m Model) currentRowName() string {
	row := m.table.SelectedRow()
	if len(row) < 2 {
		return ""
	}
	return row[1]
}

func (m Model) Init() tea.Cmd { return nil }
