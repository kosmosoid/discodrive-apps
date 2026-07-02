package mobile

import (
	"context"
	"encoding/json"
	"os"

	"discodrive.org/daemon/internal/index"
	"discodrive.org/daemon/internal/protocol"
	"discodrive.org/daemon/internal/safepath"
	"discodrive.org/daemon/internal/vault"
)

// Vault is a bindable, lazy view of one Cryptomator vault stored as a subfolder of the user's
// storage. Reads come from a local change-feed index + on-demand Download; writes go through the
// path-based sync API. Decrypted files are written to tmpDir (app-private) for the UI to open.
type Vault struct {
	v      *vault.Vault
	io     serverIO
	tmpDir string
}

type vaultEntry struct {
	Name            string `json:"name"`
	IsDir           bool   `json:"isDir"`
	DirID           string `json:"dirID"`
	FileStoragePath string `json:"fileStoragePath"`
}

// OpenVault opens the Cryptomator vault rooted at vaultRoot within the user's storage.
// indexDBPath — sqlite index (app-private); tmpDir — app-private dir for decrypted files;
// insecure — accept self-signed TLS. Wrong password → vault.ErrWrongPassword.
func OpenVault(serverURL, deviceToken, vaultRoot, password, indexDBPath, tmpDir string, insecure bool) (*Vault, error) {
	setInsecure(insecure)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return nil, err
	}
	idx, err := index.Open(indexDBPath)
	if err != nil {
		return nil, err
	}
	sio := serverIO{client: protocol.NewUnscoped(serverURL, deviceToken), idx: idx, root: vaultRoot}
	if err := pullChanges(context.Background(), sio.client, sio.idx); err != nil {
		idx.Close()
		return nil, err
	}
	v, err := vault.OpenWithSource(sio, password)
	if err != nil {
		idx.Close()
		return nil, err
	}
	return &Vault{v: v, io: sio, tmpDir: tmpDir}, nil
}

// Refresh re-pulls change-feed metadata into the index.
func (m *Vault) Refresh() error {
	return pullChanges(context.Background(), m.io.client, m.io.idx)
}

// List returns the children of dirID ("" = root) as a JSON array of vaultEntry.
func (m *Vault) List(dirID string) (string, error) {
	entries, err := m.v.ListDir(m.io, dirID)
	if err != nil {
		return "", err
	}
	out := make([]vaultEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, vaultEntry{Name: e.Name, IsDir: e.IsDir, DirID: e.DirID, FileStoragePath: e.FileStoragePath})
	}
	js, err := json.Marshal(out)
	return string(js), err
}

// OpenFile decrypts the file at fileStoragePath into tmpDir/plainName and returns the local path.
func (m *Vault) OpenFile(fileStoragePath, plainName string) (string, error) {
	data, err := m.v.ReadFile(m.io, fileStoragePath)
	if err != nil {
		return "", err
	}
	// plainName is a decrypted filename; contain it to tmpDir so a crafted name
	// (path separators / ..) can't write outside the app-private decrypt dir.
	dst, err := safepath.Join(m.tmpDir, plainName)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return "", err
	}
	return dst, nil
}

// WriteFile encrypts the local plaintext file and writes it as name into parentDirID, then refreshes.
func (m *Vault) WriteFile(parentDirID, name, localPath string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	if err := m.v.WriteFile(m.io, parentDirID, name, data); err != nil {
		return err
	}
	return m.Refresh()
}

// Mkdir creates subdirectory name in parentDirID, refreshes, and returns the new dirID.
func (m *Vault) Mkdir(parentDirID, name string) (string, error) {
	id, err := m.v.MakeDir(m.io, parentDirID, name)
	if err != nil {
		return "", err
	}
	if err := m.Refresh(); err != nil {
		return "", err
	}
	return id, nil
}

// Remove deletes file/dir name from parentDirID, then refreshes.
func (m *Vault) Remove(parentDirID, name string) error {
	if err := m.v.Remove(m.io, parentDirID, name); err != nil {
		return err
	}
	return m.Refresh()
}

// Close releases the index.
func (m *Vault) Close() error { return m.io.idx.Close() }
