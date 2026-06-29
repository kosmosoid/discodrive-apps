package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// isOSJunk reports whether a filename is a macOS system file we skip during sync
// (matching WebDAV projection behavior): .DS_Store and AppleDouble forks ._*.
func isOSJunk(name string) bool {
	return name == ".DS_Store" || strings.HasPrefix(name, "._")
}

// LocalChange is a local modification discovered by scanning the sync folder.
type LocalChange struct {
	Op      string // create | update | delete
	RelPath string
	IsDir   bool
	Hash    string
	Size    int64
}

// DetectLocal scans root and diffs it against the index: new, modified, and deleted nodes.
func (e *Engine) DetectLocal() ([]LocalChange, error) {
	knownNodes, err := e.idx.All()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(knownNodes))
	type prev struct {
		hash  string
		isDir bool
	}
	idxByPath := make(map[string]prev, len(knownNodes))
	for _, n := range knownNodes {
		idxByPath[n.RelPath] = prev{n.ContentHash, n.IsDir}
	}

	var out []LocalChange
	err = filepath.WalkDir(e.root, func(p string, d os.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if p == e.root {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".kf-tmp-") {
			return nil
		}
		// Skip macOS system files (matching WebDAV projection): .DS_Store and
		// AppleDouble forks ._* have no place on the server.
		if isOSJunk(d.Name()) {
			return nil
		}
		// Never follow symlinks. d.Type() reports the type without following, so a
		// symlink (to a file or directory) is skipped outright — otherwise a link
		// pointing outside the sync folder (e.g. ~/.ssh/id_rsa) would be read and
		// uploaded, or a symlinked directory would expose data outside the tree.
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		rel := filepath.ToSlash(mustRel(e.root, p))
		seen[rel] = true
		known, ok := idxByPath[rel]
		if d.IsDir() {
			if !ok {
				out = append(out, LocalChange{Op: "create", RelPath: rel, IsDir: true})
			}
			return nil
		}
		h, size, herr := hashFile(p)
		if herr != nil {
			return herr
		}
		switch {
		case !ok:
			out = append(out, LocalChange{Op: "create", RelPath: rel, Hash: h, Size: size})
		case known.hash != h:
			out = append(out, LocalChange{Op: "update", RelPath: rel, Hash: h, Size: size})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, n := range knownNodes {
		if !seen[n.RelPath] {
			out = append(out, LocalChange{Op: "delete", RelPath: n.RelPath, IsDir: n.IsDir})
		}
	}
	return out, nil
}

func hashFile(p string) (string, int64, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func mustRel(base, target string) string {
	r, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return r
}
