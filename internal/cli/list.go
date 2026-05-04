package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/profile"
	"github.com/tsai41/claude-account-manager/internal/usage"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List profiles",
		RunE:  runList,
	}
}

func runList(cmd *cobra.Command, args []string) error {
	profs, err := profile.List()
	if err != nil {
		return err
	}
	st, _ := profile.LoadState()
	if len(profs) == 0 {
		fmt.Println("No profiles. Run `ccm import-current <name>` to add one.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CURRENT\tNAME\tEMAIL\tAUTH\tSESSION LEFT\tWEEKLY LEFT\tLAST USED")
	for _, p := range profs {
		mark := ""
		if p.Name == st.CurrentProfile {
			mark = "*"
		}
		u, _ := usage.Load(p.Name)
		session := usage.Remaining(u.Session.Display)
		weekly := usage.Remaining(u.Weekly.Display)
		last := "--"
		if !p.LastUsedAt.IsZero() {
			last = p.LastUsedAt.Format("2006-01-02 15:04")
		}
		email := p.Email
		if email == "" {
			email = "--"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", mark, p.Name, email, p.AuthType, session, weekly, last)
	}
	return w.Flush()
}
