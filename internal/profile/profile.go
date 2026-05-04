package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/tsai41/claude-account-manager/internal/paths"
)

type Profile struct {
	Name          string    `json:"name"`
	AuthType      string    `json:"auth_type"`
	Email         string    `json:"email,omitempty"`
	AccountUUID   string    `json:"account_uuid,omitempty"`
	OrgName       string    `json:"org_name,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	LastUsedAt    time.Time `json:"last_used_at,omitempty"`
	SnapshotID    string    `json:"snapshot_id,omitempty"`
	UsageProvider string    `json:"usage_provider,omitempty"`
	Note          string    `json:"note,omitempty"`
}

type State struct {
	CurrentProfile string    `json:"current_profile"`
	LastSwitchAt   time.Time `json:"last_switch_at,omitempty"`
}

var ErrNotFound = errors.New("profile not found")
var ErrExists = errors.New("profile already exists")

func List() ([]Profile, error) {
	if err := paths.EnsureRoot(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(paths.ProfilesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Profile
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p, err := Load(e.Name())
		if err != nil {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func Load(name string) (Profile, error) {
	var p Profile
	b, err := os.ReadFile(paths.ProfileMetadata(name))
	if err != nil {
		if os.IsNotExist(err) {
			return p, ErrNotFound
		}
		return p, err
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return p, fmt.Errorf("parse profile %s: %w", name, err)
	}
	return p, nil
}

func Save(p Profile) error {
	if err := os.MkdirAll(paths.ProfileDir(p.Name), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(paths.ProfileMetadata(p.Name), b, 0o600)
}

func Exists(name string) bool {
	_, err := os.Stat(paths.ProfileMetadata(name))
	return err == nil
}

func Delete(name string) error {
	if !Exists(name) {
		return ErrNotFound
	}
	return os.RemoveAll(paths.ProfileDir(name))
}

func LoadState() (State, error) {
	var s State
	b, err := os.ReadFile(paths.StateFile())
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, err
	}
	return s, nil
}

func SaveState(s State) error {
	if err := paths.EnsureRoot(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := paths.StateFile() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, paths.StateFile())
}

// ValidateName ensures profile name is filesystem-safe.
func ValidateName(name string) error {
	if name == "" {
		return errors.New("profile name required")
	}
	if name != filepath.Base(name) || name == "." || name == ".." {
		return errors.New("invalid profile name")
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return fmt.Errorf("invalid character %q in profile name", r)
		}
	}
	return nil
}
