//go:build windows

package main

import "syscall"

// detachSysProcAttr starts the background process in a new process group, detached from the console.
func detachSysProcAttr() *syscall.SysProcAttr {
	const (
		createNewProcessGroup = 0x00000200
		detachedProcess       = 0x00000008
	)
	return &syscall.SysProcAttr{CreationFlags: createNewProcessGroup | detachedProcess}
}
