package engine

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"discodrive.org/daemon/internal/index"
)

type fakeSink struct {
	pushed  map[string]*int64
	dirs    []string
	deleted []string
}

func newFakeSink() *fakeSink { return &fakeSink{pushed: map[string]*int64{}} }

func (s *fakeSink) PushFile(_ context.Context, rel string, base *int64, r io.Reader) (RemoteNode, bool, error) {
	b, _ := io.ReadAll(r)
	s.pushed[rel] = base
	return RemoteNode{NodeID: "srv-" + rel, Version: 10, Hash: hashOf(b)}, false, nil
}
func (s *fakeSink) EnsureDir(_ context.Context, rel string) (RemoteNode, error) {
	s.dirs = append(s.dirs, rel)
	return RemoteNode{NodeID: "dir-" + rel, Version: 1}, nil
}
func (s *fakeSink) DeleteRemote(_ context.Context, rel string) error {
	s.deleted = append(s.deleted, rel)
	return nil
}

func TestPushLocalCreate(t *testing.T) {
	root := t.TempDir()
	idx, _ := index.Open(filepath.Join(t.TempDir(), "s.db"))
	defer idx.Close()
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("hi"), 0o644)
	e := New(nil, idx, root)
	sink := newFakeSink()

	if err := e.PushLocal(context.Background(), sink); err != nil {
		t.Fatal(err)
	}
	if base, ok := sink.pushed["a.txt"]; !ok || base != nil {
		t.Fatalf("expected push a.txt with base=nil, got ok=%v base=%v", ok, base)
	}
	got, _ := e.DetectLocal()
	if len(got) != 0 {
		t.Fatalf("after push, DetectLocal must be empty (echo), got %+v", got)
	}
}

func TestPushLocalUpdateSendsBaseVersion(t *testing.T) {
	root := t.TempDir()
	idx, _ := index.Open(filepath.Join(t.TempDir(), "s.db"))
	defer idx.Close()
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("v2"), 0o644)
	idx.Put(index.Node{NodeID: "n1", RelPath: "a.txt", Version: 5, ContentHash: hashOf([]byte("v1")), Size: 2})
	e := New(nil, idx, root)
	sink := newFakeSink()
	if err := e.PushLocal(context.Background(), sink); err != nil {
		t.Fatal(err)
	}
	base := sink.pushed["a.txt"]
	if base == nil || *base != 5 {
		t.Fatalf("expected base=5, got %v", base)
	}
}

func TestPushLocalDelete(t *testing.T) {
	root := t.TempDir()
	idx, _ := index.Open(filepath.Join(t.TempDir(), "s.db"))
	defer idx.Close()
	idx.Put(index.Node{NodeID: "n1", RelPath: "gone.txt", Version: 3, ContentHash: hashOf([]byte("x")), Size: 1})
	e := New(nil, idx, root)
	sink := newFakeSink()
	if err := e.PushLocal(context.Background(), sink); err != nil {
		t.Fatal(err)
	}
	if len(sink.deleted) != 1 || sink.deleted[0] != "gone.txt" {
		t.Fatalf("expected delete gone.txt, got %+v", sink.deleted)
	}
	if _, ok, _ := idx.Get("n1"); ok {
		t.Fatalf("node must be removed from the index")
	}
}
