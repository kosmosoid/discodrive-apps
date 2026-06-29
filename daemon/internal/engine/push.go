package engine

import (
	"context"
	"io"
	"os"
	"sort"
	"strings"

	"discodrive.org/daemon/internal/index"
)

// RemoteNode is what the server returns after a push (to be recorded in the index).
type RemoteNode struct {
	NodeID  string
	Version int64
	Hash    string
}

// Sink receives local changes (implemented by the HTTP protocol client).
type Sink interface {
	PushFile(ctx context.Context, relPath string, baseVersion *int64, r io.Reader) (RemoteNode, bool, error)
	EnsureDir(ctx context.Context, relPath string) (RemoteNode, error)
	DeleteRemote(ctx context.Context, relPath string) error
}

// PushLocal detects local changes and uploads them via sink, updating the index.
func (e *Engine) PushLocal(ctx context.Context, sink Sink) error {
	changes, err := e.DetectLocal()
	if err != nil {
		return err
	}
	sort.SliceStable(changes, func(i, j int) bool {
		di, dj := depth(changes[i].RelPath), depth(changes[j].RelPath)
		if changes[i].Op == "delete" && changes[j].Op == "delete" {
			return di > dj // deeper deletes go first
		}
		return di < dj // creates/dirs top-down
	})

	knownNodes := map[string]index.Node{}
	if all, err := e.idx.All(); err == nil {
		for _, n := range all {
			knownNodes[n.RelPath] = n
		}
	}

	for _, c := range changes {
		abs, err := e.abs(c.RelPath)
		if err != nil {
			return err
		}
		switch c.Op {
		case "create":
			if c.IsDir {
				rn, err := sink.EnsureDir(ctx, c.RelPath)
				if err != nil {
					return err
				}
				if err := e.idx.Put(index.Node{NodeID: rn.NodeID, RelPath: c.RelPath, IsDir: true, Version: rn.Version}); err != nil {
					return err
				}
				continue
			}
			fallthrough
		case "update":
			f, err := os.Open(abs)
			if err != nil {
				return err
			}
			var base *int64
			if n, ok := knownNodes[c.RelPath]; ok {
				v := n.Version
				base = &v
			}
			rn, conflicted, perr := sink.PushFile(ctx, c.RelPath, base, f)
			f.Close()
			if perr != nil {
				return perr
			}
			if conflicted {
				// The server version stays as the canonical copy; ours was saved as a
				// conflict copy and will arrive on the next pull (as `name (conflict...).ext`).
				// Leave the index untouched — pull will bring the server's main version and
				// overwrite the local file. No infinite loop as long as the push→pull order is respected.
				continue
			}
			hash := rn.Hash
			if hash == "" {
				hash = c.Hash
			}
			if err := e.idx.Put(index.Node{NodeID: rn.NodeID, RelPath: c.RelPath, Version: rn.Version, ContentHash: hash, Size: c.Size}); err != nil {
				return err
			}
		case "delete":
			if err := sink.DeleteRemote(ctx, c.RelPath); err != nil {
				return err
			}
			if n, ok := knownNodes[c.RelPath]; ok {
				if err := e.idx.Delete(n.NodeID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func depth(rel string) int { return strings.Count(rel, "/") }
