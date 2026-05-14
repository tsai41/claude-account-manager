package snapshot

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// OAuthCredential is the structure stored inside keychain-credential.json snapshots.
// Mirrors the Keychain entry Claude Code writes for subscription OAuth state.
type OAuthCredential struct {
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken"`
	ExpiresAt        int64  `json:"expiresAt"` // unix ms
	Scopes           any    `json:"scopes,omitempty"`
	SubscriptionType string `json:"subscriptionType,omitempty"`
	RateLimitTier    string `json:"rateLimitTier,omitempty"`
}

type keychainFile struct {
	ClaudeAiOAuth OAuthCredential `json:"claudeAiOauth"`
}

// ErrNoToken indicates the profile has no usable OAuth credential snapshot on disk.
var ErrNoToken = errors.New("no oauth credential snapshot found for profile")

// LoadLatestOAuth reads the most recent snapshot's keychain-credential.json for the given
// profile and returns its parsed OAuth credential.
func LoadLatestOAuth(profileName string) (OAuthCredential, string, error) {
	snapDir, err := Latest(profileName)
	if err != nil {
		return OAuthCredential{}, "", err
	}
	if snapDir == "" {
		return OAuthCredential{}, "", ErrNoToken
	}
	tokPath := filepath.Join(snapDir, "keychain-credential.json")
	b, err := os.ReadFile(tokPath)
	if err != nil {
		if os.IsNotExist(err) {
			return OAuthCredential{}, "", ErrNoToken
		}
		return OAuthCredential{}, "", fmt.Errorf("read %s: %w", tokPath, err)
	}
	var kc keychainFile
	if err := json.Unmarshal(b, &kc); err != nil {
		return OAuthCredential{}, "", fmt.Errorf("parse %s: %w", tokPath, err)
	}
	if kc.ClaudeAiOAuth.AccessToken == "" {
		return OAuthCredential{}, "", ErrNoToken
	}
	return kc.ClaudeAiOAuth, snapDir, nil
}

// Expired reports whether the credential's accessToken has passed its expiresAt.
func (c OAuthCredential) Expired() bool {
	if c.ExpiresAt == 0 {
		return false
	}
	return time.Now().After(c.ExpiresUnix())
}

// ExpiresUnix returns the expiry as a time.Time.
func (c OAuthCredential) ExpiresUnix() time.Time {
	return time.UnixMilli(c.ExpiresAt)
}
