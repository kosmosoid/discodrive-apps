package desktop

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestProfilePaths(t *testing.T) {
	dir, err := ProfileDir()
	if err != nil {
		t.Fatalf("ProfileDir: %v", err)
	}
	if !strings.HasSuffix(filepath.ToSlash(dir), "discodrive/desktop") {
		t.Fatalf("ProfileDir = %q, want it to end with discodrive/desktop", dir)
	}
	if got, want := filepath.Base(IndexDBPath(dir)), "index.db"; got != want {
		t.Fatalf("IndexDBPath base = %q, want %q", got, want)
	}
	if got, want := filepath.Base(ContentDir(dir)), "content"; got != want {
		t.Fatalf("ContentDir base = %q, want %q", got, want)
	}
}
