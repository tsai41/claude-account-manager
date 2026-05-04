package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/profile"
	"github.com/tsai41/claude-account-manager/internal/usage"
)

func newUsageProviderCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "usage-provider <name> <manual|local-derived>",
		Short:     "Set usage provider for a profile",
		Args:      cobra.ExactArgs(2),
		ValidArgs: []string{"manual", "local-derived"},
		RunE: func(cmd *cobra.Command, args []string) error {
			name, prov := args[0], args[1]
			if !profile.Exists(name) {
				return fmt.Errorf("profile %q not found", name)
			}
			if prov != "manual" && prov != "local-derived" {
				return fmt.Errorf("provider must be one of: manual, local-derived")
			}
			if err := usage.SetProvider(name, prov); err != nil {
				return err
			}
			fmt.Printf("Usage provider for %s set to %s\n", name, prov)
			if prov == "local-derived" {
				fmt.Println("Note: local-derived counts machine-wide jsonl turns; not an official usage bar.")
			}
			return nil
		},
	}
}
