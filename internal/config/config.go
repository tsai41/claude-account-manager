package config

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/tsai41/claude-account-manager/internal/paths"
)

// Settings is the on-disk user-tunable runtime config. Stored at
// ~/.ccm/config.json. Fields default to safe values when absent.
type Settings struct {
	UsageDisplay        string `json:"usage_display"`         // "left" or "used"
	RefetchSeconds      int    `json:"refetch_seconds"`       // OAuth usage refetch interval
	FetchSpacingSeconds int    `json:"fetch_spacing_seconds"` // gap between per-profile fetches
}

const (
	DisplayLeft = "left"
	DisplayUsed = "used"
)

// Defaults returns the built-in settings used when no config file exists.
func Defaults() Settings {
	return Settings{
		UsageDisplay:        DisplayLeft,
		RefetchSeconds:      300,
		FetchSpacingSeconds: 3,
	}
}

// Load reads ~/.ccm/config.json, falling back to defaults on any error so the
// TUI always has a usable Settings to render.
func Load() Settings {
	s := Defaults()
	b, err := os.ReadFile(paths.ConfigFile())
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s)
	s.normalize()
	return s
}

// Save writes the settings as pretty JSON to ~/.ccm/config.json.
func Save(s Settings) error {
	s.normalize()
	if err := paths.EnsureRoot(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(paths.ConfigFile(), b, 0o600)
}

func (s *Settings) normalize() {
	switch strings.ToLower(s.UsageDisplay) {
	case DisplayUsed:
		s.UsageDisplay = DisplayUsed
	default:
		s.UsageDisplay = DisplayLeft
	}
	if s.RefetchSeconds < 60 {
		s.RefetchSeconds = 60
	}
	if s.RefetchSeconds > 3600 {
		s.RefetchSeconds = 3600
	}
	if s.FetchSpacingSeconds < 1 {
		s.FetchSpacingSeconds = 1
	}
	if s.FetchSpacingSeconds > 30 {
		s.FetchSpacingSeconds = 30
	}
}

// RefetchInterval returns the refetch cadence as time.Duration.
func (s Settings) RefetchInterval() time.Duration {
	return time.Duration(s.RefetchSeconds) * time.Second
}

// FetchSpacing returns the per-profile spacing as time.Duration.
func (s Settings) FetchSpacing() time.Duration {
	return time.Duration(s.FetchSpacingSeconds) * time.Second
}

// EffectiveUsageDisplay applies the CCM_USAGE_DISPLAY env override on top of
// the persisted setting so existing env-based scripts keep working.
func (s Settings) EffectiveUsageDisplay() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("CCM_USAGE_DISPLAY"))) {
	case DisplayUsed:
		return DisplayUsed
	case DisplayLeft:
		return DisplayLeft
	}
	return s.UsageDisplay
}
