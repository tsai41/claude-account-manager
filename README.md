# claude-account-manager (ccm)

Local-only manager for Claude Code OAuth account state on macOS.

Switch between multiple Claude Code logins without re-authenticating, by snapshotting `~/.claude.json`, the macOS Keychain entry that holds the OAuth tokens, and (optionally) `~/.claude/`. Bundles a TUI that also surfaces local activity and an estimated API-equivalent cost from your jsonl transcripts.

> All data stays on the local machine. No API keys, no cloud.

## Build / install

```sh
make build          # local ./ccm
make install        # to $(go env GOPATH)/bin
make symlink        # additionally symlink to /usr/local/bin (sudo)
make uninstall      # remove both
```

Make sure `$(go env GOPATH)/bin` is on your `PATH`, or use `make symlink` to expose `ccm` under `/usr/local/bin`.

## Quick start

```sh
ccm doctor                    # check environment
ccm import-current work       # capture current login as profile "work"
# log into another account via Claude Code (claude /login)
ccm import-current personal   # capture as "personal"
ccm                           # default → TUI when profiles exist, else current/bootstrap hint
ccm use work                  # CLI switch (safe-merge by default)
```

## CLI

| Command | What |
|---|---|
| `ccm` | TUI when profiles exist; otherwise show current/bootstrap hint |
| `ccm current` | show current profile |
| `ccm list` | list profiles |
| `ccm import-current <name> [--force]` | capture currently logged-in OAuth state; `--force` to overwrite or accept duplicate email |
| `ccm use <name> [--full-restore]` | switch profile (default: safe-merge) |
| `ccm remove <name> [--keep-keychain-backup]` | delete profile |
| `ccm status [name]` | profile detail |
| `ccm usage [name]` | show usage record |
| `ccm usage-set <name> <value>` | set manual usage; parses `"session X%, weekly Y%"` |
| `ccm usage-note <name> <text>` | set usage note |
| `ccm usage-provider <name> <manual\|local-derived>` | choose provider |
| `ccm cost [-w today\|7d\|30d]` | machine-wide list-price cost estimate from jsonl |
| `ccm rollback [id]` | list safety backups, or restore one by id |
| `ccm log [-n N]` | tail recent ccm log entries |
| `ccm doctor` | diagnostics including fingerprint duplicate detection |
| `ccm tui` | interactive TUI |
| `ccm version` | version info |

## TUI

Tabs (top of screen): **1 Profiles**, **2 Costs**, **3 Activity**.

| Key | Action |
|---|---|
| `Tab` / `←` `→` / `1` `2` `3` | switch tab |
| `j` / `k` | move row (Profiles tab) |
| `Enter` | switch to selected profile |
| `e` | edit usage value |
| `u` | edit note |
| `d` | delete profile (`y`/`N` confirm) |
| `r` | refresh tab data |
| `q` / `Esc` | quit |

The Costs tab shows today's API-equivalent dollar amount with per-family breakdown (Opus / Sonnet / Haiku), 7-day and 30-day totals, and a daily history bar. Pricing is the public list rate; cache-creation and cache-read multipliers are applied. **Not an invoice.**

The Activity tab shows machine-wide turn counts (last 5h / today / last 7d) and last active timestamp. jsonl transcripts have no per-account binding, so these counts are not the official usage bar.

## Switch strategies

- **safe-merge** (default for `ccm use`): only auth-sensitive top-level keys in `~/.claude.json` are replaced; everything else (theme, MCP, preferences) is preserved. `~/.claude/` is left untouched.
- **full-restore** (`ccm use <name> --full-restore`): overwrites `~/.claude.json` and replaces `~/.claude/` contents from the snapshot tar (large transient subdirs are excluded from the snapshot anyway).

Every switch takes a safety backup under `~/.ccm/backups/` first; recover with `ccm rollback <id>`.

## Storage

- Profiles: `~/.ccm/profiles/<name>/`
- Snapshots: `~/.ccm/profiles/<name>/snapshots/<id>/` — `claude.json`, `claude-dir.tar.gz`, `account-meta.json`, `keychain-credential.json`
- State: `~/.ccm/state.json`
- Safety backups: `~/.ccm/backups/<id>/`
- Logs: `~/.ccm/logs/ccm.log` (JSONL)
- Keychain (managed by ccm): service `com.ccm.tokens`, account `<profile name>`

The Claude CLI's own keychain entry remains `Claude Code-credentials` / `<OS user>`. ccm reads/writes both via the `security` CLI.

## Notes

- Snapshot files contain raw OAuth tokens. Stored with mode `0600` inside `~/.ccm/` (mode `0700`).
- Restart any running Claude Code session after switching.
- doctor compares token fingerprints across profiles to catch the "two profiles share the same token" corruption case described in CCSwitcher's design notes.
