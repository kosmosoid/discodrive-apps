package main

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

// runKeyPath is the per-user "run at login" registry key; runValueName is our entry in it.
const (
	runKeyPath   = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValueName = "DiscoDrive"
)

// applyOpenAtLogin adds/removes an HKCU\...\Run value so Windows launches the app at
// login. When minimized, the command gets the --hidden flag so it starts in the tray.
func applyOpenAtLogin(enabled, minimized bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if !enabled {
		if err := k.DeleteValue(runValueName); err != nil && err != registry.ErrNotExist {
			return err
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// Quote the path so a Program Files / user-profile path with spaces is parsed as one arg.
	cmd := `"` + exe + `"`
	if minimized {
		cmd += " " + hiddenFlag
	}
	return k.SetStringValue(runValueName, cmd)
}
