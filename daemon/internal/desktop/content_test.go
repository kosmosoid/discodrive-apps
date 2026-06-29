package desktop

import (
	"context"
	"os"
	"testing"

	"discodrive.org/daemon/internal/engine"
)

func TestOpenDownloadsAndCaches(t *testing.T) {
	f := &fakeServer{
		pages: []fakePage{{changes: []engine.Change{
			{Seq: 1, Op: "upsert", NodeID: "a", RelPath: "a.txt", Version: 1, Size: 5},
		}, next: 1, hasMore: false}},
		download: map[string][]byte{"a": []byte("hello")},
	}
	c, h := newTestController(t, f)
	if _, err := c.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	path, err := c.Open(context.Background(), "a")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("content = %q, want hello", data)
	}
	if state, _, _ := h.idx.LocalStatus("a", 1); state != "cached" {
		t.Fatalf("state = %q, want cached", state)
	}

	// Pin then Unpin
	if _, err := c.Pin(context.Background(), "a"); err != nil {
		t.Fatalf("Pin: %v", err)
	}
	if state, _, _ := h.idx.LocalStatus("a", 1); state != "pinned" {
		t.Fatalf("state after Pin = %q, want pinned", state)
	}
	if err := c.Unpin("a"); err != nil {
		t.Fatalf("Unpin: %v", err)
	}
	if state, _, _ := h.idx.LocalStatus("a", 1); state != "cached" {
		t.Fatalf("state after Unpin = %q, want cached", state)
	}

	// RemoveLocal deletes file + record
	if err := c.RemoveLocal("a"); err != nil {
		t.Fatalf("RemoveLocal: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file should be gone, stat err = %v", err)
	}
	if state, _, _ := h.idx.LocalStatus("a", 1); state != "" {
		t.Fatalf("state after RemoveLocal = %q, want \"\" (no local record)", state)
	}
}

// TestPinReusesFreshLocalCopy verifies that pinning a node whose local copy is already
// cached and not stale promotes it to "pinned" WITHOUT re-downloading the content.
func TestPinReusesFreshLocalCopy(t *testing.T) {
	f := &fakeServer{
		pages: []fakePage{{changes: []engine.Change{
			{Seq: 1, Op: "upsert", NodeID: "a", RelPath: "a.txt", Version: 1, Size: 5},
		}, next: 1, hasMore: false}},
		download: map[string][]byte{"a": []byte("hello")},
	}
	c, h := newTestController(t, f)
	if _, err := c.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	// First Open downloads + caches (1 download).
	if _, err := c.Open(context.Background(), "a"); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if f.downloads != 1 {
		t.Fatalf("downloads after Open = %d, want 1", f.downloads)
	}

	// Pinning the already-fresh copy must NOT re-download.
	if _, err := c.Pin(context.Background(), "a"); err != nil {
		t.Fatalf("Pin: %v", err)
	}
	if f.downloads != 1 {
		t.Fatalf("downloads after Pin of fresh copy = %d, want 1 (no re-download)", f.downloads)
	}
	if state, _, _ := h.idx.LocalStatus("a", 1); state != "pinned" {
		t.Fatalf("state after Pin = %q, want pinned", state)
	}
}
