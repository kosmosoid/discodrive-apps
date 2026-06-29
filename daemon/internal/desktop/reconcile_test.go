package desktop

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"discodrive.org/daemon/internal/engine"
)

// After the server deletes a node, Refresh must remove the cached file from disk, not
// just the index record.
func TestRefreshDeleteRemovesCachedFile(t *testing.T) {
	f := &fakeServer{
		pages: []fakePage{
			{changes: []engine.Change{{Seq: 1, Op: "upsert", NodeID: "a", RelPath: "a.txt", Version: 1, Size: 5}}, next: 1},
			{changes: []engine.Change{{Seq: 2, Op: "delete", NodeID: "a", Deleted: true}}, next: 2},
		},
		download: map[string][]byte{"a": []byte("hello")},
	}
	c, h := newTestController(t, f)
	if _, err := c.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh #1: %v", err)
	}
	path, err := c.Open(context.Background(), "a")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cached file should exist after Open: %v", err)
	}

	if _, err := c.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh #2 (delete): %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cached file should be gone after delete, stat err = %v", err)
	}
	if state, _, _ := h.idx.LocalStatus("a", 1); state != "" {
		t.Fatalf("local record should be gone after delete, state = %q", state)
	}
}

// After the server renames a node, Refresh must relocate the cached file to the new
// path so the old name does not orphan on disk.
func TestRefreshRenameRelocatesCachedFile(t *testing.T) {
	f := &fakeServer{
		pages: []fakePage{
			{changes: []engine.Change{{Seq: 1, Op: "upsert", NodeID: "a", RelPath: "a.txt", Version: 1, Size: 5}}, next: 1},
			{changes: []engine.Change{{Seq: 2, Op: "upsert", NodeID: "a", RelPath: "b.txt", Version: 2, Size: 5}}, next: 2},
		},
		download: map[string][]byte{"a": []byte("hello")},
	}
	c, _ := newTestController(t, f)
	if _, err := c.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh #1: %v", err)
	}
	oldPath, err := c.Open(context.Background(), "a")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if _, err := c.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh #2 (rename): %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old-name file should be gone after rename, stat err = %v", err)
	}
	newPath := filepath.Join(filepath.Dir(oldPath), "b.txt")
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("new-name file should exist after rename: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("relocated content = %q, want hello", data)
	}
}
