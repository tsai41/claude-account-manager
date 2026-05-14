package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/dirmap"
	"github.com/tsai41/claude-account-manager/internal/profile"
)

func newBindCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bind <profile> <dir>",
		Short: "Bind a directory (or its subtree) to a profile for shell `chpwd` routing",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, dir := args[0], args[1]
			if err := profile.ValidateName(name); err != nil {
				return err
			}
			if !profile.Exists(name) {
				return fmt.Errorf("profile %q not found", name)
			}
			m, err := dirmap.Load()
			if err != nil {
				return err
			}
			if err := m.Bind(name, dir); err != nil {
				return err
			}
			if err := m.Save(); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Bound %s -> %s\n", m.Bindings[len(m.Bindings)-1].Pattern, name)
			return nil
		},
	}
}

func newUnbindCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unbind <dir>",
		Short: "Remove a directory binding",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := dirmap.Load()
			if err != nil {
				return err
			}
			if !m.Unbind(args[0]) {
				return fmt.Errorf("no binding for %q", args[0])
			}
			if err := m.Save(); err != nil {
				return err
			}
			fmt.Println("Unbound.")
			return nil
		},
	}
}

func newBindingsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bindings",
		Short: "List directory -> profile bindings (in match priority order)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := dirmap.Load()
			if err != nil {
				return err
			}
			if len(m.Bindings) == 0 {
				fmt.Println("(no bindings)")
				return nil
			}
			fmt.Printf("%-10s  %s\n", "PROFILE", "PATTERN")
			for _, b := range m.Bindings {
				fmt.Printf("%-10s  %s\n", b.Profile, b.Pattern)
			}
			return nil
		},
	}
}
