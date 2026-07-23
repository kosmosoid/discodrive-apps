//go:build !windows

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

// flockExclusive tries a non-blocking exclusive flock; false means someone else
// holds it. The kernel releases the lock automatically when the fd is closed
// (including process death), so no stale lock files survive a crash.
func flockExclusive(f *os.File) (bool, error) {
	err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err == unix.EWOULDBLOCK {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
