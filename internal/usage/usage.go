package usage

import (
	"encoding/json"
	"os"
	"time"

	"github.com/tsai41/claude-account-manager/internal/paths"
)

type Field struct {
	Display string `json:"display"`
	Source  string `json:"source"`
}

type Record struct {
	Provider  string    `json:"provider"`
	Session   Field     `json:"session"`
	Weekly    Field     `json:"weekly"`
	Manual    string    `json:"manual,omitempty"`
	Note      string    `json:"note,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
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

func SetManual(profile, value string) error {
	r, err := Load(profile)
	if err != nil {
		return err
	}
	r.Provider = "manual"
	r.Manual = value
	r.Session.Display = value
	r.Session.Source = "manual"
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
