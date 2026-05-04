package cli

import (
	"github.com/spf13/cobra"
)

func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "ccm",
		Short: "Claude Code OAuth account state manager",
		Long:  "ccm manages local Claude Code OAuth account profiles. All data stays on this machine.",
		// Default action: show current
		RunE: runCurrent,
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
		newDoctorCmd(),
		newTuiCmd(),
	)
	return root
}
