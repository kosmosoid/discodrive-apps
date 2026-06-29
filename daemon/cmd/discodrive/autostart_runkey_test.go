package main

import "testing"

func TestWindowsRunKeyValueQuotesPathsWithSpaces(t *testing.T) {
	got := windowsRunKeyValue(
		`C:\Program Files\Discodrive\discodrive.exe`,
		`C:\Users\Jane Doe\AppData\Roaming\discodrive\config.json`,
	)
	want := `"C:\Program Files\Discodrive\discodrive.exe" tray --config "C:\Users\Jane Doe\AppData\Roaming\discodrive\config.json"`
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}
