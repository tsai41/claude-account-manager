package paths

import (
	"os"
	"path/filepath"
)

func Home() string {
	h, _ := os.UserHomeDir()
	return h
}

func ClaudeJSON() string  { return filepath.Join(Home(), ".claude.json") }
func ClaudeDir() string   { return filepath.Join(Home(), ".claude") }
func CCMRoot() string     { return filepath.Join(Home(), ".ccm") }
func StateFile() string   { return filepath.Join(CCMRoot(), "state.json") }
func ConfigFile() string  { return filepath.Join(CCMRoot(), "config.json") }
func ProfilesDir() string { return filepath.Join(CCMRoot(), "profiles") }
func BackupsDir() string  { return filepath.Join(CCMRoot(), "backups") }
func LogsDir() string     { return filepath.Join(CCMRoot(), "logs") }

func ProfileDir(name string) string         { return filepath.Join(ProfilesDir(), name) }
func ProfileMetadata(name string) string    { return filepath.Join(ProfileDir(name), "metadata.json") }
func ProfileUsage(name string) string       { return filepath.Join(ProfileDir(name), "usage.json") }
func ProfileSnapshotsDir(name string) string {
	return filepath.Join(ProfileDir(name), "snapshots")
}

func EnsureRoot() error {
	for _, d := range []string{CCMRoot(), ProfilesDir(), BackupsDir(), LogsDir()} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	return nil
}
