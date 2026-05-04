package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/claudeauth"
	"github.com/tsai41/claude-account-manager/internal/keychain"
	"github.com/tsai41/claude-account-manager/internal/logger"
	"github.com/tsai41/claude-account-manager/internal/paths"
	"github.com/tsai41/claude-account-manager/internal/profile"
	"github.com/tsai41/claude-account-manager/internal/snapshot"
	"github.com/tsai41/claude-account-manager/internal/usage"
)

func newImportCurrentCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "import-current <name>",
		Short: "Capture the currently logged-in OAuth state as a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImportCurrent(args[0], force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing profile (auto-backup old)")
	return cmd
}

func runImportCurrent(name string, force bool) error {
	if err := profile.ValidateName(name); err != nil {
		return err
	}
	if err := paths.EnsureRoot(); err != nil {
		return err
	}

	if profile.Exists(name) && !force {
		return fmt.Errorf("profile %q already exists (use --force to overwrite)", name)
	}

	// Read live keychain token
	token, err := keychain.ReadLive()
	if err != nil {
		return fmt.Errorf("read live keychain: %w (is Claude Code logged in?)", err)
	}

	// Read account metadata
	meta, err := claudeauth.ReadAccountMeta()
	if err != nil {
		return err
	}
	if meta.Email == "" {
		fmt.Println("Warning: ~/.claude.json has no oauthAccount.emailAddress; profile will be unverified.")
	} else {
		// Duplicate-email detection: warn if another profile already claims this email.
		if profs, lerr := profile.List(); lerr == nil {
			for _, ex := range profs {
				if ex.Name == name {
					continue
				}
				if ex.Email != "" && ex.Email == meta.Email {
					if !force {
						return fmt.Errorf("email %s is already imported as profile %q; use --force to import again under %q",
							meta.Email, ex.Name, name)
					}
					fmt.Printf("Warning: email %s already imported as profile %q (proceeding due to --force)\n", meta.Email, ex.Name)
				}
			}
		}
	}

	// Force overwrite: backup existing profile dir first
	if profile.Exists(name) && force {
		if _, err := snapshot.BackupCurrent("overwrite-"+name, ""); err != nil {
			return fmt.Errorf("backup before overwrite: %w", err)
		}
		if err := profile.Delete(name); err != nil {
			return err
		}
	}

	// Create snapshot for this profile
	snap, err := snapshot.Create(name, token)
	if err != nil {
		return err
	}

	// Store ccm-managed keychain backup
	if err := keychain.WriteBackup(name, token); err != nil {
		return fmt.Errorf("store keychain backup: %w", err)
	}

	// Save profile metadata
	p := profile.Profile{
		Name:          name,
		AuthType:      "oauth",
		Email:         meta.Email,
		AccountUUID:   meta.AccountUUID,
		OrgName:       meta.OrgName,
		CreatedAt:     time.Now(),
		LastUsedAt:    time.Now(),
		SnapshotID:    snap.ID,
		UsageProvider: "manual",
	}
	if err := profile.Save(p); err != nil {
		return err
	}
	if err := usage.Save(p.Name, usage.Empty()); err != nil {
		return err
	}

	// Set as current
	if err := profile.SaveState(profile.State{CurrentProfile: name, LastSwitchAt: time.Now()}); err != nil {
		return err
	}

	fmt.Printf("Imported profile: %s\n", name)
	fmt.Printf("Auth: oauth\n")
	if meta.Email != "" {
		fmt.Printf("Email: %s\n", meta.Email)
	}
	fmt.Printf("Token fp: %s\n", keychain.Fingerprint(token))
	fmt.Printf("Snapshot: %s\n", snap.Dir)
	fmt.Println("Usage: not set — `ccm usage-set` to record manually")
	logger.Info("import-current", name, "profile imported", map[string]any{
		"email":     meta.Email,
		"token_fp":  keychain.Fingerprint(token),
		"snapshot":  snap.ID,
		"force":     force,
	})
	return nil
}
