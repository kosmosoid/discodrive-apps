//go:build !windows

package main

import _ "embed"

// On macOS and Linux fyne/systray accepts a PNG icon.
//
//go:embed appicon.png
var trayIcon []byte
