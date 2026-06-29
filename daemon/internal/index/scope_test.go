package index

import (
	"path/filepath"
	"testing"
)

func TestScopeEpochAndClear(t *testing.T) {
	idx, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer idx.Close()

	// Default epoch is 0.
	if e, err := idx.ScopeEpoch(); err != nil || e != 0 {
		t.Fatalf("default ScopeEpoch=%d err=%v", e, err)
	}
	if err := idx.SetScopeEpoch(5); err != nil {
		t.Fatalf("SetScopeEpoch: %v", err)
	}
	if e, _ := idx.ScopeEpoch(); e != 5 {
		t.Fatalf("ScopeEpoch=%d, want 5", e)
	}

	// Seed a node + cursor, then Clear.
	if err := idx.Put(Node{NodeID: "n1", RelPath: "a.txt", Version: 1}); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := idx.SetCursor(42); err != nil {
		t.Fatalf("SetCursor: %v", err)
	}
	if err := idx.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	all, _ := idx.All()
	if len(all) != 0 {
		t.Fatalf("after Clear, nodes=%d, want 0", len(all))
	}
	if c, _ := idx.Cursor(); c != 0 {
		t.Fatalf("after Clear, cursor=%d, want 0", c)
	}
	// Epoch survives Clear.
	if e, _ := idx.ScopeEpoch(); e != 5 {
		t.Fatalf("after Clear, epoch=%d, want 5 (preserved)", e)
	}
}
