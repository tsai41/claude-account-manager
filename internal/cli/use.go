package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/switcher"
)

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Switch to a profile (full-restore)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := switcher.Switch(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Switched to: %s\n", res.Profile.Name)
			if res.LiveEmail != "" {
				fmt.Printf("Email: %s\n", res.LiveEmail)
			}
			if !res.EmailMatches {
				fmt.Printf("Warning: post-switch email %s != target %s\n", res.LiveEmail, res.Profile.Email)
			}
			fmt.Printf("Token fp: %s\n", res.TokenFP)
			fmt.Printf("Safety backup: %s\n", res.BackupDir)
			fmt.Println("Tip: restart any running Claude Code sessions to pick up the new account.")
			return nil
		},
	}
}
