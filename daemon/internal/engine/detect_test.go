package engine

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"discodrive.org/daemon/internal/index"
)

func TestDetectLocal(t *testing.T) {
	root := t.TempDir()
	idx, _ := index.Open(filepath.Join(t.TempDir(), "s.db"))
	defer idx.Close()

	os.WriteFile(filepath.Join(root, "keep.txt"), []byte("keep"), 0o644)
	os.WriteFile(filepath.Join(root, "changed.txt"), []byte("NEW"), 0o644)
	os.WriteFile(filepath.Join(root, "new.txt"), []byte("new"), 0o644)
	os.Mkdir(filepath.Join(root, "newdir"), 0o755)
	// macOS system files — must NOT appear in the detected changes
	os.WriteFile(filepath.Join(root, ".DS_Store"), []byte("junk"), 0o644)
	os.WriteFile(filepath.Join(root, "._new.txt"), []byte("appledouble"), 0o644)

	idx.Put(index.Node{NodeID: "k", RelPath: "keep.txt", ContentHash: hashOf([]byte("keep")), Size: 4})
	idx.Put(index.Node{NodeID: "c", RelPath: "changed.txt", ContentHash: hashOf([]byte("OLD")), Size: 3})
	idx.Put(index.Node{NodeID: "g", RelPath: "gone.txt", ContentHash: hashOf([]byte("x")), Size: 1})

	e := New(nil, idx, root)
	got, err := e.DetectLocal()
	if err != nil {
		t.Fatal(err)
	}
	ops := map[string]string{}
	for _, c := range got {
		ops[c.RelPath] = c.Op
	}
	want := map[string]string{"changed.txt": "update", "new.txt": "create", "newdir": "create", "gone.txt": "delete"}
	if len(ops) != len(want) {
		sort.Slice(got, func(i, j int) bool { return got[i].RelPath < got[j].RelPath })
		t.Fatalf("got %d changes: %+v", len(ops), got)
	}
	for p, op := range want {
		if ops[p] != op {
			t.Fatalf("%s: expected %s, got %s", p, op, ops[p])
		}
	}
	if ops["keep.txt"] != "" {
		t.Fatalf("keep.txt must not appear in changes")
	}
}
