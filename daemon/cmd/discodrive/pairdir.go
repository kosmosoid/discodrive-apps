package main

import (
	"net/url"
	"os"
	"strings"

	"discodrive.org/daemon/internal/config"
)

// removeStateDB deletes the daemon's index database (with SQLite sidecars) so the
// next run rebuilds it from scratch. Used when the paired server changes.
func removeStateDB(cfgPath string) error {
	db := config.StateDBPath(cfgPath)
	for _, p := range []string{db, db + "-wal", db + "-shm", db + "-journal"} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// freshSyncDirFor derives a clean per-server sync dir from the default one, e.g.
// ~/discodrive → ~/discodrive-fd.example.dev. Used when re-pairing to a different
// server so the old server's files are never pushed to the new one.
func freshSyncDirFor(defaultDir, server string) string {
	suffix := "new"
	if u, err := url.Parse(server); err == nil && u.Hostname() != "" {
		suffix = u.Hostname()
		if p := u.Port(); p != "" {
			suffix += "-" + p
		}
	}
	var b strings.Builder
	for _, r := range suffix {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return defaultDir + "-" + b.String()
}

// resolvePairDir picks the sync dir for a pairing. Re-pairing to a DIFFERENT
// server must not adopt the old server's folder (its contents would be uploaded
// wholesale to the new server), so unless the user explicitly chose a dir, a
// fresh per-server folder is used; the old folder is left untouched on disk.
func resolvePairDir(requested string, dirExplicit bool, oldServer, newServer, defaultDir string) string {
	if oldServer == "" || oldServer == newServer || dirExplicit {
		return requested
	}
	return freshSyncDirFor(defaultDir, newServer)
}
