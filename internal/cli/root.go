package cli

import (
	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/profile"
	"github.com/tsai41/claude-account-manager/internal/tui"
)

func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "ccm",
		Short: "Claude Code OAuth account state manager",
		Long:  "ccm manages local Claude Code OAuth account profiles. All data stays on this machine.",
		// Default action: launch TUI when profiles exist, else show current/bootstrap hint.
		RunE: func(cmd *cobra.Command, args []string) error {
			profs, _ := profile.List()
			if len(profs) == 0 {
				return runCurrent(cmd, args)
			}
			return tui.Run()
		},
		SilenceUsage: true,
	}
	root.AddCommand(
		newCurrentCmd(),
		newListCmd(),
		newImportCurrentCmd(),
		newUseCmd(),
		newRemoveCmd(),
		newStatusCmd(),
		newUsageCmd(),
		newUsageSetCmd(),
		newUsageNoteCmd(),
		newUsageProviderCmd(),
		newDoctorCmd(),
		newTuiCmd(),
		newLogCmd(),
		newRollbackCmd(),
		newVersionCmd(),
		newCostCmd(),
	)
	return root
}
