package jsonlscan

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/tsai41/claude-account-manager/internal/paths"
)

// Activity is a derived-from-jsonl summary of local Claude Code usage.
// It reflects machine-wide activity (jsonl transcripts have no account binding)
// and is explicitly NOT a substitute for the official usage bar.
type Activity struct {
	Today      int       `json:"today"`
	Last7Days  int       `json:"last_7_days"`
	Last5Hours int       `json:"last_5_hours"`
	LastActive time.Time `json:"last_active"`
	Sessions   int       `json:"sessions_today"`
}

type minLine struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	SessionID string `json:"sessionId"`
}

// Scan walks ~/.claude/projects/**/*.jsonl and aggregates conversation turns.
// A "turn" is any line whose type is user or assistant.
func Scan() (Activity, error) {
	var a Activity
	root := filepath.Join(paths.ClaudeDir(), "projects")
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return a, nil
		}
		return a, err
	}
	now := time.Now()
	dayCutoff := now.Add(-24 * time.Hour)
	weekCutoff := now.Add(-7 * 24 * time.Hour)
	hourCutoff := now.Add(-5 * time.Hour)
	todaySessions := map[string]struct{}{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable
		}
		if info.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		buf := make([]byte, 0, 1<<20)
		sc.Buffer(buf, 16<<20) // some lines are large (tool results)
		for sc.Scan() {
			line := sc.Bytes()
			if len(line) == 0 {
				continue
			}
			var ml minLine
			if err := json.Unmarshal(line, &ml); err != nil {
				continue
			}
			if ml.Type != "user" && ml.Type != "assistant" {
				continue
			}
			if ml.Timestamp == "" {
				continue
			}
			t, err := time.Parse(time.RFC3339Nano, ml.Timestamp)
			if err != nil {
				continue
			}
			if t.After(a.LastActive) {
				a.LastActive = t
			}
			if t.After(weekCutoff) {
				a.Last7Days++
			}
			if t.After(dayCutoff) {
				a.Today++
				if ml.SessionID != "" {
					todaySessions[ml.SessionID] = struct{}{}
				}
			}
			if t.After(hourCutoff) {
				a.Last5Hours++
			}
		}
		return nil
	})
	a.Sessions = len(todaySessions)
	return a, err
}
