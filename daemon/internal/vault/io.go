// Package vault — io.go: storage I/O abstraction for lazy vault access.
package vault

// Entry is one raw storage entry (encrypted name) inside a vault storage directory.
type Entry struct {
	Name  string
	IsDir bool
}

// Source reads encrypted vault data by storage path.
// storagePath is slash-separated and relative to the vault root
// (e.g. "masterkey.cryptomator", "d/2R/YAZT.../name.c9r").
type Source interface {
	ListDir(storagePath string) ([]Entry, error)
	ReadFile(storagePath string) ([]byte, error)
}

// Sink is a writable Source.
type Sink interface {
	Source
	MakeDir(storagePath string) error
	WriteFile(storagePath string, data []byte) error
	Remove(storagePath string) error
}
