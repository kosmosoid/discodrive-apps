//go:build !tray

package main

import "discodrive.org/daemon/internal/i18n"

func cmdTray(_ []string) {
	fatal(i18n.T("tray_no_tray_build"))
}
