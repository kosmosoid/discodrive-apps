package desktop

import (
	"context"
	"testing"

	"discodrive.org/daemon/internal/engine"
)

func TestListReturnsChildrenWithState(t *testing.T) {
	f := &fakeServer{pages: []fakePage{{changes: []engine.Change{
		{Seq: 1, Op: "upsert", NodeID: "d", RelPath: "docs", IsDir: true, Version: 1},
		{Seq: 2, Op: "upsert", NodeID: "a", RelPath: "docs/a.txt", Version: 1, Size: 3},
	}, next: 2, hasMore: false}}}
	c, h := newTestController(t, f)
	if _, err := c.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	// mark a.txt cached at version 1
	if err := h.idx.SetLocal("a", "cached", 1, "/tmp/a.txt"); err != nil {
		t.Fatalf("SetLocal: %v", err)
	}

	entries, err := c.List("docs")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	e := entries[0]
	if e.Node.NodeID != "a" || e.State != "cached" || e.Stale || e.LocalPath != "/tmp/a.txt" {
		t.Fatalf("entry = %+v, want nodeID=a state=cached stale=false path=/tmp/a.txt", e)
	}
}
