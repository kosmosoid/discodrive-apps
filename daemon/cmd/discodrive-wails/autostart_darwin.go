package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// applyOpenAtLogin registers/unregisters a macOS LaunchAgent that runs the app at login.
// When minimized, the LaunchAgent passes the --hidden flag so the app starts in the tray.
func applyOpenAtLogin(enabled, minimized bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.wails.discodrive-wails.plist")
	if !enabled {
		if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args := fmt.Sprintf("<string>%s</string>", exe)
	if minimized {
		args += fmt.Sprintf("<string>%s</string>", hiddenFlag)
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>com.wails.discodrive-wails</string>
  <key>ProgramArguments</key><array>%s</array>
  <key>RunAtLoad</key><true/>
</dict></plist>
`, args)
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(plistPath, []byte(plist), 0o644)
}
