package safepath

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestJoin_AllowsLegitimateNestedPaths(t *testing.T) {
	root := t.TempDir()
	cases := []string{"file.txt", "sub/dir/file.txt", "a/b/c/d.bin", "."}
	for _, rel := range cases {
		got, err := Join(root, rel)
		if err != nil {
			t.Fatalf("Join(%q, %q) unexpected error: %v", root, rel, err)
		}
		if !strings.HasPrefix(got, filepath.Clean(root)) {
			t.Fatalf("Join(%q, %q) = %q, escaped root", root, rel, got)
		}
	}
}

func TestJoin_RejectsTraversal(t *testing.T) {
	root := t.TempDir()
	cases := []string{"../evil.txt", "../../etc/passwd", "sub/../../escape", "a/b/../../../out"}
	for _, rel := range cases {
		if _, err := Join(root, rel); err == nil {
			t.Fatalf("Join(%q, %q) should have been rejected as an escape", root, rel)
		}
	}
}

func TestJoin_NeutralizesAbsolutePaths(t *testing.T) {
	root := t.TempDir()
	// An absolute rel_path must be contained under root, never resolve to the real /etc.
	abs := "/etc/passwd"
	if runtime.GOOS == "windows" {
		abs = `C:\Windows\system32\drivers\etc\hosts`
	}
	got, err := Join(root, abs)
	if err != nil {
		t.Fatalf("Join(%q, %q) unexpected error: %v", root, abs, err)
	}
	if !strings.HasPrefix(got, filepath.Clean(root)) {
		t.Fatalf("absolute rel_path escaped root: %q", got)
	}
}

func TestJoin_RejectsSymlinkedComponent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	// Plant a symlinked directory inside root pointing outside it.
	link := filepath.Join(root, "evil")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unsupported here: %v", err)
	}
	if _, err := Join(root, "evil/loot.txt"); err == nil {
		t.Fatal("Join must reject a path through a symlinked directory")
	}
	// A sibling real path is still fine.
	if _, err := Join(root, "ok/file.txt"); err != nil {
		t.Fatalf("normal path should be allowed: %v", err)
	}
}

// The vault opens into a temp dir OUTSIDE its storage; the guard must use that temp
// dir as the root and allow every legitimate file under it.
func TestJoin_TempDirRootOutsideStorage(t *testing.T) {
	tmp := t.TempDir() // stands in for the ddvopen-* decrypt dir
	for _, rel := range []string{"d/masterkey.cryptomator", "d/AB/CDEF/file.c9r", "."} {
		if _, err := Join(tmp, rel); err != nil {
			t.Fatalf("legitimate vault path %q under temp root rejected: %v", rel, err)
		}
	}
	if _, err := Join(tmp, "../../../Users/victim/.zshrc"); err == nil {
		t.Fatal("traversal out of the temp root must be rejected")
	}
}
