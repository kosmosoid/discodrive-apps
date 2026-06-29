package desktop

import (
	"path/filepath"
	"testing"

	"discodrive.org/daemon/internal/config"
)

func TestSaveAndOpenProfile(t *testing.T) {
	profile := t.TempDir()
	cfg := config.Config{ServerURL: "https://example.test", DeviceToken: "kfd_x"}
	if err := SaveConfig(profile, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if filepath.Base(DesktopConfigPath(profile)) != "config.json" {
		t.Fatalf("DesktopConfigPath base = %q", filepath.Base(DesktopConfigPath(profile)))
	}

	ctrl, idx, err := Open(profile)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer idx.Close()
	if ctrl == nil {
		t.Fatal("controller is nil")
	}
}

func TestOpenMissingConfigErrors(t *testing.T) {
	if _, _, err := Open(t.TempDir()); err == nil {
		t.Fatal("Open with no config should error (drives UI to pairing)")
	}
}
