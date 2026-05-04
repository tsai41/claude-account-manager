package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/switcher"
)

func newRemoveCmd() *cobra.Command {
	var keepKeychain bool
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := switcher.Remove(args[0], keepKeychain); err != nil {
				return err
			}
			fmt.Printf("Removed profile: %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&keepKeychain, "keep-keychain-backup", false, "keep ccm-managed keychain backup item")
	return cmd
}
