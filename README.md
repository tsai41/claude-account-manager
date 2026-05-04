# claude-account-manager (ccm)

Local-only manager for Claude Code OAuth account state on macOS.

Switch between multiple Claude Code logins without re-authenticating, by snapshotting `~/.claude.json`, `~/.claude/`, and the macOS Keychain entry that holds the OAuth tokens.

> All data stays on the local machine. No API keys, no cloud.

## Status

Phase 1 (MVP). Switch strategy: full-restore.

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
ccm list
ccm use work                  # switch back
ccm                           # show current
```

## Commands

| Command | What |
|---|---|
| `ccm` / `ccm current` | show current profile |
| `ccm list` | list profiles |
| `ccm import-current <name>` | capture currently logged-in OAuth state |
| `ccm use <name>` | switch profile (full-restore) |
| `ccm remove <name>` | delete profile |
| `ccm status [name]` | profile detail |
| `ccm usage [name]` | show usage record |
| `ccm usage-set <name> <value>` | set manual usage |
| `ccm usage-note <name> <text>` | set usage note |
| `ccm doctor` | diagnostics |
| `ccm tui` | interactive table UI (j/k Enter switch, r refresh, u edit-note, d delete, q quit) |

## Storage

- Profiles: `~/.ccm/profiles/<name>/`
- Snapshots: `~/.ccm/profiles/<name>/snapshots/<id>/`
  - `claude.json`, `claude-dir.tar.gz`, `account-meta.json`, `keychain-credential.json`
- State: `~/.ccm/state.json`
- Safety backups: `~/.ccm/backups/`
- Keychain (managed by ccm): service `com.ccm.tokens`, account `<profile name>`

The Claude CLI's own keychain entry remains `Claude Code-credentials` / `<OS user>`. ccm reads/writes both via the `security` CLI.

## Notes

- Switching uses full-restore: claude.json overwritten, claude dir contents replaced from tar (large transient subdirs like `projects/`, `cache/`, `sessions/` are excluded from snapshots and not touched on restore).
- Snapshot files contain raw OAuth tokens. Stored with mode 0600 inside `~/.ccm/` (mode 0700).
- Restart any running Claude Code session after switching.
