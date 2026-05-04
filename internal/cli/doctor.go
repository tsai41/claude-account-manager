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
		for _, p := range profs {
			snapDir, _ := snapshot.Latest(p.Name)
			if snapDir == "" {
				add(check{"profile " + p.Name + " snapshot", false, "missing", "Re-run `ccm import-current --force " + p.Name + "`"})
			} else {
				add(check{"profile " + p.Name + " snapshot", true, snapDir, ""})
			}
			if _, err := keychain.ReadBackup(p.Name); err != nil {
				add(check{"profile " + p.Name + " keychain backup", false, err.Error(), "Re-run `ccm import-current --force " + p.Name + "`"})
			} else {
				add(check{"profile " + p.Name + " keychain backup", true, "ok", ""})
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
