package snapshot

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// EnsureConfigDir creates ~/.ccm/configs/<profile>/ if missing, and seeds its
// .claude.json from the profile's latest snapshot when the file is absent.
// The seed copies the oauthAccount metadata Claude Code's interactive flow
// needs to skip /login when CLAUDE_CODE_OAUTH_TOKEN is present.
//
// Idempotent: existing .claude.json is left untouched on subsequent calls so
// per-profile preferences accumulated through usage are preserved.
func EnsureConfigDir(configDir, profileName string) error {
	if configDir == "" {
		return errors.New("configDir required")
	}
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}
	dst := filepath.Join(configDir, ".claude.json")
	if _, err := os.Stat(dst); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	snapDir, err := Latest(profileName)
	if err != nil || snapDir == "" {
		return nil // no snapshot to seed from; let claude bootstrap fresh
	}
	src := filepath.Join(snapDir, "claude.json")
	in, err := os.Open(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
