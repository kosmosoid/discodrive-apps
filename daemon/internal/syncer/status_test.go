package syncer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWriteReadRoundtrip verifies that writeStatus→ReadStatus correctly round-trips.
func TestWriteReadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	s := &Syncer{statusPath: filepath.Join(dir, "status.json")}

	want := Status{
		State:     StateOffline,
		LastError: "connection refused",
		LastSync:  time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	s.writeStatus(want)

	got, err := ReadStatus(s.statusPath)
	if err != nil {
		t.Fatalf("ReadStatus: %v", err)
	}
	if got.State != want.State {
		t.Errorf("State: got %q, want %q", got.State, want.State)
	}
	if got.LastError != want.LastError {
		t.Errorf("LastError: got %q, want %q", got.LastError, want.LastError)
	}
	if !got.LastSync.Equal(want.LastSync) {
		t.Errorf("LastSync: got %v, want %v", got.LastSync, want.LastSync)
	}
	if got.Pid <= 0 {
		t.Errorf("Pid is not set: %d", got.Pid)
	}
	if got.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is not set")
	}
}

// TestAtomicWrite verifies that the temp file is cleaned up after a successful writeStatus.
func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	s := &Syncer{statusPath: filepath.Join(dir, "status.json")}

	s.writeStatus(Status{State: StateIdle, LastSync: time.Now()})

	// main file must exist
	if _, err := os.Stat(s.statusPath); err != nil {
		t.Fatalf("status.json was not created: %v", err)
	}
	// temp file must be removed
	tmp := s.statusPath + ".tmp"
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Errorf("tmp file was not removed: %v", err)
	}
}

// TestReadStatusMissing verifies that ReadStatus returns an error when the file is absent.
func TestReadStatusMissing(t *testing.T) {
	_, err := ReadStatus(filepath.Join(t.TempDir(), "no-such.json"))
	if err == nil {
		t.Error("expected an error for a missing file")
	}
}
