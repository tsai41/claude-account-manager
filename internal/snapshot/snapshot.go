package snapshot

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tsai41/claude-account-manager/internal/claudeauth"
	"github.com/tsai41/claude-account-manager/internal/paths"
)

// Excluded subpaths (relative to ~/.claude/) that are large/transient and not needed for auth state.
var excludedDirs = map[string]bool{
	"projects":        true,
	"cache":           true,
	"shell-snapshots": true,
	"paste-cache":     true,
	"file-history":    true,
	"sessions":        true,
	"telemetry":       true,
	"session-tracker": true,
	"plugins":         true,
	"skills":          true,
	"plans":           true,
	"hooks":           true,
	"agents":          true,
	"commands":        true,
	"hud":             true,
	"ide":             true,
	"session-env":     true,
	"backups":         true,
	".omc":            true,
}

// Snapshot represents an on-disk snapshot of one profile's state.
type Snapshot struct {
	ID       string
	Dir      string
	Token    string                 // raw JSON from keychain (may be empty if not captured here)
	Account  claudeauth.AccountMeta
}

func NewID() string { return time.Now().Format("20060102-150405") }

// Create writes a snapshot bundle for profile: claude.json copy, claude-dir.tar.gz, account-meta.json,
// keychain-credential.json (if token non-empty).
func Create(profileName string, token string) (Snapshot, error) {
	id := NewID()
	dir := filepath.Join(paths.ProfileSnapshotsDir(profileName), id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Snapshot{}, err
	}

	// claude.json
	if err := copyFile(paths.ClaudeJSON(), filepath.Join(dir, "claude.json"), 0o600); err != nil {
		if !os.IsNotExist(err) {
			return Snapshot{}, fmt.Errorf("snapshot claude.json: %w", err)
		}
	}

	// claude-dir.tar.gz
	if err := tarClaudeDir(filepath.Join(dir, "claude-dir.tar.gz")); err != nil {
		return Snapshot{}, fmt.Errorf("snapshot claude dir: %w", err)
	}

	// account-meta.json
	meta, _ := claudeauth.ReadAccountMeta()
	if err := writeJSON(filepath.Join(dir, "account-meta.json"), meta, 0o600); err != nil {
		return Snapshot{}, fmt.Errorf("snapshot account-meta: %w", err)
	}

	// keychain-credential.json (token raw JSON content)
	if token != "" {
		if err := os.WriteFile(filepath.Join(dir, "keychain-credential.json"), []byte(token), 0o600); err != nil {
			return Snapshot{}, fmt.Errorf("snapshot keychain credential: %w", err)
		}
	}

	return Snapshot{ID: id, Dir: dir, Token: token, Account: meta}, nil
}

// Latest returns most recent snapshot dir for profile, or "" if none.
func Latest(profileName string) (string, error) {
	root := paths.ProfileSnapshotsDir(profileName)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "", nil
	}
	// IDs are sortable lexicographically (timestamp format)
	latest := names[0]
	for _, n := range names[1:] {
		if n > latest {
			latest = n
		}
	}
	return filepath.Join(root, latest), nil
}

// Restore applies a snapshot dir as full-restore: overwrite ~/.claude.json and merge ~/.claude/ contents.
// Returns the keychain token JSON if present in snapshot.
func Restore(snapshotDir string) (token string, err error) {
	cj := filepath.Join(snapshotDir, "claude.json")
	if _, statErr := os.Stat(cj); statErr == nil {
		if err := copyFile(cj, paths.ClaudeJSON(), 0o600); err != nil {
			return "", fmt.Errorf("restore claude.json: %w", err)
		}
	}
	tarPath := filepath.Join(snapshotDir, "claude-dir.tar.gz")
	if _, statErr := os.Stat(tarPath); statErr == nil {
		if err := untarClaudeDir(tarPath); err != nil {
			return "", fmt.Errorf("restore claude dir: %w", err)
		}
	}
	tokPath := filepath.Join(snapshotDir, "keychain-credential.json")
	if b, readErr := os.ReadFile(tokPath); readErr == nil {
		token = string(b)
	}
	return token, nil
}

// BackupCurrent creates a global backup of the current LIVE state (claude.json + claude dir + token).
// Used as a safety net before destructive operations.
func BackupCurrent(label, token string) (string, error) {
	id := NewID() + "-" + sanitize(label)
	dir := filepath.Join(paths.BackupsDir(), id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if err := copyFile(paths.ClaudeJSON(), filepath.Join(dir, "claude.json"), 0o600); err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err := tarClaudeDir(filepath.Join(dir, "claude-dir.tar.gz")); err != nil {
		return "", err
	}
	if token != "" {
		if err := os.WriteFile(filepath.Join(dir, "keychain-credential.json"), []byte(token), 0o600); err != nil {
			return "", err
		}
	}
	return dir, nil
}

func sanitize(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "backup"
	}
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			out = append(out, r)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func writeJSON(path string, v any, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := jsonEncoder(f)
	return enc.Encode(v)
}

func tarClaudeDir(outPath string) error {
	root := paths.ClaudeDir()
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			// nothing to tar
			f, e := os.Create(outPath)
			if e != nil {
				return e
			}
			gz := gzip.NewWriter(f)
			tw := tar.NewWriter(gz)
			tw.Close()
			gz.Close()
			f.Close()
			return nil
		}
		return err
	}
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}
		// skip excluded top-level dirs
		topComp := strings.SplitN(filepath.ToSlash(rel), "/", 2)[0]
		if excludedDirs[topComp] {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		// skip sockets, devices, etc.
		if !info.Mode().IsRegular() && !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			return nil
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			hdr.Linkname = link
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			fh, err := os.Open(path)
			if err != nil {
				return err
			}
			_, err = io.Copy(tw, fh)
			fh.Close()
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func untarClaudeDir(tarPath string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	root := paths.ClaudeDir()
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// safety: reject absolute paths and ".."
		clean := filepath.Clean(hdr.Name)
		if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			continue
		}
		dst := filepath.Join(root, clean)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dst, os.FileMode(hdr.Mode)&0o777); err != nil {
				return err
			}
		case tar.TypeSymlink:
			os.Remove(dst)
			if err := os.Symlink(hdr.Linkname, dst); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
				return err
			}
			out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}
