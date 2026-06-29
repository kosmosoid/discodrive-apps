//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"discodrive.org/daemon/internal/i18n"
)

const plistPath = "Library/LaunchAgents/org.discodrive.sync.plist"

var plistTmpl = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>org.discodrive.sync</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinPath}}</string>
        <string>tray</string>
        <string>--config</string>
        <string>{{.CfgPath}}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{{.LogPath}}</string>
    <key>StandardErrorPath</key>
    <string>{{.LogPath}}</string>
</dict>
</plist>
`))

func installAutostart(binPath, cfgPath, logPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	dest := filepath.Join(home, plistPath)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf(i18n.T("autostart_mkdir_launchagents"), err)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf(i18n.T("autostart_write_plist"), err)
	}
	data := struct{ BinPath, CfgPath, LogPath string }{binPath, cfgPath, logPath}
	if err := plistTmpl.Execute(f, data); err != nil {
		f.Close()
		return fmt.Errorf("render plist: %w", err)
	}
	f.Close()

	if out, err := exec.Command("launchctl", "load", dest).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load: %w\n%s", err, out)
	}
	fmt.Printf(i18n.T("autostart_installed")+"\n", dest)
	return nil
}

func uninstallAutostart() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	dest := filepath.Join(home, plistPath)
	// unload — ignore error if the agent is already unloaded
	_ = exec.Command("launchctl", "unload", dest).Run()
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf(i18n.T("autostart_remove_plist"), err)
	}
	fmt.Println(i18n.T("autostart_removed"))
	return nil
}
