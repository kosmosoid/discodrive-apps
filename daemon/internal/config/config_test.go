package config

import (
	"path/filepath"
	"testing"
)

func TestConfigRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "config.json")
	c := Config{ServerURL: "https://x", DeviceToken: "kfd_1", SyncDir: "/data/sync"}
	if err := c.Save(p); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if got != c {
		t.Fatalf("round-trip: %+v != %+v", got, c)
	}
	if db := StateDBPath(p); filepath.Dir(db) != filepath.Dir(p) || filepath.Base(db) != "state.db" {
		t.Fatalf("StateDBPath: %q", db)
	}
}
