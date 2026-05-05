package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

func displayOrDash(s string) string {
	if s == "" || s == "unknown" {
		return "--"
	}
	return s
}

func jsonEncoder(cmd *cobra.Command) *json.Encoder {
	e := json.NewEncoder(cmd.OutOrStdout())
	e.SetIndent("", "  ")
	return e
}
