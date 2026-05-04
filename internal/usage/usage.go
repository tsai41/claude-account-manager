package usage

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/tsai41/claude-account-manager/internal/paths"
)

type Field struct {
	Display string `json:"display"`
	Source  string `json:"source"`
}

type Record struct {
	Provider      string    `json:"provider"`
	Session       Field     `json:"session"`
	Weekly        Field     `json:"weekly"`
	Manual        string    `json:"manual,omitempty"`
	Note          string    `json:"note,omitempty"`
	ActivityToday int       `json:"activity_today,omitempty"`
	Activity7d    int       `json:"activity_7d,omitempty"`
	Activity5h    int       `json:"activity_5h,omitempty"`
	LastActive    time.Time `json:"last_active,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func Empty() Record {
	return Record{
		Provider:  "manual",
		Session:   Field{Display: "unknown", Source: "none"},
		Weekly:    Field{Display: "unknown", Source: "none"},
		UpdatedAt: time.Now(),
	}
}

func Load(profile string) (Record, error) {
	b, err := os.ReadFile(paths.ProfileUsage(profile))
	if err != nil {
		if os.IsNotExist(err) {
			return Empty(), nil
		}
		return Record{}, err
	}
	var r Record
	if err := json.Unmarshal(b, &r); err != nil {
		return Record{}, err
	}
	return r, nil
}

func Save(profile string, r Record) error {
	r.UpdatedAt = time.Now()
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(paths.ProfileUsage(profile), b, 0o600)
}

var (
	sessionRe = regexp.MustCompile(`(?i)session[^0-9-]*([0-9]+(?:\.[0-9]+)?%?)`)
	weeklyRe  = regexp.MustCompile(`(?i)weekly[^0-9-]*([0-9]+(?:\.[0-9]+)?%?)`)
	pctRe     = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)%$`)
)

// Remaining converts a "consumed" percentage display ("42%") into a remaining one ("58%").
// Non-percent inputs (or the literal "unknown"/"--"/"") return "--".
func Remaining(consumed string) string {
	if consumed == "" || consumed == "unknown" || consumed == "--" {
		return "--"
	}
	m := pctRe.FindStringSubmatch(consumed)
	if len(m) < 2 {
		return consumed
	}
	var n float64
	for _, c := range m[1] {
		_ = c
	}
	if _, err := fmtSscan(m[1], &n); err != nil {
		return consumed
	}
	left := 100 - n
	if left < 0 {
		left = 0
	}
	if left == float64(int(left)) {
		return formatInt(int(left)) + "%"
	}
	return formatFloat(left) + "%"
}

// ParseManual extracts "session X%" and "weekly Y%" tokens from a free-form value.
// Returns ("", "") if neither label is present.
func ParseManual(value string) (session, weekly string) {
	if m := sessionRe.FindStringSubmatch(value); len(m) > 1 {
		session = m[1]
	}
	if m := weeklyRe.FindStringSubmatch(value); len(m) > 1 {
		weekly = m[1]
	}
	return
}

func SetManual(profile, value string) error {
	r, err := Load(profile)
	if err != nil {
		return err
	}
	r.Provider = "manual"
	r.Manual = strings.TrimSpace(value)
	session, weekly := ParseManual(value)
	if session != "" {
		r.Session = Field{Display: session, Source: "manual"}
	} else {
		r.Session = Field{Display: r.Manual, Source: "manual"}
	}
	if weekly != "" {
		r.Weekly = Field{Display: weekly, Source: "manual"}
	}
	return Save(profile, r)
}

func SetNote(profile, note string) error {
	r, err := Load(profile)
	if err != nil {
		return err
	}
	r.Note = note
	return Save(profile, r)
}

func SetProvider(profile, provider string) error {
	r, err := Load(profile)
	if err != nil {
		return err
	}
	r.Provider = provider
	return Save(profile, r)
}
