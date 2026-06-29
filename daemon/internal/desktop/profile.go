// Package desktop implements the on-demand controller for the cross-platform
// Fyne desktop client: tree browsing, lazy content download, pin/cache, and
// vault/pairing wiring over the daemon's internal protocol/index/vaultmgr.
package desktop

import (
	"os"
	"path/filepath"
)

// ProfileDir is the desktop client's private profile directory, isolated from the
// CLI daemon's config so the two never fight over one config.json / state.db.
func ProfileDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "discodrive", "desktop"), nil
}

// IndexDBPath is the SQLite index mirror inside the profile.
func IndexDBPath(profileDir string) string { return filepath.Join(profileDir, "index.db") }

// ContentDir is where on-demand file content is cached inside the profile.
func ContentDir(profileDir string) string { return filepath.Join(profileDir, "content") }
