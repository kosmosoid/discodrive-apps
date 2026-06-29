//go:build !darwin && !linux && !windows

package main

import (
	"fmt"

	"discodrive.org/daemon/internal/i18n"
)

func installAutostart(_, _, _ string) error {
	return fmt.Errorf(i18n.T("autostart_not_supported"))
}

func uninstallAutostart() error {
	return fmt.Errorf(i18n.T("autostart_not_supported"))
}
