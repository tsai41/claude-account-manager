package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/claudeauth"
	"github.com/tsai41/claude-account-manager/internal/keychain"
	"github.com/tsai41/claude-account-manager/internal/profile"
	"github.com/tsai41/claude-account-manager/internal/usage"
)

func newCurrentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current profile",
		RunE:  runCurrent,
	}
}

func runCurrent(cmd *cobra.Command, args []string) error {
	st, err := profile.LoadState()
	if err != nil {
		return err
	}
	meta, _ := claudeauth.ReadAccountMeta()
	liveTok, liveErr := keychain.ReadLive()
	liveFP := ""
	if liveErr == nil {
		liveFP = keychain.Fingerprint(liveTok)
	}

	if st.CurrentProfile == "" {
		fmt.Println("Current: (none — no profile selected)")
		if meta.Email != "" {
			fmt.Printf("Live email: %s\n", meta.Email)
		}
		if liveFP != "" {
			fmt.Printf("Live token fp: %s\n", liveFP)
		}
		if liveErr != nil {
			fmt.Printf("Warning: cannot read live keychain (%v)\n", liveErr)
		}
		fmt.Println("Hint: run `ccm import-current <name>` to capture the current login.")
		return nil
	}

	p, err := profile.Load(st.CurrentProfile)
	if err != nil {
		fmt.Printf("Current: %s (metadata missing: %v)\n", st.CurrentProfile, err)
		return nil
	}
	u, _ := usage.Load(p.Name)

	fmt.Printf("Current: %s\n", p.Name)
	fmt.Printf("Auth: %s\n", p.AuthType)
	if p.Email != "" {
		fmt.Printf("Email: %s\n", p.Email)
	}
	if meta.Email != "" && meta.Email != p.Email {
		fmt.Printf("Warning: live email %s differs from profile email %s (state desync)\n", meta.Email, p.Email)
	}
	if liveFP != "" {
		fmt.Printf("Live token fp: %s\n", liveFP)
	}
	if u.Manual != "" {
		fmt.Printf("Usage: %s (manual)\n", u.Manual)
	} else {
		fmt.Printf("Usage: session %s, weekly %s\n", u.Session.Display, u.Weekly.Display)
	}
	if !p.LastUsedAt.IsZero() {
		fmt.Printf("Last used: %s\n", p.LastUsedAt.Format("2006-01-02 15:04:05"))
	}
	if u.Note != "" {
		fmt.Printf("Note: %s\n", u.Note)
	}
	return nil
}
