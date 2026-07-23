package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"discodrive.org/daemon/internal/index"
	"discodrive.org/daemon/internal/safepath"
)

const pageLimit = 500

// Engine applies server changes to the local root directory.
type Engine struct {
	src  Source
	idx  *index.Index
	root string
}

func New(src Source, idx *index.Index, root string) *Engine {
	return &Engine{src: src, idx: idx, root: root}
}

// PullOnce fetches all changes after the cursor and applies them in order. The cursor
// advances after EACH applied change, so interruption mid-way is safe.
func (e *Engine) PullOnce(ctx context.Context) error {
	since, err := e.idx.Cursor()
	if err != nil {
		return err
	}
	for {
		changes, cursor, hasMore, err := e.src.Changes(ctx, since, pageLimit)
		if err != nil {
			return err
		}
		for _, c := range changes {
			if err := e.apply(ctx, c); err != nil {
				return fmt.Errorf("seq %d (%s): %w", c.Seq, c.RelPath, err)
			}
			if err := e.idx.SetCursor(c.Seq); err != nil {
				return err
			}
		}
		since = cursor
		if !hasMore {
			return nil
		}
	}
}

// ScopeEpoch returns the scope epoch the engine last reconciled to.
func (e *Engine) ScopeEpoch() (int64, error) { return e.idx.ScopeEpoch() }

// ResetForScope reconciles the local mirror after the server's sync scope changed: wipe the
// index + cursor, re-pull the (now differently scoped) tree from scratch, then delete local
// files that are no longer in scope. PushLocal is intentionally NOT run — local files were
// mapped to the OLD scope and must not be pushed into the new one. The resulting local data
// loss of the old mirror is expected and is gated behind the web danger-confirm modal.
func (e *Engine) ResetForScope(ctx context.Context, epoch int64) error {
	if err := e.idx.Clear(); err != nil {
		return err
	}
	if err := e.PullOnce(ctx); err != nil {
		return err
	}
	if err := e.sweepOrphans(); err != nil {
		return err
	}
	return e.idx.SetScopeEpoch(epoch)
}

// sweepOrphans deletes any file/dir under root that is not part of the index. The keep set
// includes every ancestor directory of every indexed node, so directories implicitly
// created for a file (without their own change entry) are never swept away.
func (e *Engine) sweepOrphans() error {
	nodes, err := e.idx.All()
	if err != nil {
		return err
	}
	keep := map[string]bool{".": true}
	for _, n := range nodes {
		for p := filepath.Clean(filepath.FromSlash(n.RelPath)); p != "." && p != "/" && p != ""; {
			keep[p] = true
			parent := filepath.Dir(p)
			if parent == p {
				break
			}
			p = parent
		}
	}
	var orphans []string
	_ = filepath.WalkDir(e.root, func(p string, d os.DirEntry, werr error) error {
		if werr != nil || p == e.root {
			return nil
		}
		if strings.HasPrefix(filepath.Base(p), ".kf-tmp-") {
			return nil
		}
		rel, rerr := filepath.Rel(e.root, p)
		if rerr != nil {
			return nil
		}
		if !keep[rel] {
			orphans = append(orphans, p)
		}
		return nil
	})
	// Deepest paths first, so a removed subtree never trips up a later removal.
	sort.Slice(orphans, func(i, j int) bool { return len(orphans[i]) > len(orphans[j]) })
	for _, p := range orphans {
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}
	return nil
}

// abs converts a server rel_path to an absolute path and verifies it stays within root.
// Besides the lexical containment check, it rejects paths whose existing components
// include a symlink: os.RemoveAll/MkdirAll/Rename would otherwise follow a symlinked
// directory and read, write, or delete data outside the sync folder. The same guard is
// shared (via internal/safepath) by the desktop and mobile file-write paths.
func (e *Engine) abs(rel string) (string, error) {
	return safepath.Join(e.root, rel)
}

func (e *Engine) apply(ctx context.Context, c Change) error {
	abs, err := e.abs(c.RelPath)
	if err != nil {
		return err
	}

	if c.Deleted {
		// Never let an empty / "." / "/" rel_path resolve the delete to the sync root
		// itself: a malicious server could otherwise RemoveAll the whole synced folder.
		if abs == filepath.Clean(e.root) {
			return fmt.Errorf("refusing to delete sync root (rel_path %q)", c.RelPath)
		}
		if err := os.RemoveAll(abs); err != nil {
			return err
		}
		return e.idx.Delete(c.NodeID)
	}

	if c.IsDir {
		// A known dir arriving under a new rel_path is a server-side move/rename.
		// Rename the whole subtree on disk in one shot — descendants' own feed
		// entries then find their files already in place and become index-only
		// no-ops. Without this the old dir survived empty, DetectLocal reported it
		// as a local create, and PushLocal resurrected a ghost folder on the server.
		if existing, ok, gerr := e.idx.Get(c.NodeID); gerr != nil {
			return gerr
		} else if ok && existing.RelPath != c.RelPath {
			if oldAbs, aerr := e.abs(existing.RelPath); aerr == nil {
				if fi, serr := os.Stat(oldAbs); serr == nil && fi.IsDir() {
					if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
						return err
					}
					// Rename may fail legitimately (e.g. target already exists after
					// a partial manual move) — fall through to MkdirAll; descendants
					// will be applied by their own feed entries.
					_ = os.Rename(oldAbs, abs)
				}
			}
		}
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return err
		}
		return e.idx.Put(index.Node{NodeID: c.NodeID, RelPath: c.RelPath, IsDir: true, Version: c.Version})
	}

	existing, ok, err := e.idx.Get(c.NodeID)
	if err != nil {
		return err
	}

	if ok && existing.RelPath != c.RelPath {
		if oldAbs, aerr := e.abs(existing.RelPath); aerr == nil {
			if _, serr := os.Stat(oldAbs); serr == nil {
				if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
					return err
				}
				if err := os.Rename(oldAbs, abs); err != nil {
					return err
				}
			}
		}
	}

	if ok && c.ContentHash != "" && existing.ContentHash == c.ContentHash {
		if _, serr := os.Stat(abs); serr == nil {
			return e.idx.Put(index.Node{NodeID: c.NodeID, RelPath: c.RelPath, Version: c.Version, ContentHash: c.ContentHash, Size: c.Size})
		}
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(abs), ".kf-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	h := sha256.New()
	if derr := e.src.Download(ctx, c.NodeID, io.MultiWriter(tmp, h)); derr != nil {
		tmp.Close()
		os.Remove(tmpName)
		return derr
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if c.ContentHash != "" && hex.EncodeToString(h.Sum(nil)) != c.ContentHash {
		os.Remove(tmpName)
		return fmt.Errorf("downloaded content hash mismatch")
	}
	if err := os.Rename(tmpName, abs); err != nil {
		os.Remove(tmpName)
		return err
	}
	return e.idx.Put(index.Node{NodeID: c.NodeID, RelPath: c.RelPath, Version: c.Version, ContentHash: c.ContentHash, Size: c.Size})
}
