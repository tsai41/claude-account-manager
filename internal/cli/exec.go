package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/paths"
	"github.com/tsai41/claude-account-manager/internal/profile"
	"github.com/tsai41/claude-account-manager/internal/snapshot"
)

func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec <profile> -- <command> [args...]",
		Short: "Run a command with isolated CLAUDE_CONFIG_DIR + CLAUDE_CODE_OAUTH_TOKEN for the given profile",
		Long: "Resolves the profile's latest OAuth snapshot, sets CLAUDE_CONFIG_DIR to a per-profile " +
			"directory under ~/.ccm/configs/, exports the access token via CLAUDE_CODE_OAUTH_TOKEN, " +
			"and exec()s the command. Use -- to separate ccm args from the command.",
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			rest := args[1:]
			// Strip a leading "--" if present (POSIX argv separator).
			if len(rest) > 0 && rest[0] == "--" {
				rest = rest[1:]
			}
			if len(rest) == 0 {
				return errors.New("missing command to exec")
			}
			return runExec(name, rest)
		},
	}
	cmd.Flags().SetInterspersed(false) // stop parsing at first non-flag so trailing -p etc. pass through
	return cmd
}

func runExec(name string, argv []string) error {
	if err := profile.ValidateName(name); err != nil {
		return err
	}
	if !profile.Exists(name) {
		return fmt.Errorf("profile %q not found", name)
	}

	cred, _, err := snapshot.LoadLatestOAuth(name)
	if err != nil {
		return fmt.Errorf("load oauth for %s: %w", name, err)
	}
	if cred.Expired() {
		fmt.Fprintf(os.Stderr,
			"warning: profile %q access token expired at %s; refresh not yet implemented — request may fail\n",
			name, cred.ExpiresUnix().Format("2006-01-02 15:04:05"))
	}

	cfgDir := paths.ProfileConfigDir(name)
	if err := snapshot.EnsureConfigDir(cfgDir, name); err != nil {
		return fmt.Errorf("prepare config dir %s: %w", cfgDir, err)
	}

	binPath, err := exec.LookPath(argv[0])
	if err != nil {
		return fmt.Errorf("resolve %s: %w", argv[0], err)
	}

	env := mergeEnv(os.Environ(), map[string]string{
		"CLAUDE_CONFIG_DIR":       cfgDir,
		"CLAUDE_CODE_OAUTH_TOKEN": cred.AccessToken,
	})

	// syscall.Exec replaces the current process so signals / TTY / exit code pass through cleanly.
	return syscall.Exec(binPath, argv, env)
}

// mergeEnv returns environ with overrides applied, replacing existing keys.
func mergeEnv(environ []string, overrides map[string]string) []string {
	seen := make(map[string]bool, len(overrides))
	out := make([]string, 0, len(environ)+len(overrides))
	for _, e := range environ {
		key := envKey(e)
		if v, ok := overrides[key]; ok {
			out = append(out, key+"="+v)
			seen[key] = true
			continue
		}
		out = append(out, e)
	}
	for k, v := range overrides {
		if !seen[k] {
			out = append(out, k+"="+v)
		}
	}
	return out
}

func envKey(entry string) string {
	for i := 0; i < len(entry); i++ {
		if entry[i] == '=' {
			return entry[:i]
		}
	}
	return entry
}
