# claude-account-manager (ccm)

Local-only manager for Claude Code OAuth account state on macOS.

Switch between multiple Claude Code logins without re-authenticating, by snapshotting `~/.claude.json`, the macOS Keychain entry that holds the OAuth tokens, and (optionally) `~/.claude/`. Bundles a TUI that also surfaces local activity and an estimated API-equivalent cost from your jsonl transcripts.

> All data stays on the local machine. No API keys, no cloud.

## Install

Pre-built binaries are published on the [Releases page](https://github.com/tsai41/claude-account-manager/releases). Download the macOS tarball matching your CPU (`darwin_arm64` for Apple Silicon, `darwin_amd64` for Intel), extract, and drop `ccm` somewhere on your `PATH`.

Or build from source:

```sh
make build          # local ./ccm
make install        # to $(go env GOPATH)/bin
make symlink        # additionally symlink to /usr/local/bin (sudo)
make uninstall      # remove both
```

Make sure `$(go env GOPATH)/bin` is on your `PATH`, or use `make symlink` to expose `ccm` under `/usr/local/bin`.

## Releasing

Tagged `v*` pushes trigger `.github/workflows/release.yml`, which runs goreleaser to build macOS arm64 + amd64 archives, generate a checksum file, and publish a GitHub release. Try a dry run locally:

```sh
make release-snapshot
```

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
| `ccm cost [-w today\|7d\|30d] [--json]` | machine-wide list-price cost estimate from jsonl |
| `ccm pricing show\|init\|path` | inspect or scaffold the pricing override file |
| `ccm rollback [id]` | list safety backups, or restore one by id |
| `ccm log [-n N]` | tail recent ccm log entries |
| `ccm doctor` | diagnostics including fingerprint duplicate detection |
| `ccm tui` | interactive TUI |
| `ccm version` | version info |
| `ccm completion <bash\|zsh\|fish\|powershell>` | print shell completion script |

## Shell completion

```sh
# zsh (one-shot session)
source <(ccm completion zsh)

# zsh (persistent — install to fpath)
ccm completion zsh > "${fpath[1]}/_ccm"

# bash
ccm completion bash > /usr/local/etc/bash_completion.d/ccm
```

## TUI

Tabs (top of screen): **1 Profiles**, **2 Costs**, **3 Activity**.

| Key | Action |
|---|---|
| `Tab` / `←` `→` / `1` `2` `3` | switch tab |
| `j` / `k` | move row (Profiles tab) |
| `Enter` | switch to selected profile (asks Y/n confirmation) |
| `e` | edit usage value |
| `u` | edit note |
| `d` | delete profile (`y`/`N` confirm) |
| `r` | refresh tab data |
| `q` / `Esc` | quit |

The Costs tab shows the **API-equivalent** dollar amount: what those tokens would cost on the pay-as-you-go API at public list rates, with cache-creation and cache-read multipliers applied. If you use Claude Pro / Max / Team, you pay a flat subscription, **not this number** — it is a usage signal, not an invoice. Today's number plus per-family breakdown, 7-day and 30-day totals, and a daily history bar.

By default, sub-agent (Task tool) messages with `isSidechain: true` are excluded from token totals; set `CCM_INCLUDE_SIDECHAIN=1` to include them.

To override per-model pricing, run `ccm pricing init` to scaffold `~/.ccm/pricing.json` and edit the multipliers. Set `cache_create_5m_mult` and `cache_create_1h_mult` to `0.1` if you want totals to align with tools that price cache writes like cache reads.

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
- doctor compares token fingerprints across profile keychain backups to catch the "two profiles share the same token" corruption case.
