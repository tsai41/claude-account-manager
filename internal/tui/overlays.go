package tui

import (
	"fmt"
	"strings"

	"github.com/tsai41/claude-account-manager/internal/config"
	"github.com/tsai41/claude-account-manager/internal/format"
	"github.com/tsai41/claude-account-manager/internal/keychain"
	"github.com/tsai41/claude-account-manager/internal/profile"
	"github.com/tsai41/claude-account-manager/internal/snapshot"
	"github.com/tsai41/claude-account-manager/internal/usage"
)

func (m Model) viewHelp() string {
	rows := [][2]string{
		{"Navigation", ""},
		{"Tab / →", "next tab"},
		{"Shift+Tab / ←", "prev tab"},
		{"1–5", "jump to Profiles / Costs / Activity / History / Config"},
		{"j / k / ↓ / ↑", "move row"},
		{"", ""},
		{"Profiles tab", ""},
		{"Enter", "switch to profile (Y/n confirm)"},
		{"i", "show profile detail (fp / snapshot / email)"},
		{"e", "edit usage value (parses session/weekly %)"},
		{"u", "edit note"},
		{"d", "delete profile (y/N confirm)"},
		{"r", "reload table from disk"},
		{"R", "refetch OAuth usage now (auto every 5min)"},
		{"", ""},
		{"Costs / Activity / History tabs", ""},
		{"↑/↓", "scroll viewport"},
		{"r", "rescan jsonl transcripts / reload log"},
		{"", ""},
		{"Config tab", ""},
		{"j / k", "move cursor"},
		{"Enter / l / →", "cycle value forward"},
		{"h / ←", "cycle value backward"},
		{"s", "save config to ~/.ccm/config.json"},
		{"r", "reset to defaults"},
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
	row("Email", format.MaskEmail(p.Email))
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
	mode := m.settings.EffectiveUsageDisplay()
	label := "Usage left"
	if mode == config.DisplayUsed {
		label = "Usage used"
	}
	row(label, "session "+usage.Render(u.Session, mode)+"  weekly "+usage.Render(u.Weekly, mode))
	if !u.SessionResetsAt.IsZero() || !u.WeeklyResetsAt.IsZero() {
		row("Resets in", usage.FormatResetPair(u.SessionResetsAt, u.WeeklyResetsAt))
	}
	if !u.SessionResetsAt.IsZero() {
		row("Session reset", u.SessionResetsAt.Local().Format("2006-01-02 15:04:05"))
	}
	if !u.WeeklyResetsAt.IsZero() {
		row("Weekly reset", u.WeeklyResetsAt.Local().Format("2006-01-02 15:04:05"))
	}
	if !u.UpdatedAt.IsZero() {
		row("Usage fetched", u.UpdatedAt.Local().Format("2006-01-02 15:04:05")+"  ("+format.RelTime(u.UpdatedAt)+")")
	}
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
