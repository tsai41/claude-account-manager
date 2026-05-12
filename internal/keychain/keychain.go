package keychain

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"os/user"
	"strings"
)

const (
	LiveService   = "Claude Code-credentials"
	BackupService = "com.ccm.tokens"
)

var ErrNotFound = errors.New("keychain item not found")

func currentUser() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.Username, nil
}

// LiveAccount returns the account name (OS username) used by Claude CLI's keychain entry.
func LiveAccount() (string, error) { return currentUser() }

// Read returns password for given service+account.
func Read(service, account string) (string, error) {
	cmd := exec.Command("security", "find-generic-password", "-s", service, "-a", account, "-w")
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		s := errb.String() + out.String()
		if strings.Contains(s, "could not be found") || strings.Contains(s, "SecKeychainSearchCopyNext") {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("security read %s/%s: %w (%s)", service, account, err, strings.TrimSpace(s))
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

// Write upserts password.
func Write(service, account, password, label string) error {
	if label == "" {
		label = service
	}
	cmd := exec.Command("security", "add-generic-password",
		"-U",
		"-s", service,
		"-a", account,
		"-l", label,
		"-w", password,
	)
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("security write %s/%s: %w (%s)", service, account, err, strings.TrimSpace(errb.String()))
	}
	return nil
}

// Delete removes an item. Returns nil if already absent.
func Delete(service, account string) error {
	cmd := exec.Command("security", "delete-generic-password", "-s", service, "-a", account)
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		s := errb.String()
		if strings.Contains(s, "could not be found") {
			return nil
		}
		return fmt.Errorf("security delete %s/%s: %w (%s)", service, account, err, strings.TrimSpace(s))
	}
	return nil
}

// ReadLive returns the live Claude CLI token JSON.
func ReadLive() (string, error) {
	acct, err := LiveAccount()
	if err != nil {
		return "", err
	}
	return Read(LiveService, acct)
}

// WriteLive overwrites the live Claude CLI token JSON.
func WriteLive(token string) error {
	acct, err := LiveAccount()
	if err != nil {
		return err
	}
	return Write(LiveService, acct, token, "Claude Code-credentials")
}

// ReadBackup returns ccm-managed backup token for profile.
func ReadBackup(profile string) (string, error) {
	return Read(BackupService, profile)
}

// WriteBackup stores ccm-managed backup token for profile.
func WriteBackup(profile, token string) error {
	return Write(BackupService, profile, token, "ccm backup: "+profile)
}

// DeleteBackup removes the ccm-managed backup for profile.
func DeleteBackup(profile string) error {
	return Delete(BackupService, profile)
}

// ExtractAccessToken pulls the OAuth accessToken from a token JSON blob.
// Accepts both wrapped ({"claudeAiOauth":{"accessToken":...}}) and flat
// ({"accessToken":...}) shapes. Returns "" when not found.
func ExtractAccessToken(tokenJSON string) string {
	const key = "\"accessToken\""
	i := strings.Index(tokenJSON, key)
	if i < 0 {
		return ""
	}
	rest := tokenJSON[i+len(key):]
	q1 := strings.Index(rest, "\"")
	if q1 < 0 {
		return ""
	}
	rest = rest[q1+1:]
	q2 := strings.Index(rest, "\"")
	if q2 <= 0 {
		return ""
	}
	return rest[:q2]
}

// Fingerprint returns the last 8 chars of accessToken from a token JSON, or "" if unavailable.
// This is a cheap heuristic mirroring CCSwitcher's diagnostic approach; we do NOT log the full token.
func Fingerprint(tokenJSON string) string {
	tok := ExtractAccessToken(tokenJSON)
	if len(tok) < 8 {
		return ""
	}
	return tok[len(tok)-8:]
}
