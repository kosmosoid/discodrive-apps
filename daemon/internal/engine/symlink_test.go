package engine

import (
	"os"
	"path/filepath"
	"testing"

	"discodrive.org/daemon/internal/index"
)

// A symlink inside the sync folder pointing at a file outside it must NOT be
// detected/uploaded (otherwise e.g. ~/.ssh/id_rsa could leak to the server).
func TestDetectLocal_SkipsSymlinkFile(t *testing.T) {
	root := t.TempDir()
	idx, _ := index.Open(filepath.Join(t.TempDir(), "s.db"))
	defer idx.Close()

	secret := filepath.Join(t.TempDir(), "secret.key")
	os.WriteFile(secret, []byte("PRIVATE"), 0o600)
	if err := os.Symlink(secret, filepath.Join(root, "link.key")); err != nil {
		t.Skipf("symlinks unsupported here: %v", err)
	}
	os.WriteFile(filepath.Join(root, "real.txt"), []byte("ok"), 0o644)

	got, err := New(nil, idx, root).DetectLocal()
	if err != nil {
		t.Fatal(err)
	}
	var sawReal bool
	for _, c := range got {
		if c.RelPath == "link.key" {
			t.Fatalf("symlink must not be synced (would leak %s)", secret)
		}
		if c.RelPath == "real.txt" {
			sawReal = true
		}
	}
	if !sawReal {
		t.Fatal("real.txt should still be detected")
	}
}

// A pull targeting a path that traverses a symlinked directory must be rejected,
// so file ops can't write/delete outside the sync folder.
func TestAbs_RejectsSymlinkDirComponent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "evil")); err != nil {
		t.Skipf("symlinks unsupported here: %v", err)
	}
	e := New(nil, nil, root)

	if _, err := e.abs("evil/passwd"); err == nil {
		t.Fatal("abs must reject a path through a symlinked directory")
	}
	if _, err := e.abs("ok/file.txt"); err != nil {
		t.Fatalf("normal (non-symlink) path should be allowed: %v", err)
	}
}
