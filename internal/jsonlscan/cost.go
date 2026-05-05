package jsonlscan

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tsai41/claude-account-manager/internal/paths"
)

// Pricing in USD per 1M tokens. Cache-creation tokens are billed at 1.25x base input
// for 5-minute TTL or 2x for 1-hour; cache-read at 0.1x. Defaults below mirror the
// public list price for the Claude 4 family.
type Pricing struct {
	InputPerM         float64 `json:"input_per_m"`
	OutputPerM        float64 `json:"output_per_m"`
	CacheCreate5mMult float64 `json:"cache_create_5m_mult"`
	CacheCreate1hMult float64 `json:"cache_create_1h_mult"`
	CacheReadMult     float64 `json:"cache_read_mult"`
}

func (p Pricing) Cost(t Tokens) float64 {
	in := float64(t.Input) * p.InputPerM / 1e6
	out := float64(t.Output) * p.OutputPerM / 1e6
	cc5 := float64(t.CacheCreate5m) * p.InputPerM * p.CacheCreate5mMult / 1e6
	cc1 := float64(t.CacheCreate1h) * p.InputPerM * p.CacheCreate1hMult / 1e6
	cr := float64(t.CacheRead) * p.InputPerM * p.CacheReadMult / 1e6
	return in + out + cc5 + cc1 + cr
}

// DefaultPricing maps Claude 4.x model id substrings to a Pricing record.
// Numbers reflect the public list price for the Claude 4.5/4.6 family
// (Opus $5/$25 per M, Sonnet $3/$15 per M, Haiku $1/$5 per M); cache write at
// 1.25x base input, cache read at 0.1x base input.
var DefaultPricing = []struct {
	Match   string
	Family  string
	Pricing Pricing
}{
	{"opus", "Opus", Pricing{5, 25, 1.25, 2.0, 0.1}},
	{"sonnet", "Sonnet", Pricing{3, 15, 1.25, 2.0, 0.1}},
	{"haiku", "Haiku", Pricing{1, 5, 1.25, 2.0, 0.1}},
}

func familyOf(model string) (family string, p Pricing, ok bool) {
	low := strings.ToLower(model)
	for _, e := range DefaultPricing {
		if strings.Contains(low, e.Match) {
			return e.Family, e.Pricing, true
		}
	}
	return "", Pricing{}, false
}

// Tokens aggregates token counts across messages.
type Tokens struct {
	Input         int64 `json:"input"`
	Output        int64 `json:"output"`
	CacheCreate5m int64 `json:"cache_create_5m"`
	CacheCreate1h int64 `json:"cache_create_1h"`
	CacheRead     int64 `json:"cache_read"`
}

func (t *Tokens) add(o Tokens) {
	t.Input += o.Input
	t.Output += o.Output
	t.CacheCreate5m += o.CacheCreate5m
	t.CacheCreate1h += o.CacheCreate1h
	t.CacheRead += o.CacheRead
}

func (t Tokens) Total() int64 {
	return t.Input + t.Output + t.CacheCreate5m + t.CacheCreate1h + t.CacheRead
}

// CostBucket is per-model usage in a given time window.
type CostBucket struct {
	Model  string  `json:"model"`
	Family string  `json:"family"`
	Turns  int     `json:"turns"`
	Tokens Tokens  `json:"tokens"`
	Cost   float64 `json:"cost"`
}

// CostReport summarises one window.
type CostReport struct {
	Window      string       `json:"window"` // "today", "7d", "30d"
	Turns       int          `json:"turns"`
	Tokens      Tokens       `json:"tokens"`
	Cost        float64      `json:"cost"`
	Sessions    int          `json:"sessions"`
	ActiveDur   time.Duration `json:"active_duration"`
	ByFamily    []CostBucket `json:"by_family"`
	DailyTotals []DailyTotal `json:"daily_totals"`
	LastActive  time.Time    `json:"last_active"`
}

