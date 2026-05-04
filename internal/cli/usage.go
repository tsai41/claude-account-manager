package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/profile"
	"github.com/tsai41/claude-account-manager/internal/usage"
)

func resolveProfileArg(args []string) (string, error) {
	if len(args) > 0 && args[0] != "" {
		return args[0], nil
	}
	st, _ := profile.LoadState()
	if st.CurrentProfile == "" {
		return "", fmt.Errorf("no profile selected; pass a name or run `ccm use <name>`")
	}
	return st.CurrentProfile, nil
}

func newUsageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "usage [name]",
		Short: "Show usage record",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := resolveProfileArg(args)
			if err != nil {
				return err
			}
			if !profile.Exists(name) {
				return fmt.Errorf("profile %q not found", name)
			}
			u, err := usage.LoadAndDerive(name)
			if err != nil {
				return err
			}
			fmt.Printf("Profile: %s\n", name)
			fmt.Printf("Provider: %s\n", u.Provider)
			if u.Manual != "" {
				fmt.Printf("Manual: %s\n", u.Manual)
			}
			fmt.Printf("Session: %s (%s)\n", displayOrDash(u.Session.Display), u.Session.Source)
			fmt.Printf("Weekly: %s (%s)\n", displayOrDash(u.Weekly.Display), u.Weekly.Source)
			if u.Provider == "local-derived" {
				fmt.Printf("Today turns: %d\n", u.ActivityToday)
				fmt.Printf("7-day turns: %d\n", u.Activity7d)
				fmt.Printf("5-hour turns: %d\n", u.Activity5h)
				if !u.LastActive.IsZero() {
					fmt.Printf("Last active: %s\n", u.LastActive.Format("2006-01-02 15:04:05"))
				}
			}
			if u.Note != "" {
				fmt.Printf("Note: %s\n", u.Note)
			}
			fmt.Printf("Updated: %s\n", u.UpdatedAt.Format("2006-01-02 15:04:05"))
			return nil
		},
	}
}

func newUsageSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "usage-set <name> <value>",
		Short: "Set manual usage value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !profile.Exists(args[0]) {
				return fmt.Errorf("profile %q not found", args[0])
			}
			if err := usage.SetManual(args[0], args[1]); err != nil {
				return err
			}
			fmt.Printf("Usage updated for %s: %s\n", args[0], args[1])
			return nil
		},
	}
}

func newUsageNoteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "usage-note <name> <text>",
		Short: "Set usage note",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !profile.Exists(args[0]) {
				return fmt.Errorf("profile %q not found", args[0])
			}
			if err := usage.SetNote(args[0], args[1]); err != nil {
				return err
			}
			fmt.Printf("Note updated for %s\n", args[0])
			return nil
		},
	}
}
