package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"discodrive.org/daemon/internal/i18n"
)

// detachAndExit re-launches itself in the background (without --detach), redirecting
// output to a log file next to the config, and releases the terminal. Minimal
// daemonization with no external supervisor; for persistent autostart use launchd/systemd.
func detachAndExit(cfgPath string) {
	exe, err := os.Executable()
	if err != nil {
		fatal(fmt.Sprintf(i18n.T("detach_no_exe"), err))
	}
	logPath := filepath.Join(filepath.Dir(cfgPath), "daemon.log")
	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fatal(fmt.Sprintf(i18n.T("detach_log_error"), logPath, err))
	}
	cmd := exec.Command(exe, "run", "--config", cfgPath)
	cmd.Stdout = lf
	cmd.Stderr = lf
	cmd.SysProcAttr = detachSysProcAttr() // platform-specific (Setsid on unix)
	if err := cmd.Start(); err != nil {
		fatal(fmt.Sprintf(i18n.T("detach_start_error"), err))
	}
	fmt.Printf(i18n.T("detach_started"), cmd.Process.Pid, logPath, cmd.Process.Pid)
}
