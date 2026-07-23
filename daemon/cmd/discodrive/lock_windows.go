//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// flockExclusive tries a non-blocking exclusive LockFileEx; false means another
// process holds it. Windows releases the lock when the handle is closed
// (including process death), so no stale lock files survive a crash.
func flockExclusive(f *os.File) (bool, error) {
	ol := new(windows.Overlapped)
	err := windows.LockFileEx(windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, ol)
	if err == windows.ERROR_LOCK_VIOLATION {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
