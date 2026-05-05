package safemerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMergeFromSnapshot_ReplacesAuthKeysOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	live := map[string]any{
		"oauthAccount": map[string]any{
			"emailAddress": "old@example.com",
			"accountUuid":  "old-uuid",
		},
		"theme":          "dark",
		"mcpServers":     map[string]any{"local": "preserved"},
		"customSettings": "preserved",
	}
	writeJSON(t, filepath.Join(home, ".claude.json"), live)

	snapshotDir := t.TempDir()
	snap := map[string]any{
		"oauthAccount": map[string]any{
			"emailAddress": "new@example.com",
			"accountUuid":  "new-uuid",
		},
		"userID":        "user-123",
		"theme":         "light",     // would-be auth-irrelevant override; should NOT replace live theme
		"customSettings": "ignored",  // not in authKeys; should NOT replace
	}
	writeJSON(t, filepath.Join(snapshotDir, "claude.json"), snap)

	replaced, err := MergeFromSnapshot(snapshotDir)
	if err != nil {
		t.Fatalf("MergeFromSnapshot: %v", err)
	}
	if !contains(replaced, "oauthAccount") || !contains(replaced, "userID") {
		t.Fatalf("replaced keys = %v, expected oauthAccount and userID", replaced)
	}

	got := readJSON(t, filepath.Join(home, ".claude.json"))

	oa, ok := got["oauthAccount"].(map[string]any)
	if !ok {
		t.Fatalf("oauthAccount missing or wrong type: %#v", got["oauthAccount"])
	}
	if oa["emailAddress"] != "new@example.com" {
		t.Errorf("oauthAccount.emailAddress = %v, want new@example.com", oa["emailAddress"])
	}
	if oa["accountUuid"] != "new-uuid" {
		t.Errorf("oauthAccount.accountUuid = %v, want new-uuid", oa["accountUuid"])
	}

	if got["userID"] != "user-123" {
		t.Errorf("userID = %v, want user-123 (snapshot value)", got["userID"])
	}

	// Non-auth keys must be preserved from live, NOT overwritten by snapshot.
	if got["theme"] != "dark" {
		t.Errorf("theme = %v, want preserved 'dark' (snapshot 'light' should not stomp)", got["theme"])
	}
	if got["customSettings"] != "preserved" {
		t.Errorf("customSettings = %v, want preserved", got["customSettings"])
	}
	if mcp, ok := got["mcpServers"].(map[string]any); !ok || mcp["local"] != "preserved" {
		t.Errorf("mcpServers preserved check failed: %#v", got["mcpServers"])
	}
}

func TestMergeFromSnapshot_MissingSnapshotKeyDoesNotDeleteLive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	live := map[string]any{
		"oauthAccount": map[string]any{"emailAddress": "old@example.com"},
		"userID":       "preserve-me",
	}
	writeJSON(t, filepath.Join(home, ".claude.json"), live)

	snapshotDir := t.TempDir()
	snap := map[string]any{
		"oauthAccount": map[string]any{"emailAddress": "new@example.com"},
		// no userID in snapshot
	}
	writeJSON(t, filepath.Join(snapshotDir, "claude.json"), snap)

	if _, err := MergeFromSnapshot(snapshotDir); err != nil {
		t.Fatalf("MergeFromSnapshot: %v", err)
	}
	got := readJSON(t, filepath.Join(home, ".claude.json"))
	if got["userID"] != "preserve-me" {
		t.Errorf("userID = %v, want 'preserve-me' (snapshot lacked the key, must not delete)", got["userID"])
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
