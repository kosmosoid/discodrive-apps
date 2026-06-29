package main

import "fmt"

// hiddenFlag is the CLI flag the autostart entry passes so an auto-launched instance
// starts minimized to the tray; manual launches omit it and show the window. Parsed in
// main(); appended to the per-platform autostart command when StartMinimized is on.
const hiddenFlag = "--hidden"

// linuxDesktopEntry builds an XDG autostart .desktop file that launches exe at login.
// Kept in its own (untagged) file so it can be unit-tested on any host; it is only used
// by the linux build. The Exec path is double-quoted per the Desktop Entry spec so paths
// with spaces are handled; the flag (if any) is appended outside the quotes.
func linuxDesktopEntry(exe, flag string) string {
	exec := fmt.Sprintf(`"%s"`, exe)
	if flag != "" {
		exec += " " + flag
	}
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=DiscoDrive
Exec=%s
X-GNOME-Autostart-enabled=true
`, exec)
}
