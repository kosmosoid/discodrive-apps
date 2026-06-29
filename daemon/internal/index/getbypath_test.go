package index

import (
	"path/filepath"
	"testing"
)

func TestGetByPath(t *testing.T) {
	idx, err := Open(filepath.Join(t.TempDir(), "i.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()

	if err := idx.Put(Node{NodeID: "n1", RelPath: "MyVault/masterkey.cryptomator", Version: 1}); err != nil {
		t.Fatal(err)
	}

	n, ok, err := idx.GetByPath("MyVault/masterkey.cryptomator")
	if err != nil || !ok {
		t.Fatalf("GetByPath: ok=%v err=%v", ok, err)
	}
	if n.NodeID != "n1" {
		t.Fatalf("nodeID = %q, want n1", n.NodeID)
	}

	if _, ok, _ := idx.GetByPath("nope"); ok {
		t.Fatal("expected miss for unknown path")
	}
}
