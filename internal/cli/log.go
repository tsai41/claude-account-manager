package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/logger"
)

func newLogCmd() *cobra.Command {
	var n int
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show recent ccm log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := logger.Tail(n)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Println("(no log entries)")
				fmt.Printf("Log file: %s\n", logger.LogPath())
				return nil
			}
			for _, e := range entries {
				fmt.Println(logger.FormatEntry(e))
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&n, "lines", "n", 50, "max entries to show (most recent)")
	return cmd
}
