package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tsai41/claude-account-manager/internal/paths"
)

const (
	logFile    = "ccm.log"
	maxLogSize = 5 * 1024 * 1024 // 5MB rotate threshold
)

var mu sync.Mutex

type Entry struct {
	Time    time.Time      `json:"time"`
	Level   string         `json:"level"`
	Event   string         `json:"event"`
	Profile string         `json:"profile,omitempty"`
	Message string         `json:"message,omitempty"`
	Fields  map[string]any `json:"fields,omitempty"`
}

func write(level, event, profile, message string, fields map[string]any) {
	if err := paths.EnsureRoot(); err != nil {
		return
	}
	if err := os.MkdirAll(paths.LogsDir(), 0o700); err != nil {
		return
	}
	e := Entry{
		Time:    time.Now(),
		Level:   level,
		Event:   event,
		Profile: profile,
		Message: message,
		Fields:  scrub(fields),
	}
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	path := filepath.Join(paths.LogsDir(), logFile)
	if st, err := os.Stat(path); err == nil && st.Size() >= maxLogSize {
		_ = os.Rename(path, path+".1")
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(b)
	f.Write([]byte("\n"))
}

// scrub removes obvious token-like fields. Caller should never put raw tokens here,
// but as defense-in-depth we drop any key whose name looks token-bearing.
func scrub(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		lk := k
		switch lk {
		case "token", "access_token", "accessToken", "refresh_token", "refreshToken", "password", "credential":
			out[k] = "[redacted]"
			continue
		}
		// also redact long string values that look like Anthropic OAuth tokens
		if s, ok := v.(string); ok && (len(s) > 60 && (hasPrefix(s, "sk-ant-oat") || hasPrefix(s, "sk-ant-ort"))) {
			out[k] = "[redacted]"
			continue
		}
		out[k] = v
	}
	return out
}

func hasPrefix(s, p string) bool {
	if len(s) < len(p) {
		return false
	}
	return s[:len(p)] == p
}

func Info(event, profile, message string, fields map[string]any) {
	write("info", event, profile, message, fields)
}

func Warn(event, profile, message string, fields map[string]any) {
	write("warn", event, profile, message, fields)
}

func Error(event, profile, message string, fields map[string]any) {
	write("error", event, profile, message, fields)
}

func LogPath() string { return filepath.Join(paths.LogsDir(), logFile) }

// Tail returns the last n entries (decoded), oldest-first. Returns nil if file missing.
func Tail(n int) ([]Entry, error) {
	if n <= 0 {
		n = 50
	}
	f, err := os.Open(LogPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	// simple full-read; log is meant to be small. Acceptable for v1.
	stat, _ := f.Stat()
	if stat.Size() == 0 {
		return nil, nil
	}
	buf := make([]byte, stat.Size())
	if _, err := f.Read(buf); err != nil {
		return nil, err
	}
	var entries []Entry
	start := 0
	for i := 0; i < len(buf); i++ {
		if buf[i] == '\n' {
			line := buf[start:i]
			start = i + 1
			if len(line) == 0 {
				continue
			}
			var e Entry
			if json.Unmarshal(line, &e) == nil {
				entries = append(entries, e)
			}
		}
	}
	if len(entries) > n {
		entries = entries[len(entries)-n:]
	}
	return entries, nil
}

// FormatEntry returns a one-line human-readable rendering.
func FormatEntry(e Entry) string {
	base := fmt.Sprintf("%s %-5s %-18s", e.Time.Format("2006-01-02 15:04:05"), e.Level, e.Event)
	if e.Profile != "" {
		base += " " + e.Profile
	}
	if e.Message != "" {
		base += " — " + e.Message
	}
	return base
}
