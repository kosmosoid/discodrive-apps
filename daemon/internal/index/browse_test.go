package index

import (
	"path/filepath"
	"sort"
	"testing"
)

func names(ns []Node) []string {
	out := make([]string, len(ns))
	for i, n := range ns {
		out[i] = n.RelPath
	}
	sort.Strings(out)
	return out
}

func TestChildren(t *testing.T) {
	idx, err := Open(filepath.Join(t.TempDir(), "i.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()

	for _, n := range []Node{
		{NodeID: "d1", RelPath: "docs", IsDir: true, Version: 1},
		{NodeID: "f1", RelPath: "docs/a.txt", Version: 1, Size: 3},
		{NodeID: "f2", RelPath: "docs/b.txt", Version: 1, Size: 4},
		{NodeID: "sub", RelPath: "docs/sub", IsDir: true, Version: 1},
		{NodeID: "deep", RelPath: "docs/sub/c.txt", Version: 1},
		{NodeID: "top", RelPath: "top.txt", Version: 1},
	} {
		if err := idx.Put(n); err != nil {
			t.Fatal(err)
		}
	}

	root, _ := idx.Children("")
	if got := names(root); len(got) != 2 || got[0] != "docs" || got[1] != "top.txt" {
		t.Fatalf("root children = %v, want [docs top.txt]", got)
	}
	docs, _ := idx.Children("docs")
	if got := names(docs); len(got) != 3 || got[0] != "docs/a.txt" || got[1] != "docs/b.txt" || got[2] != "docs/sub" {
		t.Fatalf("docs children = %v, want [docs/a.txt docs/b.txt docs/sub] (NOT docs/sub/c.txt)", got)
	}
}

func TestLocalTable(t *testing.T) {
	idx, err := Open(filepath.Join(t.TempDir(), "i.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()

	if err := idx.SetLocal("f1", "pinned", 1, "/p/docs/a.txt"); err != nil {
		t.Fatal(err)
	}
	state, stale, path := idx.LocalStatus("f1", 1)
	if state != "pinned" || stale || path != "/p/docs/a.txt" {
		t.Fatalf("local: %s %v %s", state, stale, path)
	}
	if _, st, _ := idx.LocalStatus("f1", 2); !st {
		t.Fatal("serverVersion 2 > 1 must be stale")
	}
	if idx.LocalPathOf("f1") != "/p/docs/a.txt" {
		t.Fatal("LocalPathOf mismatch")
	}
	if pins := idx.ListPinned(); len(pins) != 1 || pins[0] != "f1" {
		t.Fatalf("pinned: %v", pins)
	}
	if err := idx.DeleteLocal("f1"); err != nil {
		t.Fatal(err)
	}
	if state, _, _ := idx.LocalStatus("f1", 1); state != "" {
		t.Fatalf("after delete state=%q", state)
	}
}
