package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/dirmap"
	"github.com/tsai41/claude-account-manager/internal/paths"
	"github.com/tsai41/claude-account-manager/internal/snapshot"
)

// newDirExportCmd is invoked by the shell `chpwd` hook (see `ccm shell-init`).
// It prints sh-eval-able statements that update CLAUDE_CONFIG_DIR + CLAUDE_CODE_OAUTH_TOKEN
// for the given directory, or unset them when no binding matches.
func newDirExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "dir-export <dir>",
		Short:  "Emit shell `export` / `unset` statements for the given directory (used by shell hook)",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			out := cmd.OutOrStdout()
			m, err := dirmap.Load()
			if err != nil {
				return err
			}
			name := m.Resolve(dir)
			if name == "" {
				fmt.Fprintln(out, "unset CLAUDE_CONFIG_DIR CLAUDE_CODE_OAUTH_TOKEN CCM_ACTIVE_PROFILE")
				return nil
			}
			cred, _, err := snapshot.LoadLatestOAuth(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ccm: %s -> profile %s, but no oauth snapshot: %v\n", dir, name, err)
				fmt.Fprintln(out, "unset CLAUDE_CONFIG_DIR CLAUDE_CODE_OAUTH_TOKEN CCM_ACTIVE_PROFILE")
				return nil
			}
			if cred.Expired() {
				fmt.Fprintf(os.Stderr, "ccm: profile %s access token expired at %s — falling back to global keychain\n",
					name, cred.ExpiresUnix().Format("2006-01-02 15:04:05"))
				fmt.Fprintln(out, "unset CLAUDE_CONFIG_DIR CLAUDE_CODE_OAUTH_TOKEN CCM_ACTIVE_PROFILE")
				return nil
			}
			cfgDir := paths.ProfileConfigDir(name)
			if err := snapshot.EnsureConfigDir(cfgDir, name); err != nil {
				return err
			}
			fmt.Fprintf(out, "export CLAUDE_CONFIG_DIR=%s\n", shellQuote(cfgDir))
			fmt.Fprintf(out, "export CLAUDE_CODE_OAUTH_TOKEN=%s\n", shellQuote(cred.AccessToken))
			fmt.Fprintf(out, "export CCM_ACTIVE_PROFILE=%s\n", shellQuote(name))
			return nil
		},
	}
}

// shellQuote produces a single-quoted shell literal safe for `eval`.
func shellQuote(s string) string {
	// Single-quote and escape any embedded single quote by closing, escaping, and reopening.
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
