// Package config holds the sync daemon configuration (a JSON file stored alongside the local index).
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	ServerURL   string `json:"server_url"`
	DeviceToken string `json:"device_token"`
	SyncDir     string `json:"sync_dir"`
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "discodrive", "config.json"), nil
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	err = json.Unmarshal(b, &c)
	return c, err
}

func (c Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func StateDBPath(cfgPath string) string {
	return filepath.Join(filepath.Dir(cfgPath), "state.db")
}
