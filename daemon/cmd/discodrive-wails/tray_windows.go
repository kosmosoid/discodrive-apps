//go:build windows

package main

import _ "embed"

// On Windows fyne/systray requires ICO bytes — a PNG renders as an empty tray slot.
//
//go:embed icon.ico
var trayIcon []byte
