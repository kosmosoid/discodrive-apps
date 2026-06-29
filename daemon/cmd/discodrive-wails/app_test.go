package main

import (
	"strings"
	"testing"
)

func TestNameTooLong(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"short ascii", "a.pdf", false},
		{"exactly 255 bytes", strings.Repeat("a", 255), false},
		{"256 bytes", strings.Repeat("a", 256), true},
		// Cyrillic is 2 bytes/char in UTF-8: 128 chars = 256 bytes > 255.
		{"128 cyrillic chars (256 bytes)", strings.Repeat("я", 128), true},
		{"127 cyrillic chars (254 bytes)", strings.Repeat("я", 127), false},
		// The real-world file that triggered the server 500 (258 bytes).
		{"reported pdf (258 bytes)",
			"Заявление_о_прекраении_предпринимательской_деятельности,_в_отношении_которой_применялась_патентная_система_налогообложения_(форма_N_26.5-4).pdf",
			true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := nameTooLong(c.in); got != c.want {
				t.Fatalf("nameTooLong(%d bytes) = %v, want %v", len(c.in), got, c.want)
			}
		})
	}
}

func TestLinuxDesktopEntry(t *testing.T) {
	out := linuxDesktopEntry("/home/u/.local/bin/My App", "")
	for _, want := range []string{
		"[Desktop Entry]",
		"Type=Application",
		`Exec="/home/u/.local/bin/My App"`, // quoted so spaces survive
		"X-GNOME-Autostart-enabled=true",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("desktop entry missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, hiddenFlag) {
		t.Fatalf("entry should not carry %s when flag is empty:\n%s", hiddenFlag, out)
	}

	withFlag := linuxDesktopEntry("/home/u/app", hiddenFlag)
	if !strings.Contains(withFlag, `Exec="/home/u/app" --hidden`) {
		t.Fatalf("entry missing flagged exec:\n%s", withFlag)
	}
}
