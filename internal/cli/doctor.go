package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/claudeauth"
	"github.com/tsai41/claude-account-manager/internal/keychain"
	"github.com/tsai41/claude-account-manager/internal/paths"
	"github.com/tsai41/claude-account-manager/internal/profile"
	"github.com/tsai41/claude-account-manager/internal/snapshot"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose environment and profile health",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}
}

type check struct {
	name string
	ok   bool
	msg  string
	hint string
}

func runDoctor() error {
	var checks []check
	add := func(c check) { checks = append(checks, c) }

	// 1. ~/.claude.json
	if _, err := os.Stat(paths.ClaudeJSON()); err == nil {
		add(check{"~/.claude.json readable", true, paths.ClaudeJSON(), ""})
	} else {
		add(check{"~/.claude.json readable", false, err.Error(), "Run Claude Code at least once."})
	}

	// 2. ~/.claude/
	if fi, err := os.Stat(paths.ClaudeDir()); err == nil && fi.IsDir() {
		add(check{"~/.claude/ exists", true, paths.ClaudeDir(), ""})
	} else {
		add(check{"~/.claude/ exists", false, fmt.Sprintf("%v", err), "Run Claude Code at least once."})
	}

	// 3. account meta
	meta, err := claudeauth.ReadAccountMeta()
	if err != nil {
		add(check{"oauthAccount readable", false, err.Error(), "Check ~/.claude.json for parse errors."})
	} else if meta.Email == "" {
		add(check{"oauthAccount.emailAddress present", false, "missing", "Re-login Claude Code (claude /login)."})
	} else {
		add(check{"oauthAccount.emailAddress present", true, meta.Email, ""})
	}

	// 4. live keychain
	tok, kerr := keychain.ReadLive()
	if kerr != nil {
		add(check{"keychain LIVE readable", false, kerr.Error(), "Run `claude /login` to authenticate."})
	} else {
		fp := keychain.Fingerprint(tok)
		if fp == "" {
			add(check{"keychain LIVE token shape", false, "no accessToken field", "Re-login Claude Code."})
		} else {
			add(check{"keychain LIVE readable", true, "fp=" + fp, ""})
		}
	}

	// 5. ccm root
	if err := paths.EnsureRoot(); err != nil {
		add(check{"~/.ccm/ writable", false, err.Error(), "Check filesystem permissions."})
	} else {
		add(check{"~/.ccm/ writable", true, paths.CCMRoot(), ""})
	}

	// 6. profiles
	profs, perr := profile.List()
	if perr != nil {
		add(check{"profiles directory", false, perr.Error(), ""})
	} else {
		add(check{"profiles directory", true, fmt.Sprintf("%d profile(s)", len(profs)), ""})
		fps := make(map[string]string) // fp -> profile name (first seen)
		liveFP := ""
		if tok != "" {
			liveFP = keychain.Fingerprint(tok)
		}
		st, _ := profile.LoadState()
		for _, p := range profs {
			snapDir, _ := snapshot.Latest(p.Name)
			if snapDir == "" {
				add(check{"profile " + p.Name + " snapshot", false, "missing", "Re-run `ccm import-current --force " + p.Name + "`"})
			} else {
				add(check{"profile " + p.Name + " snapshot", true, snapDir, ""})
			}
			bkTok, berr := keychain.ReadBackup(p.Name)
			if berr != nil {
				add(check{"profile " + p.Name + " keychain backup", false, berr.Error(), "Re-run `ccm import-current --force " + p.Name + "`"})
				continue
			}
			fp := keychain.Fingerprint(bkTok)
			add(check{"profile " + p.Name + " keychain backup", true, "fp=" + fp, ""})
			if fp != "" {
				if other, dup := fps[fp]; dup {
					add(check{
						"fingerprint duplicate",
						false,
						fmt.Sprintf("profiles %q and %q share token fp %s", other, p.Name, fp),
						"Re-run `ccm import-current --force <name>` for the offending profile while logged in as the correct account.",
					})
				} else {
					fps[fp] = p.Name
				}
			}
		}
		// CHECK 2: live token vs current profile's backup
		if liveFP != "" && st.CurrentProfile != "" {
			if currentProfileFP, ok := fps[liveFP]; ok {
				if currentProfileFP == st.CurrentProfile {
					add(check{"live token matches current profile", true, "fp=" + liveFP, ""})
				} else {
					add(check{
						"live token vs current profile",
						false,
						fmt.Sprintf("live fp matches profile %q but current is %q (state desync)", currentProfileFP, st.CurrentProfile),
						"Run `ccm use " + currentProfileFP + "` or `ccm use " + st.CurrentProfile + "` to resync.",
					})
				}
			} else {
				add(check{
					"live token vs backups",
					false,
					"live fp matches no backup (Claude CLI may have refreshed the token)",
					"Run `ccm import-current --force " + st.CurrentProfile + "` to refresh the backup.",
				})
			}
		}
	}

	// print
	failed := 0
	for _, c := range checks {
		mark := "OK "
		if !c.ok {
			mark = "FAIL"
			failed++
		}
		fmt.Printf("[%s] %s — %s\n", mark, c.name, c.msg)
		if !c.ok && c.hint != "" {
			fmt.Printf("       hint: %s\n", c.hint)
		}
	}
	fmt.Println()
	if failed == 0 {
		fmt.Println("All checks passed.")
	} else {
		fmt.Printf("%d check(s) failed.\n", failed)
	}
	return nil
}
