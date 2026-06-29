package index

import (
	"path/filepath"
	"testing"
)

func TestIndexRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.db")
	idx, err := Open(p)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer idx.Close()

	if c, _ := idx.Cursor(); c != 0 {
		t.Fatalf("default cursor %d", c)
	}
	if err := idx.SetCursor(42); err != nil {
		t.Fatal(err)
	}
	if c, _ := idx.Cursor(); c != 42 {
		t.Fatalf("cursor after set: %d", c)
	}

	n := Node{NodeID: "n1", RelPath: "a/b.txt", IsDir: false, Version: 3, ContentHash: "h", Size: 10}
	if err := idx.Put(n); err != nil {
		t.Fatal(err)
	}
	got, ok, err := idx.Get("n1")
	if err != nil || !ok {
		t.Fatalf("Get: ok=%v err=%v", ok, err)
	}
	if got != n {
		t.Fatalf("round-trip: %+v != %+v", got, n)
	}

	n.RelPath = "a/c.txt"
	_ = idx.Put(n)
	got, _, _ = idx.Get("n1")
	if got.RelPath != "a/c.txt" {
		t.Fatalf("upsert rel_path: %q", got.RelPath)
	}

	if err := idx.Delete("n1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := idx.Get("n1"); ok {
		t.Fatalf("still present after Delete")
	}
}

func TestIndexCursorPersistsAcrossReopen(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.db")
	idx, _ := Open(p)
	_ = idx.SetCursor(7)
	idx.Close()

	idx2, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer idx2.Close()
	if c, _ := idx2.Cursor(); c != 7 {
		t.Fatalf("cursor after reopen: %d", c)
	}
}
