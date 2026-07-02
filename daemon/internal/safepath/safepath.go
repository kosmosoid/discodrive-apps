// Package safepath validates that a remote- or vault-derived relative path stays
// within a trusted root before it is used in a filesystem write. It mirrors the
// containment + symlink-rejection the sync engine already applies, so every code
// path that turns server-controlled names into local files shares one guard.
//
// root is the trusted base for a SPECIFIC write — a cache mirror, a per-open temp
// dir, the app-private decrypt dir — not any global location. Callers pass whichever
// root the write actually targets (e.g. the temp folder a vault is opened into, which
// lives OUTSIDE the vault storage), so the check adapts to each destination.
package safepath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Join cleans rel, joins it under root, and returns the absolute path only if it
// stays within root and no existing component along the way is a symlink. A malicious
// server sending "../../.zshrc", "/etc/passwd", or a path through a pre-planted
// symlinked directory is rejected. Legitimate nested paths pass unchanged.
func Join(root, rel string) (string, error) {
	root = filepath.Clean(root)
	p := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
	if p != root && !strings.HasPrefix(p, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes root: %q", rel)
	}
	if err := NoSymlinkComponents(root, p); err != nil {
		return "", err
	}
	return p, nil
}

// NoSymlinkComponents walks each path component below root and fails if an existing
// one is a symlink. Components that don't exist yet are fine — they'll be created as
// real directories. (Best-effort against pre-planted symlinks; not a TOCTOU guarantee.)
func NoSymlinkComponents(root, p string) error {
	root = filepath.Clean(root)
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil
	}
	cur := root
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		if part == "" || part == "." {
			continue
		}
		cur = filepath.Join(cur, part)
		fi, lerr := os.Lstat(cur)
		if lerr != nil {
			if os.IsNotExist(lerr) {
				return nil // the rest of the path doesn't exist yet — safe
			}
			return lerr
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path component is a symlink, refusing to follow: %q", cur)
		}
	}
	return nil
}
