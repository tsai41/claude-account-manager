package cli

import (
	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/tui"
)

func newTuiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Interactive TUI for browsing and switching profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run()
		},
	}
}
