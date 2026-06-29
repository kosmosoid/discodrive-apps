//go:build !darwin && !windows && !linux

package main

// applyOpenAtLogin is a no-op on platforms without an autostart implementation.
func applyOpenAtLogin(enabled, minimized bool) error { return nil }
