package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/term"

	"discodrive.org/daemon/internal/i18n"
)

// shouldDetach decides whether a run/tray command re-launches itself in the
// background. Interactive console runs detach by default (so the terminal is
// released); service runs (launchd/systemd — no tty) stay in the foreground, which
// their supervisors require. --detach forces, --foreground opts out.
func shouldDetach(detachFlag, foregroundFlag, stdoutIsTTY bool) bool {
	if detachFlag {
		return true
	}
	return stdoutIsTTY && !foregroundFlag
}

// stdoutIsTerminal reports whether stdout is an interactive terminal. A real
// isatty check (not ModeCharDevice): /dev/null is a char device but not a tty,
// so service runs with null-redirected output are not mistaken for consoles.
func stdoutIsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// maybeDaemonize arbitrates a run/tray start: refuses when another instance
// already holds the profile lock (exit 0 — not a crash, so launchd's
// SuccessfulExit=false KeepAlive does not respawn us), detaches when started
// from a console, and otherwise hands the held lock back to the foreground
// process. proceed=false means the caller must return immediately.
func maybeDaemonize(subcommand, cfgPath string, detach, foreground bool) (release func(), proceed bool) {
	release, ok, err := acquireLock(cfgPath)
	if err != nil {
		fatal(fmt.Sprintf(i18n.T("lock_error"), err))
	}
	if !ok {
		fmt.Fprintln(os.Stderr, i18n.T("already_running"))
		return nil, false
	}
	if shouldDetach(detach, foreground, stdoutIsTerminal()) {
		release() // the detached child re-acquires it
		detachAndExit(subcommand, cfgPath)
		return nil, false
	}
	return release, true
}

// detachAndExit re-launches the given subcommand in the background with
// --foreground (so the child does not detach again), redirecting output to a log
// file next to the config, and releases the terminal. Minimal daemonization with
// no external supervisor; for persistent autostart use launchd/systemd.
func detachAndExit(subcommand, cfgPath string) {
	exe, err := os.Executable()
	if err != nil {
		fatal(fmt.Sprintf(i18n.T("detach_no_exe"), err))
	}
	logPath := filepath.Join(filepath.Dir(cfgPath), "daemon.log")
	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fatal(fmt.Sprintf(i18n.T("detach_log_error"), logPath, err))
	}
	cmd := exec.Command(exe, subcommand, "--config", cfgPath, "--foreground")
	cmd.Stdout = lf
	cmd.Stderr = lf
	cmd.SysProcAttr = detachSysProcAttr() // platform-specific (Setsid on unix)
	if err := cmd.Start(); err != nil {
		fatal(fmt.Sprintf(i18n.T("detach_start_error"), err))
	}
	fmt.Printf(i18n.T("detach_started"), cmd.Process.Pid, logPath, cmd.Process.Pid)
}
