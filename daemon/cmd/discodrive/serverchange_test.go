package main

import (
	"path/filepath"
	"testing"

	"discodrive.org/daemon/internal/config"
	"discodrive.org/daemon/internal/index"
)

// writeProfile lays out a config.json + state.db as if the daemon had synced
// against oldServer, then re-points config.json at newServer.
func writeProfile(t *testing.T, oldServer, newServer string) (cfgPath string) {
	t.Helper()
	dir := t.TempDir()
	cfgPath = filepath.Join(dir, "config.json")
	idx, err := index.Open(config.StateDBPath(cfgPath))
	if err != nil {
		t.Fatal(err)
	}
	_ = idx.Put(index.Node{NodeID: "n1", RelPath: "stale.txt", Version: 1})
	_ = idx.SetServerURL(oldServer)
	idx.Close()
	cfg := config.Config{ServerURL: newServer, DeviceToken: "t", SyncDir: filepath.Join(dir, "sync")}
	if err := cfg.Save(cfgPath); err != nil {
		t.Fatal(err)
	}
	return cfgPath
}

func TestBuildSyncerWipesIndexOnServerChange(t *testing.T) {
	// 127.0.0.1:1 → connection refused, so the language fetch fails fast (ignored).
	cfgPath := writeProfile(t, "https://old.test", "http://127.0.0.1:1")
	_, cleanup, _, err := buildSyncer(cfgPath)
	if err != nil {
		t.Fatalf("buildSyncer: %v", err)
	}
	cleanup()
	idx, err := index.Open(config.StateDBPath(cfgPath))
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	if _, ok, _ := idx.Get("n1"); ok {
		t.Fatal("index from the old server must be wiped on server change")
	}
	if u, _ := idx.ServerURL(); u != "http://127.0.0.1:1" {
		t.Fatalf("index must be stamped with the new server, got %q", u)
	}
}

func TestBuildSyncerKeepsIndexOnSameServer(t *testing.T) {
	cfgPath := writeProfile(t, "http://127.0.0.1:1", "http://127.0.0.1:1")
	_, cleanup, _, err := buildSyncer(cfgPath)
	if err != nil {
		t.Fatalf("buildSyncer: %v", err)
	}
	cleanup()
	idx, err := index.Open(config.StateDBPath(cfgPath))
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	if _, ok, _ := idx.Get("n1"); !ok {
		t.Fatal("same server — index must survive")
	}
}
