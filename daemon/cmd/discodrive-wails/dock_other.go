//go:build !darwin

package main

// setDockVisible is a no-op off macOS (dock is a macOS concept).
func setDockVisible(bool) {}
