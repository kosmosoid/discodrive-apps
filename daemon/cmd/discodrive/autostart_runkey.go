package main

// windowsRunKeyValue builds the command line for a Windows registry value
// (the ...\CurrentVersion\Run key): the daemon in tray mode with an explicit
// config path. Both path substitutions are wrapped in double quotes so that
// spaces in paths (e.g. "C:\Program Files\...") are preserved. This file has no
// build tag — the function and its test must compile on any OS; only the
// Windows build calls it (autostart_windows.go).
func windowsRunKeyValue(binPath, cfgPath string) string {
	return `"` + binPath + `" tray --config "` + cfgPath + `"`
}
