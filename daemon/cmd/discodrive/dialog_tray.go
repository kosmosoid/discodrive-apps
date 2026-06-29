//go:build tray

package main

import (
	"os/exec"
	"runtime"
	"strings"
)

// promptText shows a native input dialog (hidden for passwords) and returns the entered value.
// macOS uses osascript, Linux uses zenity. On error or cancel returns ("", false).
func promptText(title, prompt string, hidden bool) (string, bool) {
	switch runtime.GOOS {
	case "darwin":
		hiddenArg := ""
		if hidden {
			hiddenArg = " with hidden answer"
		}
		script := `display dialog "` + escapeAS(prompt) + `" default answer "" with title "` + escapeAS(title) + `"` + hiddenArg
		out, err := exec.Command("osascript", "-e", script).Output()
		if err != nil {
			return "", false // cancel or error
		}
		// output format: button returned:OK, text returned:<value>
		s := string(out)
		i := strings.Index(s, "text returned:")
		if i < 0 {
			return "", false
		}
		return strings.TrimRight(s[i+len("text returned:"):], "\n"), true
	default: // linux
		args := []string{"--entry", "--title=" + title, "--text=" + prompt}
		if hidden {
			args = []string{"--password", "--title=" + title}
		}
		out, err := exec.Command("zenity", args...).Output()
		if err != nil {
			return "", false
		}
		return strings.TrimRight(string(out), "\n"), true
	}
}

func escapeAS(s string) string { return strings.ReplaceAll(s, `"`, `\"`) }

// notify shows a native desktop notification (best-effort).
func notify(title, msg string) {
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("osascript", "-e", `display notification "`+escapeAS(msg)+`" with title "`+escapeAS(title)+`"`).Start()
	default:
		_ = exec.Command("notify-send", title, msg).Start()
	}
}
