package main

import (
	"os"
	"path/filepath"
)

// acquireLock takes an exclusive advisory lock on daemon.lock next to the config,
// so only one daemon instance (run or tray) per profile can be active. Returns
// ok=false when another instance already holds the lock. The lock is released by
// the returned func and implicitly when the process dies (no stale-pid problem).
func acquireLock(cfgPath string) (release func(), ok bool, err error) {
	p := filepath.Join(filepath.Dir(cfgPath), "daemon.lock")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return nil, false, err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false, err
	}
	locked, err := flockExclusive(f)
	if err != nil {
		f.Close()
		return nil, false, err
	}
	if !locked {
		f.Close()
		return nil, false, nil
	}
	return func() { f.Close() }, true, nil
}
