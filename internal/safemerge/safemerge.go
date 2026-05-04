// Package safemerge implements the safe-merge switch strategy: only auth-sensitive
// keys in ~/.claude.json are replaced with values from the target profile's snapshot.
// All other keys (theme, MCP, user preferences) are preserved.
package safemerge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tsai41/claude-account-manager/internal/paths"
)

// authKeys are the top-level fields in ~/.claude.json that we treat as auth-sensitive.
// Anything not in this set is preserved from the live file.
var authKeys = []string{
	"oauthAccount",
	"userID",
	"organizations",
	"primaryApiKey",
	"hasCompletedOnboarding",
}

// MergeFromSnapshot reads the snapshot's claude.json, copies the auth-sensitive keys
// into the live ~/.claude.json (preserving everything else), and writes it back.
// Returns the merged set of auth keys actually replaced.
func MergeFromSnapshot(snapshotDir string) ([]string, error) {
	snapPath := filepath.Join(snapshotDir, "claude.json")
	snapBytes, err := os.ReadFile(snapPath)
	if err != nil {
		return nil, fmt.Errorf("read snapshot claude.json: %w", err)
	}
	var snap map[string]any
	if err := json.Unmarshal(snapBytes, &snap); err != nil {
		return nil, fmt.Errorf("parse snapshot claude.json: %w", err)
	}

	livePath := paths.ClaudeJSON()
	live := map[string]any{}
	if b, err := os.ReadFile(livePath); err == nil {
		_ = json.Unmarshal(b, &live)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read live claude.json: %w", err)
	}

	var replaced []string
	for _, k := range authKeys {
		if v, ok := snap[k]; ok {
			live[k] = v
			replaced = append(replaced, k)
		} else {
			// snapshot didn't have this key — leave live untouched (don't delete)
			// (alternative: delete live[k]; we choose not to, to avoid data loss.)
		}
	}

	out, err := json.MarshalIndent(live, "", "  ")
	if err != nil {
		return nil, err
	}
	tmp := livePath + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, livePath); err != nil {
		return nil, err
	}
	return replaced, nil
}
