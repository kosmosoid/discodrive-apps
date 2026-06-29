package desktop

import (
	"os"
	"path/filepath"

	"discodrive.org/daemon/internal/config"
	"discodrive.org/daemon/internal/index"
	"discodrive.org/daemon/internal/protocol"
)

// DesktopConfigPath is the desktop client's config.json inside the profile.
func DesktopConfigPath(profileDir string) string {
	return filepath.Join(profileDir, "config.json")
}

// SaveConfig writes the desktop config into the profile, creating the dir.
func SaveConfig(profileDir string, cfg config.Config) error {
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return err
	}
	return cfg.Save(DesktopConfigPath(profileDir))
}

// Open loads the profile's config and index and builds a controller backed by an
// unscoped server client (the browser navigates the whole vault). The returned
// index is handed back so the caller controls its lifetime. A missing/invalid
// config returns an error, signalling the UI to show pairing.
func Open(profileDir string) (*Controller, *index.Index, error) {
	cfg, err := config.Load(DesktopConfigPath(profileDir))
	if err != nil {
		return nil, nil, err
	}
	idx, err := index.Open(IndexDBPath(profileDir))
	if err != nil {
		return nil, nil, err
	}
	srv := protocol.NewUnscoped(cfg.ServerURL, cfg.DeviceToken)
	return NewController(srv, idx, ContentDir(profileDir)), idx, nil
}
