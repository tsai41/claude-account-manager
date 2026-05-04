package switcher

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/tsai41/claude-account-manager/internal/claudeauth"
	"github.com/tsai41/claude-account-manager/internal/keychain"
	"github.com/tsai41/claude-account-manager/internal/logger"
	"github.com/tsai41/claude-account-manager/internal/profile"
	"github.com/tsai41/claude-account-manager/internal/safemerge"
	"github.com/tsai41/claude-account-manager/internal/snapshot"
)

// Strategy controls how a switch applies the target snapshot to live state.
type Strategy int

const (
	// StrategySafeMerge replaces only auth-sensitive keys in ~/.claude.json and the keychain LIVE entry.
	// All other live state (~/.claude/ contents, non-auth settings) is preserved.
	StrategySafeMerge Strategy = iota
	// StrategyFullRestore overwrites ~/.claude.json entirely and replaces ~/.claude/ contents from the snapshot tar.
	StrategyFullRestore
)

// Result describes the outcome of a successful Switch.
type Result struct {
	Profile      profile.Profile
	BackupDir    string
	TokenFP      string
	LiveEmail    string
	EmailMatches bool
	Strategy     Strategy
	MergedKeys   []string // populated when Strategy == StrategySafeMerge
}

// Switch performs the default (safe-merge) switch to the named profile.
func Switch(name string) (Result, error) {
	return SwitchWith(name, StrategySafeMerge)
}

// SwitchWith performs a switch using the given strategy.
func SwitchWith(name string, strategy Strategy) (Result, error) {
	target, err := profile.Load(name)
	if err != nil {
		logger.Error("switch", name, "profile load failed", map[string]any{"err": err.Error()})
		return Result{}, err
	}
	logger.Info("switch.start", name, "switching profile", nil)
	st, _ := profile.LoadState()

	liveTok, liveErr := keychain.ReadLive()
	if liveErr != nil && !errors.Is(liveErr, keychain.ErrNotFound) {
		return Result{}, fmt.Errorf("read live keychain: %w", liveErr)
	}
	bkDir, err := snapshot.BackupCurrent("pre-switch-to-"+name, liveTok)
	if err != nil {
		return Result{}, fmt.Errorf("safety backup: %w", err)
	}

	// Refresh source profile snapshot if email matches its profile.
	if st.CurrentProfile != "" && st.CurrentProfile != name && liveErr == nil {
		if srcProf, perr := profile.Load(st.CurrentProfile); perr == nil {
			meta, _ := claudeauth.ReadAccountMeta()
			if srcProf.Email != "" && meta.Email == srcProf.Email {
				if snap, serr := snapshot.Create(srcProf.Name, liveTok); serr == nil {
					srcProf.SnapshotID = snap.ID
					srcProf.LastUsedAt = time.Now()
					_ = profile.Save(srcProf)
					_ = keychain.WriteBackup(srcProf.Name, liveTok)
				}
			}
		}
	}

	snapDir, err := snapshot.Latest(name)
	if err != nil {
		return Result{}, err
	}
	if snapDir == "" {
		return Result{}, fmt.Errorf("profile %s has no snapshot", name)
	}
	var (
		tokenFromSnap string
		mergedKeys    []string
	)
	switch strategy {
	case StrategyFullRestore:
		tokenFromSnap, err = snapshot.Restore(snapDir)
		if err != nil {
			return Result{}, fmt.Errorf("restore snapshot: %w (safety backup at %s)", err, bkDir)
		}
	case StrategySafeMerge:
		mergedKeys, err = safemerge.MergeFromSnapshot(snapDir)
		if err != nil {
			return Result{}, fmt.Errorf("safe-merge snapshot: %w (safety backup at %s)", err, bkDir)
		}
		// also pick up token from snapshot file if present
		if b, rerr := os.ReadFile(snapDir + "/keychain-credential.json"); rerr == nil {
			tokenFromSnap = string(b)
		}
	default:
		return Result{}, fmt.Errorf("unknown switch strategy")
	}

	token := tokenFromSnap
	if t, kerr := keychain.ReadBackup(name); kerr == nil && t != "" {
		token = t
	}
	if token == "" {
		return Result{}, fmt.Errorf("no keychain token for profile %s (safety backup at %s)", name, bkDir)
	}
	if err := keychain.WriteLive(token); err != nil {
		return Result{}, fmt.Errorf("write live keychain: %w (safety backup at %s)", err, bkDir)
	}

	meta, _ := claudeauth.ReadAccountMeta()
	target.LastUsedAt = time.Now()
	_ = profile.Save(target)
	if err := profile.SaveState(profile.State{CurrentProfile: name, LastSwitchAt: time.Now()}); err != nil {
		return Result{}, err
	}

	res := Result{
		Profile:      target,
		BackupDir:    bkDir,
		TokenFP:      keychain.Fingerprint(token),
		LiveEmail:    meta.Email,
		EmailMatches: target.Email == "" || meta.Email == "" || meta.Email == target.Email,
		Strategy:     strategy,
		MergedKeys:   mergedKeys,
	}
	logger.Info("switch.done", name, "switch complete", map[string]any{
		"backup_dir":    bkDir,
		"token_fp":      res.TokenFP,
		"live_email":    meta.Email,
		"email_matches": res.EmailMatches,
	})
	if !res.EmailMatches {
		logger.Warn("switch.email_mismatch", name, "post-switch email differs from profile email",
			map[string]any{"profile_email": target.Email, "live_email": meta.Email})
	}
	return res, nil
}

// Remove deletes a profile and its keychain backup. If the profile was current, clears state.
func Remove(name string, keepKeychain bool) error {
	if !profile.Exists(name) {
		return fmt.Errorf("profile %q not found", name)
	}
	if err := profile.Delete(name); err != nil {
		logger.Error("remove", name, "profile delete failed", map[string]any{"err": err.Error()})
		return err
	}
	if !keepKeychain {
		_ = keychain.DeleteBackup(name)
	}
	st, _ := profile.LoadState()
	if st.CurrentProfile == name {
		st.CurrentProfile = ""
		_ = profile.SaveState(st)
	}
	logger.Info("remove", name, "profile removed", map[string]any{"keep_keychain": keepKeychain})
	return nil
}
