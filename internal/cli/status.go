package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/keychain"
	"github.com/tsai41/claude-account-manager/internal/profile"
	"github.com/tsai41/claude-account-manager/internal/snapshot"
	"github.com/tsai41/claude-account-manager/internal/usage"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [name]",
		Short: "Show profile status and usage",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				st, _ := profile.LoadState()
				name = st.CurrentProfile
			}
			if name == "" {
				return fmt.Errorf("no profile selected; pass a name or run `ccm use <name>`")
			}
			return runStatus(name)
		},
	}
}

func runStatus(name string) error {
	p, err := profile.Load(name)
	if err != nil {
		return err
	}
	u, _ := usage.Load(name)
	snapDir, _ := snapshot.Latest(name)
	bkTok, bkErr := keychain.ReadBackup(name)

	fmt.Printf("Profile: %s\n", p.Name)
	fmt.Printf("Auth: %s\n", p.AuthType)
	if p.Email != "" {
		fmt.Printf("Email: %s\n", p.Email)
	}
	if p.OrgName != "" {
		fmt.Printf("Org: %s\n", p.OrgName)
	}
	fmt.Printf("Created: %s\n", p.CreatedAt.Format("2006-01-02 15:04:05"))
	if !p.LastUsedAt.IsZero() {
		fmt.Printf("Last used: %s\n", p.LastUsedAt.Format("2006-01-02 15:04:05"))
	}
	if snapDir != "" {
		fmt.Printf("Latest snapshot: %s\n", snapDir)
	} else {
		fmt.Println("Latest snapshot: (none)")
	}
	if bkErr == nil {
		fmt.Printf("Keychain backup fp: %s\n", keychain.Fingerprint(bkTok))
	} else {
		fmt.Printf("Keychain backup: missing (%v)\n", bkErr)
	}
	if u.Manual != "" {
		fmt.Printf("Usage: %s (manual)\n", u.Manual)
	} else {
		fmt.Printf("Usage: session %s, weekly %s\n", u.Session.Display, u.Weekly.Display)
	}
	if u.Note != "" {
		fmt.Printf("Note: %s\n", u.Note)
	}
	return nil
}