type DailyTotal struct {
	Date     string  `json:"date"`
	Turns    int     `json:"turns"`
	Cost     float64 `json:"cost"`
	Families []string `json:"families"`
}

// CostStats holds today/7d/30d reports.
type CostStats struct {
	Today CostReport `json:"today"`
	Last7 CostReport `json:"last_7"`
	Last30 CostReport `json:"last_30"`
}

type assistantMsg struct {
	Type        string `json:"type"`
	Timestamp   string `json:"timestamp"`
	SessionID   string `json:"sessionId"`
	IsSidechain bool   `json:"isSidechain"`
	RequestID   string `json:"requestId"`
	Message     struct {
		Model string `json:"model"`
		Usage struct {
			Input         int64 `json:"input_tokens"`
			Output        int64 `json:"output_tokens"`
			CacheCreate   int64 `json:"cache_creation_input_tokens"`
			CacheRead     int64 `json:"cache_read_input_tokens"`
			CacheCreation struct {
				Ephemeral5m int64 `json:"ephemeral_5m_input_tokens"`
				Ephemeral1h int64 `json:"ephemeral_1h_input_tokens"`
			} `json:"cache_creation"`
		} `json:"usage"`
	} `json:"message"`
}

// IncludeSidechainEnv is the env var to include Task-tool sub-agent messages.
// Default behaviour excludes them so totals align with CCSwitcher and a single
// "main thread" view of activity.
const IncludeSidechainEnv = "CCM_INCLUDE_SIDECHAIN"

// ScanCosts walks ~/.claude/projects/**/*.jsonl and aggregates token usage and
// estimated cost over today, last 7 days, and last 30 days. The result is
// machine-wide; jsonl transcripts are not bound to a Claude account. By default
// sub-agent (Task tool) messages with isSidechain=true are excluded; set
// CCM_INCLUDE_SIDECHAIN=1 to include them.
func ScanCosts() (CostStats, error) {
	includeSidechain := os.Getenv(IncludeSidechainEnv) == "1"
	return ScanCostsWith(includeSidechain)
}

