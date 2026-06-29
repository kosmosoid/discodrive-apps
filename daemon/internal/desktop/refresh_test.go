package desktop

import (
	"context"
	"testing"

	"discodrive.org/daemon/internal/engine"
)

func TestRefreshAppliesUpsertsAndDeletes(t *testing.T) {
	f := &fakeServer{pages: []fakePage{
		{changes: []engine.Change{
			{Seq: 1, Op: "upsert", NodeID: "n1", RelPath: "docs", IsDir: true, Version: 1},
			{Seq: 2, Op: "upsert", NodeID: "n2", RelPath: "docs/a.txt", Version: 1, Size: 3},
		}, next: 2, hasMore: true},
		{changes: []engine.Change{
			{Seq: 3, Op: "delete", NodeID: "n2", RelPath: "docs/a.txt", Deleted: true},
		}, next: 3, hasMore: false},
	}}
	c, h := newTestController(t, f)

	applied, err := c.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if applied != 3 {
		t.Fatalf("applied = %d, want 3", applied)
	}
	if _, ok, _ := h.idx.Get("n1"); !ok {
		t.Fatal("n1 should be present after upsert")
	}
	if _, ok, _ := h.idx.Get("n2"); ok {
		t.Fatal("n2 should be gone after delete")
	}
	if cur, _ := h.idx.Cursor(); cur != 3 {
		t.Fatalf("cursor = %d, want 3", cur)
	}
}
