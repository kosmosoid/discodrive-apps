//go:build !tray

package main

import (
	"fmt"
	"os"

	"discodrive.org/daemon/internal/i18n"
)

// cmdTray in a no-tray build logs the notice once and falls back to the headless
// sync loop. It must NOT exit: the autostart agent (launchd KeepAlive / run key)
// restarts the process on exit, which used to flood daemon.log with this message
// in an endless restart loop.
func cmdTray(args []string) {
	fmt.Fprintln(os.Stderr, i18n.T("tray_no_tray_build"))
	cmdRun(args)
}
