//go:build !windows

package main

import "syscall"

// detachSysProcAttr detaches the background process from the controlling terminal
// (new session) so that closing the console does not kill the daemon.
func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
