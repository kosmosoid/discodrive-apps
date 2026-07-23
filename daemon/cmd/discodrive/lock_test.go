package main

import (
	"path/filepath"
	"testing"
)

func TestAcquireLockIsExclusive(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")

	release, ok, err := acquireLock(cfgPath)
	if err != nil || !ok {
		t.Fatalf("first acquire: ok=%v err=%v", ok, err)
	}

	// A second daemon instance must not get the lock.
	if _, ok2, err2 := acquireLock(cfgPath); err2 != nil || ok2 {
		t.Fatalf("second acquire must be refused: ok=%v err=%v", ok2, err2)
	}

	release()

	// After the first instance exits, the lock is free again.
	release3, ok3, err3 := acquireLock(cfgPath)
	if err3 != nil || !ok3 {
		t.Fatalf("acquire after release: ok=%v err=%v", ok3, err3)
	}
	release3()
}
