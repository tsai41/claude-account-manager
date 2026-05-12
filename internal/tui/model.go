package tui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/tsai41/claude-account-manager/internal/config"
	"github.com/tsai41/claude-account-manager/internal/format"
	"github.com/tsai41/claude-account-manager/internal/jsonlscan"
	"github.com/tsai41/claude-account-manager/internal/keychain"
	"github.com/tsai41/claude-account-manager/internal/logger"
	"github.com/tsai41/claude-account-manager/internal/profile"
	"github.com/tsai41/claude-account-manager/internal/usage"
)

type tickMsg time.Time
type oauthRefetchMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}
type oauthUsageMsg struct {
	profile string
	u       usage.OAuthUsage
	err     error
}
type oauthBatchDoneMsg struct {
	results []oauthUsageMsg
}

func oauthTickCmd(d time.Duration) tea.Cmd {
	if d <= 0 {
		d = 5 * time.Minute
	}
	return tea.Tick(d, func(t time.Time) tea.Msg { return oauthRefetchMsg(t) })
}

type tabID int

const (
	tabProfiles tabID = iota
	tabCosts
	tabActivity
	tabHistory
	tabConfig
)

var tabNames = []string{"Profiles", "Costs", "Activity", "History", "Config"}

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
	fetchingOAuth bool
	width, height int
	bodyVP        viewport.Model
	settings      config.Settings
	configCursor  int
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
	m := Model{settings: config.Load()}
	if err := m.reload(); err != nil {
		return m, err
	}
	m.loadHistory()
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

	mode := m.settings.EffectiveUsageDisplay()
	sessionTitle, weeklyTitle := "Session Left", "Weekly Left"
	if mode == config.DisplayUsed {
		sessionTitle, weeklyTitle = "Session Used", "Weekly Used"
	}
	cols := []table.Column{
		{Title: "", Width: 2},
		{Title: "Name", Width: 14},
		{Title: "Email", Width: 28},
		{Title: sessionTitle, Width: 13},
		{Title: weeklyTitle, Width: 12},
		{Title: "Reset (S/W)", Width: 18},
		{Title: "Updated", Width: 10},
	}
	rows := make([]table.Row, 0, len(profs))
	for _, p := range profs {
		mark := " "
		if p.Name == st.CurrentProfile {
			mark = "*"
		}
		u, _ := usage.Load(p.Name)
		session := usage.Render(u.Session, mode)
		weekly := usage.Render(u.Weekly, mode)
		if isStale(u.SessionResetsAt, u.UpdatedAt, 5*time.Hour) {
			session = "--"
		}
		if isStale(u.WeeklyResetsAt, u.UpdatedAt, 7*24*time.Hour) {
			weekly = "--"
		}
		leftMode := mode != config.DisplayUsed
		session = colourUsage(session, leftMode)
		weekly = colourUsage(weekly, leftMode)
		email := format.MaskEmail(p.Email)
		if email == "" {
			email = "--"
		}
		reset := usage.FormatResetPair(u.SessionResetsAt, u.WeeklyResetsAt)
		updated := "--"
		if !u.UpdatedAt.IsZero() {
			updated = relTime(u.UpdatedAt)
		}
		rows = append(rows, table.Row{mark, p.Name, email, session, weekly, reset, updated})
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeightFor(len(rows))),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("237")).
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

func (m Model) Init() tea.Cmd {
	m.fetchingOAuth = true
	return tea.Batch(oauthTickCmd(m.settings.RefetchInterval()), m.refetchAllOAuthCmd(), tickCmd())
}

// refetchAllOAuthCmd fetches OAuth usage for every profile sequentially in one
// command, spacing requests by settings.FetchSpacing to avoid rate-limiter hits.
func (m Model) refetchAllOAuthCmd() tea.Cmd {
	profs, err := profile.List()
	if err != nil {
		return nil
	}
	current := m.current
	spacing := m.settings.FetchSpacing()
	names := make([]string, 0, len(profs))
	for _, p := range profs {
		names = append(names, p.Name)
	}
	if len(names) == 0 {
		return nil
	}
	return func() tea.Msg {
		results := make([]oauthUsageMsg, 0, len(names))
		for i, name := range names {
			if i > 0 {
				time.Sleep(spacing)
			}
			results = append(results, fetchOAuthOnce(name, current))
		}
		return oauthBatchDoneMsg{results: results}
	}
}

// fetchOAuthOnce performs a single profile fetch synchronously.
func fetchOAuthOnce(profileName, current string) oauthUsageMsg {
	var tokenJSON string
	var err error
	if profileName == current {
		tokenJSON, err = keychain.ReadLive()
		if err != nil {
			tokenJSON, err = keychain.ReadBackup(profileName)
		}
	} else {
		tokenJSON, err = keychain.ReadBackup(profileName)
	}
	if err != nil {
		return oauthUsageMsg{profile: profileName, err: fmt.Errorf("keychain: %w", err)}
	}
	if tokenJSON == "" {
		return oauthUsageMsg{profile: profileName, err: errors.New("keychain: empty token")}
	}
	access := keychain.ExtractAccessToken(tokenJSON)
	if access == "" {
		return oauthUsageMsg{profile: profileName, err: errors.New("no accessToken in token JSON")}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	u, err := usage.FetchOAuthUsage(ctx, access)
	return oauthUsageMsg{profile: profileName, u: u, err: err}
}

// colourUsage applies a colour to a rendered usage string based on remaining %.
// s is the already-rendered string (e.g. "58%" for left mode or "42%" for used mode).
// leftMode true means s represents remaining; false means s represents consumed.
func colourUsage(s string, leftMode bool) string {
	if s == "--" || s == "??" || s == "" || s == "unknown" {
		return s
	}
	trimmed := strings.TrimSuffix(s, "%")
	if trimmed == s {
		return s
	}
	n, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return s
	}
	remaining := n
	if !leftMode {
		remaining = 100 - n
	}
	var col string
	switch {
	case remaining <= 20:
		col = "196"
	case remaining <= 50:
		col = "214"
	default:
		if leftMode {
			col = "46"
		} else {
			col = "231"
		}
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(col)).Render(s)
}

// isStale reports whether a usage value should be considered out-of-date.
// Prefers an explicit reset deadline; falls back to UpdatedAt + window when
// reset is unknown (e.g. manual entries without OAuth data).
func isStale(reset, updated time.Time, window time.Duration) bool {
	now := time.Now()
	if !reset.IsZero() {
		return now.After(reset)
	}
	if updated.IsZero() {
		return false
	}
	return now.After(updated.Add(window))
}

// tableHeightFor returns rows+header capped at 4..14 so the Profiles table
// matches its content instead of stretching down with empty rows.
func tableHeightFor(rows int) int {
	h := rows + 1
	if h < 4 {
		return 4
	}
	if h > 14 {
		return 14
	}
	return h
}

func relTime(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmtInt(int(d/time.Minute)) + "m ago"
	}
	if d < 24*time.Hour {
		return fmtInt(int(d/time.Hour)) + "h ago"
	}
	return fmtInt(int(d/(24*time.Hour))) + "d ago"
}

func fmtInt(n int) string {
	if n < 0 {
		n = 0
	}
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