// ScanCostsWith aggregates with explicit sidechain inclusion control.
func ScanCostsWith(includeSidechain bool) (CostStats, error) {
	overrides, _ := LoadPricingOverrides()
	seenReq := map[string]struct{}{}
	var cs CostStats
	root := filepath.Join(paths.ClaudeDir(), "projects")
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return cs, nil
		}
		return cs, err
	}

	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekCutoff := now.Add(-7 * 24 * time.Hour)
	monthCutoff := now.Add(-30 * 24 * time.Hour)

	type bucket struct {
		turns      int
		tokensFam  map[string]*CostBucket
		tokens     Tokens
		cost       float64
		sessions   map[string]struct{}
		lastActive time.Time
		stamps     []time.Time
	}
	newBucket := func() *bucket {
		return &bucket{
			tokensFam: map[string]*CostBucket{},
			sessions:  map[string]struct{}{},
		}
	}
	today := newBucket()
	week := newBucket()
	month := newBucket()
	dailyAgg := map[string]*DailyTotal{} // YYYY-MM-DD → totals (last 30 days)
	dailyFamilies := map[string]map[string]struct{}{}

	apply := func(b *bucket, t time.Time, model, family string, tk Tokens, cost float64, sessID string) {
		b.turns++
		b.tokens.add(tk)
		b.cost += cost
		if sessID != "" {
			b.sessions[sessID] = struct{}{}
		}
		if t.After(b.lastActive) {
			b.lastActive = t
		}
		b.stamps = append(b.stamps, t)
		if family == "" {
			family = "Other"
		}
		key := family
		row, ok := b.tokensFam[key]
		if !ok {
			row = &CostBucket{Family: family, Model: model}
			b.tokensFam[key] = row
		}
		row.Turns++
		row.Tokens.add(tk)
		row.Cost += cost
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		buf := make([]byte, 0, 1<<20)
		sc.Buffer(buf, 16<<20)
		for sc.Scan() {
			line := sc.Bytes()
			if len(line) == 0 {
				continue
			}
			var m assistantMsg
			if err := json.Unmarshal(line, &m); err != nil {
				continue
			}
			if m.Type != "assistant" || m.Message.Model == "" {
				continue
			}
			if m.IsSidechain && !includeSidechain {
				continue
			}
			if m.RequestID != "" {
				if _, dup := seenReq[m.RequestID]; dup {
					continue
				}
				seenReq[m.RequestID] = struct{}{}
			}
			t, err := time.Parse(time.RFC3339Nano, m.Timestamp)
			if err != nil || t.Before(monthCutoff) {
				continue
			}
			tk := Tokens{
				Input:     m.Message.Usage.Input,
				Output:    m.Message.Usage.Output,
				CacheRead: m.Message.Usage.CacheRead,
			}
			// Prefer detailed split when present; fall back to total cache_creation_input_tokens treated as 5m.
			if m.Message.Usage.CacheCreation.Ephemeral5m > 0 || m.Message.Usage.CacheCreation.Ephemeral1h > 0 {
				tk.CacheCreate5m = m.Message.Usage.CacheCreation.Ephemeral5m
				tk.CacheCreate1h = m.Message.Usage.CacheCreation.Ephemeral1h
			} else {
				tk.CacheCreate5m = m.Message.Usage.CacheCreate
			}
			family, pricing, ok := resolvedPricing(m.Message.Model, overrides)
			cost := 0.0
			if ok {
				cost = pricing.Cost(tk)
			}
			apply(month, t, m.Message.Model, family, tk, cost, m.SessionID)
			if t.After(weekCutoff) {
				apply(week, t, m.Message.Model, family, tk, cost, m.SessionID)
			}
			if !t.Before(dayStart) {
				apply(today, t, m.Message.Model, family, tk, cost, m.SessionID)
			}
			day := t.Format("2006-01-02")
			row := dailyAgg[day]
			if row == nil {
				row = &DailyTotal{Date: day}
				dailyAgg[day] = row
				dailyFamilies[day] = map[string]struct{}{}
			}
			row.Turns++
			row.Cost += cost
			if family != "" {
				dailyFamilies[day][family] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		return cs, err
	}

	finalize := func(label string, b *bucket) CostReport {
		var bucketsSlice []CostBucket
		for _, v := range b.tokensFam {
			bucketsSlice = append(bucketsSlice, *v)
		}
		sort.Slice(bucketsSlice, func(i, j int) bool { return bucketsSlice[i].Cost > bucketsSlice[j].Cost })
		return CostReport{
			Window:     label,
			Turns:      b.turns,
			Tokens:     b.tokens,
			Cost:       b.cost,
			Sessions:   len(b.sessions),
			ActiveDur:  activeDuration(b.stamps),
			ByFamily:   bucketsSlice,
			LastActive: b.lastActive,
		}
	}
	cs.Today = finalize("today", today)
	cs.Last7 = finalize("7d", week)
	cs.Last30 = finalize("30d", month)

	// daily totals: include 30 days only (sorted descending).
	var dates []string
	for d := range dailyAgg {
		dates = append(dates, d)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	for _, d := range dates {
		row := dailyAgg[d]
		fams := dailyFamilies[d]
		for f := range fams {
			row.Families = append(row.Families, f)
		}
		sort.Strings(row.Families)
		cs.Last30.DailyTotals = append(cs.Last30.DailyTotals, *row)
	}
	return cs, nil
}

// activeDuration sums gaps between consecutive timestamps that are < 5 minutes apart,
// approximating active engagement time.
func activeDuration(stamps []time.Time) time.Duration {
	if len(stamps) < 2 {
		return 0
	}
	sort.Slice(stamps, func(i, j int) bool { return stamps[i].Before(stamps[j]) })
	const gap = 5 * time.Minute
	var total time.Duration
	for i := 1; i < len(stamps); i++ {
		d := stamps[i].Sub(stamps[i-1])
		if d > 0 && d <= gap {
			total += d
		}
	}
	return total
}
