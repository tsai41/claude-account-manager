package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const zshHook = `# ccm shell integration. Updates CLAUDE_CONFIG_DIR + CLAUDE_CODE_OAUTH_TOKEN
# from ~/.ccm/dir-map.json on every chpwd. Add to ~/.zshrc:
#   eval "$(ccm shell-init zsh)"

__ccm_chpwd() {
  command -v ccm >/dev/null 2>&1 || return 0
  local out
  out=$(ccm dir-export "$PWD" 2>/dev/null) || return 0
  [[ -n "$out" ]] && eval "$out"
}

autoload -Uz add-zsh-hook 2>/dev/null && add-zsh-hook chpwd __ccm_chpwd
__ccm_chpwd
`

const bashHook = `# ccm shell integration. Updates CLAUDE_CONFIG_DIR + CLAUDE_CODE_OAUTH_TOKEN
# from ~/.ccm/dir-map.json after every cd. Add to ~/.bashrc:
#   eval "$(ccm shell-init bash)"

__ccm_dir_apply() {
  command -v ccm >/dev/null 2>&1 || return 0
  local out
  out=$(ccm dir-export "$PWD" 2>/dev/null) || return 0
  [[ -n "$out" ]] && eval "$out"
}

cd() { builtin cd "$@" && __ccm_dir_apply; }
__ccm_dir_apply
`

func newShellInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "shell-init <shell>",
		Short:     "Print shell hook code that auto-routes Claude Code auth per `ccm bind` rules",
		ValidArgs: []string{"zsh", "bash"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			switch args[0] {
			case "zsh":
				fmt.Fprint(out, zshHook)
			case "bash":
				fmt.Fprint(out, bashHook)
			}
			return nil
		},
	}
}
