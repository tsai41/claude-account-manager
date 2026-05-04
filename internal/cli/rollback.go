package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/tsai41/claude-account-manager/internal/keychain"
	"github.com/tsai41/claude-account-manager/internal/logger"
	"github.com/tsai41/claude-account-manager/internal/paths"
	"github.com/tsai41/claude-account-manager/internal/snapshot"
)

func newRollbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback [id]",
		Short: "List safety backups, or restore one by id",
		Long: "Without arguments, lists safety backups stored in ~/.ccm/backups/.\n" +
			"With an id, restores claude.json, ~/.claude/, and the keychain LIVE token from that backup.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return listBackups(cmd)
			}
			return restoreBackup(args[0])
		},
	}
	return cmd
}

func listBackups(cmd *cobra.Command) error {
	root := paths.BackupsDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("(no backups)")
			return nil
		}
		return err
	}
	var dirs []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e)
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() > dirs[j].Name() })
	if len(dirs) == 0 {
		fmt.Println("(no backups)")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPATH\tSIZE")
	for _, d := range dirs {
		full := filepath.Join(root, d.Name())
		size := dirSize(full)
		fmt.Fprintf(w, "%s\t%s\t%s\n", d.Name(), full, humanSize(size))
	}
	return w.Flush()
}

func restoreBackup(id string) error {
	dir := filepath.Join(paths.BackupsDir(), id)
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return fmt.Errorf("backup %q not found at %s", id, dir)
	}
	// Take one more safety backup of *current* state before rolling back, so we can undo the rollback.
	curTok, _ := keychain.ReadLive()
	pre, err := snapshot.BackupCurrent("pre-rollback-"+id, curTok)
	if err != nil {
		return fmt.Errorf("pre-rollback safety backup: %w", err)
	}

	token, err := snapshot.Restore(dir)
	if err != nil {
		return fmt.Errorf("restore from backup: %w (pre-rollback backup at %s)", err, pre)
	}
	if token != "" {
		if err := keychain.WriteLive(token); err != nil {
			return fmt.Errorf("write live keychain: %w (pre-rollback backup at %s)", err, pre)
		}
	}
	logger.Info("rollback", "", "rollback complete", map[string]any{
		"backup_id":          id,
		"pre_rollback_dir":   pre,
		"token_restored":     token != "",
	})
	fmt.Printf("Rolled back from: %s\n", dir)
	if token != "" {
		fmt.Printf("Restored keychain token fp: %s\n", keychain.Fingerprint(token))
	}
	fmt.Printf("Pre-rollback safety backup: %s\n", pre)
	fmt.Println("Tip: run `ccm doctor` to verify state, restart any running Claude Code sessions.")
	return nil
}

func dirSize(p string) int64 {
	var total int64
	filepath.Walk(p, func(_ string, info os.FileInfo, err error) error {
		if err == nil && info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total
}

func humanSize(n int64) string {
	const k = 1024
	switch {
	case n < k:
		return fmt.Sprintf("%dB", n)
	case n < k*k:
		return fmt.Sprintf("%.1fK", float64(n)/k)
	case n < k*k*k:
		return fmt.Sprintf("%.1fM", float64(n)/(k*k))
	default:
		return fmt.Sprintf("%.1fG", float64(n)/(k*k*k))
	}
}
