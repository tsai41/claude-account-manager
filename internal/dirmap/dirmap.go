// Package dirmap manages the directory-to-profile mapping used by `ccm bind`
// and the shell `chpwd` integration. The file lives at ~/.ccm/dir-map.json.
//
// Matching semantics: a binding's Pattern matches a target path when the path
// is the pattern itself or sits underneath it (prefix match honoring path
// separators). Longest-pattern-wins, ties broken by insertion order.
package dirmap

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tsai41/claude-account-manager/internal/paths"
)

// Binding pairs a profile name with an absolute directory pattern.
type Binding struct {
	Profile string `json:"profile"`
	Pattern string `json:"pattern"`
}

// Map is the on-disk dir-map.json structure.
type Map struct {
	Bindings []Binding `json:"bindings"`
}

// Path returns the dir-map.json location.
func Path() string { return filepath.Join(paths.CCMRoot(), "dir-map.json") }

// Load reads the dir-map. A missing file returns an empty Map (not an error).
func Load() (Map, error) {
	var m Map
	b, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return m, err
	}
	if len(b) == 0 {
		return m, nil
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("parse %s: %w", Path(), err)
	}
	return m, nil
}

// Save writes the dir-map atomically.
func (m Map) Save() error {
	if err := paths.EnsureRoot(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := Path() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, Path())
}

// Resolve returns the profile bound to dir, or "" if none.
func (m Map) Resolve(dir string) string {
	target := canonicalize(dir)
	if target == "" {
		return ""
	}
	bestLen := -1
	var bestProfile string
	for _, b := range m.Bindings {
		if !patternMatches(b.Pattern, target) {
			continue
		}
		if len(b.Pattern) > bestLen {
			bestLen = len(b.Pattern)
			bestProfile = b.Profile
		}
	}
	return bestProfile
}

// Bind adds or replaces a binding for the (canonicalized) pattern.
func (m *Map) Bind(profile, pattern string) error {
	if profile == "" {
		return errors.New("profile required")
	}
	pattern = canonicalize(pattern)
	if pattern == "" {
		return errors.New("pattern required")
	}
	for i := range m.Bindings {
		if m.Bindings[i].Pattern == pattern {
			m.Bindings[i].Profile = profile
			return nil
		}
	}
	m.Bindings = append(m.Bindings, Binding{Profile: profile, Pattern: pattern})
	return nil
}

// Unbind removes the binding for pattern. Returns false if not found.
func (m *Map) Unbind(pattern string) bool {
	pattern = canonicalize(pattern)
	for i, b := range m.Bindings {
		if b.Pattern == pattern {
			m.Bindings = append(m.Bindings[:i], m.Bindings[i+1:]...)
			return true
		}
	}
	return false
}

// canonicalize expands ~, makes absolute, cleans separators.
func canonicalize(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if p == "~" {
		p = paths.Home()
	} else if strings.HasPrefix(p, "~/") {
		p = filepath.Join(paths.Home(), p[2:])
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	return filepath.Clean(abs)
}

func patternMatches(pattern, target string) bool {
	if pattern == target {
		return true
	}
	sep := string(filepath.Separator)
	if !strings.HasSuffix(pattern, sep) {
		pattern += sep
	}
	return strings.HasPrefix(target+sep, pattern)
}
