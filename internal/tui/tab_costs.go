package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/tsai41/claude-account-manager/internal/format"
	"github.com/tsai41/claude-account-manager/internal/jsonlscan"
)

func (m Model) viewCosts() string {
	if m.costsErr != nil {
		return errStyle.Render("scan error: " + m.costsErr.Error())
	}
	if m.costs.Today.Window == "" {
		if m.costsLoading {
			return dimStyle.Render("Loading costs… (scanning ~/.claude/projects/*.jsonl)")
		}
		return dimStyle.Render("No cost data yet — open Costs tab to load.")
	}
	c := m.costs
	if c.Last30.Turns == 0 {
		return dimStyle.Render("No assistant messages found in ~/.claude/projects/*.jsonl in the last 30 days.\nUse Claude Code at least once to populate the transcript history.")
	}
	var b strings.Builder

	todayLines := []string{
		cardLabel.Render(fmt.Sprintf("Today's API-equivalent cost  ·  %s", time.Now().Format("Mon Jan 2"))),
		costAmountStyle.Render(fmt.Sprintf("$%.2f", c.Today.Cost)),
	}
	if len(c.Today.ByFamily) > 0 {
		todayLines = append(todayLines, dimStyle.Render(strings.Repeat("─", 38)))
		for _, fb := range c.Today.ByFamily {
			todayLines = append(todayLines,
				familyColor(fb.Family).Render(format.PadRight(fb.Family, 10))+
					dimStyle.Render(format.PadLeft(format.HumanTokens(fb.Tokens.Total())+" tok", 18))+
					cardValue.Render(fmt.Sprintf("  $%.2f", fb.Cost)))
		}
	}
	footer := fmt.Sprintf("%d sessions  ·  %s tokens  ·  active %s",
		c.Today.Sessions, format.HumanTokens(c.Today.Tokens.Total()), format.HumanDuration(c.Today.ActiveDur))
	todayLines = append(todayLines, dimStyle.Render(strings.Repeat("─", 38)), dimStyle.Render(footer))
	b.WriteString(cardStyle.Width(46).Render(strings.Join(todayLines, "\n")))
	b.WriteString("\n\n")

	mkCard := func(label string, r jsonlscan.CostReport) string {
		lines := []string{
			cardLabel.Render(label),
			cardValue.Render(fmt.Sprintf("$%.2f", r.Cost)),
			dimStyle.Render(fmt.Sprintf("%d turns · %s tok", r.Turns, format.HumanTokens(r.Tokens.Total()))),
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
			bar := format.Bar(d.Cost, max, 20)
			date := strings.Replace(d.Date[5:], "-", "/", 1)
			fams := strings.Join(d.Families, ",")
			line := fmt.Sprintf("  %s   $%-8.2f  %s  %s", date, d.Cost, bar, dimStyle.Render(fams))
			if d.Date == today {
				line = todayRow.Render(fmt.Sprintf("· %s   $%-8.2f  ", date, d.Cost)) + bar + "  " + dimStyle.Render(fams)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString(helpStyle.Render("List price · Opus $5/$25 · Sonnet $3/$15 · Haiku $1/$5 per 1M (in/out) · Not an invoice"))
	return b.String()
}
