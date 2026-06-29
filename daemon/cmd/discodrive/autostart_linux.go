//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"discodrive.org/daemon/internal/i18n"
)

const unitName = "discodrive.service"

var unitTmpl = template.Must(template.New("unit").Parse(`[Unit]
Description=Discodrive sync daemon
After=network.target

[Service]
ExecStart={{.BinPath}} tray --config {{.CfgPath}}
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
`))

func installAutostart(binPath, cfgPath, _ string) error {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("config dir: %w", err)
	}
	unitDir := filepath.Join(cfgDir, "systemd", "user")
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		return fmt.Errorf("mkdir systemd/user: %w", err)
	}
	dest := filepath.Join(unitDir, unitName)

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf(i18n.T("autostart_write_unit"), err)
	}
	data := struct{ BinPath, CfgPath string }{binPath, cfgPath}
	if err := unitTmpl.Execute(f, data); err != nil {
		f.Close()
		return fmt.Errorf("render unit: %w", err)
	}
	f.Close()

	if out, err := exec.Command("systemctl", "--user", "enable", "--now", unitName).CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl enable: %w\n%s", err, out)
	}
	fmt.Printf(i18n.T("autostart_installed")+"\n", dest)
	return nil
}

func uninstallAutostart() error {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("config dir: %w", err)
	}
	dest := filepath.Join(cfgDir, "systemd", "user", unitName)
	// disable+stop — ignore error if the unit is not loaded
	_ = exec.Command("systemctl", "--user", "disable", "--now", unitName).Run()
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf(i18n.T("autostart_remove_unit"), err)
	}
	fmt.Println(i18n.T("autostart_removed"))
	return nil
}
