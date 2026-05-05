package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/profile"
	"github.com/tsai41/claude-account-manager/internal/usage"
)

func newListCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			if asJSON {
				return runListJSON(cmd)
			}
			return runList(cmd, args)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

func runListJSON(cmd *cobra.Command) error {
	profs, err := profile.List()
	if err != nil {
		return err
	}
	st, _ := profile.LoadState()
	type row struct {
		Name        string `json:"name"`
		Current     bool   `json:"current"`
		Email       string `json:"email,omitempty"`
		AuthType    string `json:"auth_type"`
		SessionLeft string `json:"session_left,omitempty"`
		WeeklyLeft  string `json:"weekly_left,omitempty"`
		LastUsedAt  string `json:"last_used_at,omitempty"`
		Note        string `json:"note,omitempty"`
	}
	out := make([]row, 0, len(profs))
	for _, p := range profs {
		u, _ := usage.Load(p.Name)
		r := row{
			Name:        p.Name,
			Current:     p.Name == st.CurrentProfile,
			Email:       p.Email,
			AuthType:    p.AuthType,
			SessionLeft: usage.Remaining(u.Session.Display),
			WeeklyLeft:  usage.Remaining(u.Weekly.Display),
			Note:        u.Note,
		}
		if !p.LastUsedAt.IsZero() {
			r.LastUsedAt = p.LastUsedAt.Format(time.RFC3339)
		}
		out = append(out, r)
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(out)
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
