package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// ResetForScope: after a scope change, re-pull the new tree and delete local files that fell
// out of scope, while keeping (and not re-deleting) ancestor directories of kept files.
func TestResetForScope(t *testing.T) {
	body := []byte("keep me")
	src := &fakeSource{
		changes: []Change{{Seq: 1, Op: "create", NodeID: "k1", RelPath: "dir/keep.txt", ContentHash: hashOf(body), Size: int64(len(body))}},
		bodies:  map[string][][]byte{"k1": {body}},
	}
	e, root := newEngine(t, src)

	// Simulate leftovers from a previous (wider) scope: an orphan file and an orphan subtree.
	if err := os.WriteFile(filepath.Join(root, "old.txt"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "oldsub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "oldsub", "x.txt"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := e.ResetForScope(context.Background(), 2); err != nil {
		t.Fatalf("ResetForScope: %v", err)
	}

	// Kept file (and its ancestor dir) present.
	if got, err := os.ReadFile(filepath.Join(root, "dir", "keep.txt")); err != nil || string(got) != "keep me" {
		t.Fatalf("keep.txt: %q err=%v", got, err)
	}
	// Orphans gone.
	if _, err := os.Stat(filepath.Join(root, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("old.txt should be swept, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "oldsub")); !os.IsNotExist(err) {
		t.Fatalf("oldsub/ should be swept, err=%v", err)
	}
	// Epoch recorded.
	if ep, _ := e.ScopeEpoch(); ep != 2 {
		t.Fatalf("ScopeEpoch=%d, want 2", ep)
	}
}
