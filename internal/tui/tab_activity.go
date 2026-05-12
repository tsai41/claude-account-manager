package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/tsai41/claude-account-manager/internal/format"
)

func activityStat(label, value string, width int) string {
	v := cardValue.Render(value)
	l := dimStyle.Render(label)
	return lipgloss.NewStyle().Width(width).Render(v + "\n" + l)
}

func (m Model) viewActivity() string {
	if m.statsErr != nil {
		return errStyle.Render("scan error: " + m.statsErr.Error())
	}
	if m.stats.LastActive.IsZero() {
		if m.statsLoading {
			return dimStyle.Render("Loading activity… (scanning ~/.claude/projects/*.jsonl)")
		}
		return dimStyle.Render("No activity yet — use Claude Code so transcripts land in ~/.claude/projects/.")
	}
	s := m.stats
	c := m.costs
	var b strings.Builder

	todayLines := []string{
		cardLabel.Render(fmt.Sprintf("Today's Activity  ·  %s", time.Now().Format("Mon Jan 2"))),
		lipgloss.JoinHorizontal(lipgloss.Top,
			activityStat("Turns", fmt.Sprintf("%d", s.Today), 14),
			activityStat("Active", format.HumanDuration(c.Today.ActiveDur), 14),
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
			bar := format.BarInt(d.Turns, max, 24)
			date := strings.Replace(d.Date[5:], "-", "/", 1)
			cost := fmt.Sprintf("  $%.2f", d.Cost)
			line := fmt.Sprintf("  %s   %5d turns  %s%s", date, d.Turns, bar, cost)
			if d.Date == today {
				line = todayRow.Render(fmt.Sprintf("▶ %s   %5d turns  ", date, d.Turns)) + bar + todayRow.Render(cost)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Turn = user or assistant message in jsonl. Active = sum of <5min gaps."))
	return b.String()
}
