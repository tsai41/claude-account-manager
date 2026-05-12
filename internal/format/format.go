// Package format collects formatting helpers shared by the CLI and TUI.
package format

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// MaskEmailEnv toggles email masking across CLI/TUI output (for screenshots).
const MaskEmailEnv = "CCM_MASK_EMAIL"

// HumanTokens renders a token count as 1.2K / 3.4M / 5.6B with one decimal place.
func HumanTokens(n int64) string {
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

// HumanDuration renders a duration as "0m" / "12m" / "1h 5m".
func HumanDuration(d time.Duration) string {
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

// Bar returns a fixed-width bar of █ characters proportional to v/max.
func Bar(v, max float64, width int) string {
	if max <= 0 {
		return strings.Repeat(" ", width)
	}
	n := int((v / max) * float64(width))
	if n < 0 {
		n = 0
	}
	if n > width {
		n = width
	}
	return strings.Repeat("█", n) + strings.Repeat(" ", width-n)
}

// BarInt is a convenience wrapper for integer scales.
func BarInt(v, max, width int) string { return Bar(float64(v), float64(max), width) }

// MaskEmail returns a redacted version of an email when CCM_MASK_EMAIL=1.
func MaskEmail(email string) string {
	if os.Getenv(MaskEmailEnv) != "1" || email == "" {
		return email
	}
	at := strings.IndexByte(email, '@')
	if at < 1 {
		return "***"
	}
	first := string(email[0])
	dot := strings.LastIndexByte(email[at:], '.')
	tld := ""
	if dot >= 0 && at+dot < len(email)-1 {
		tld = email[at+dot:]
	}
	return first + "***@***" + tld
}

// PadRight pads s with spaces on the right to width w.
func PadRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

// PadLeft pads s with spaces on the left to width w.
func PadLeft(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return strings.Repeat(" ", w-len(s)) + s
}

// RelTime returns a human-readable relative time string for t.
func RelTime(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return FmtInt(int(d/time.Minute)) + "m ago"
	}
	if d < 24*time.Hour {
		return FmtInt(int(d/time.Hour)) + "h ago"
	}
	return FmtInt(int(d/(24*time.Hour))) + "d ago"
}

// FmtInt converts a non-negative integer to its decimal string representation.
func FmtInt(n int) string {
	if n <= 0 {
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
