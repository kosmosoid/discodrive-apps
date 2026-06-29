// Package vault — osio.go: local-filesystem implementation of Sink.
package vault

import (
	"os"
	"path/filepath"
)

// OSIO implements Sink over the local filesystem rooted at Root.
// storagePath arguments are slash-separated paths relative to Root.
type OSIO struct{ Root string }

func (o OSIO) abs(storagePath string) string {
	return filepath.Join(o.Root, filepath.FromSlash(storagePath))
}

func (o OSIO) ListDir(storagePath string) ([]Entry, error) {
	des, err := os.ReadDir(o.abs(storagePath))
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(des))
	for _, d := range des {
		out = append(out, Entry{Name: d.Name(), IsDir: d.IsDir()})
	}
	return out, nil
}

func (o OSIO) ReadFile(storagePath string) ([]byte, error) {
	return os.ReadFile(o.abs(storagePath))
}

func (o OSIO) MakeDir(storagePath string) error {
	return os.MkdirAll(o.abs(storagePath), 0o755)
}

func (o OSIO) WriteFile(storagePath string, data []byte) error {
	p := o.abs(storagePath)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func (o OSIO) Remove(storagePath string) error {
	return os.RemoveAll(o.abs(storagePath))
}
