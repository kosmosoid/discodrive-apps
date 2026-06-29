//go:build windows

package main

import (
	"fmt"

	"golang.org/x/sys/windows/registry"

	"discodrive.org/daemon/internal/i18n"
)

// runKeyPath is the per-user autostart key: values here run on login without
// administrator rights.
const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

// runValueName is the name of our value under the Run key.
const runValueName = "Discodrive"

func installAutostart(binPath, cfgPath, _ string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open Run key: %w", err)
	}
	defer k.Close()
	if err := k.SetStringValue(runValueName, windowsRunKeyValue(binPath, cfgPath)); err != nil {
		return fmt.Errorf("set Run value: %w", err)
	}
	fmt.Printf(i18n.T("autostart_installed")+"\n", runKeyPath+`\`+runValueName)
	return nil
}

func uninstallAutostart() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			fmt.Println(i18n.T("autostart_removed"))
			return nil
		}
		return fmt.Errorf("open Run key: %w", err)
	}
	defer k.Close()
	if err := k.DeleteValue(runValueName); err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("delete Run value: %w", err)
	}
	fmt.Println(i18n.T("autostart_removed"))
	return nil
}
