package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/switcher"
)

func newUseCmd() *cobra.Command {
	var fullRestore bool
	cmd := &cobra.Command{
		Use:   "use <name>",
		Short: "Switch to a profile (default: safe-merge)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			strategy := switcher.StrategySafeMerge
			if fullRestore {
				strategy = switcher.StrategyFullRestore
			}
			res, err := switcher.SwitchWith(args[0], strategy)
			if err != nil {
				return err
			}
			fmt.Printf("Switched to: %s\n", res.Profile.Name)
			switch res.Strategy {
			case switcher.StrategyFullRestore:
				fmt.Println("Strategy: full-restore (claude.json and ~/.claude/ replaced)")
			case switcher.StrategySafeMerge:
				fmt.Printf("Strategy: safe-merge (auth keys: %v)\n", res.MergedKeys)
			}
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
	cmd.Flags().BoolVar(&fullRestore, "full-restore", false, "use full-restore strategy (overwrite claude.json and ~/.claude/ from snapshot)")
	return cmd
}
