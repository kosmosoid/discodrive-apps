// Package vault — encrypt.go: file tree encryption (Cryptomator format 8 flatten).
package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// shorteningThreshold is the encrypted name length limit (including the ".c9r" suffix).
// When exceeded, .c9s shortening is applied (Cryptomator format 8).
const shorteningThreshold = 220

// EncryptTree encrypts the plaintext tree at srcDir into a Cryptomator-compatible vault at vaultDir.
// Root dirID = "" (empty string). Creates d/XX/YYY/... structure.
// Names longer than 220 characters are automatically shortened via .c9s.
func (v *Vault) EncryptTree(srcDir, vaultDir string) error {
	return v.encryptDir(srcDir, vaultDir, "")
}

// encryptDir recursively encrypts the contents of dirPath with the given dirID into vaultDir.
func (v *Vault) encryptDir(dirPath, vaultDir, dirID string) error {
	// Compute the storage path for this directory
	hashPath, err := v.DirIdHash(dirID)
	if err != nil {
		return fmt.Errorf("vault: DirIdHash(%q): %w", dirID, err)
	}
	storagePath := filepath.Join(vaultDir, hashPath)
	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		return fmt.Errorf("vault: creating storage directory %q: %w", storagePath, err)
	}

	// Write dirid.c9r — the encrypted ID of this directory
	if err := v.writeDirIDFile(storagePath, dirID); err != nil {
		return fmt.Errorf("vault: writing dirid.c9r for dirID=%q: %w", dirID, err)
	}

	// Walk srcDir
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("vault: ReadDir %q: %w", dirPath, err)
	}

	for _, e := range entries {
		name := e.Name()
		encName, err := v.EncryptName(name, dirID)
		if err != nil {
			return fmt.Errorf("vault: encrypting name %q: %w", name, err)
		}

		if len(encName) > shorteningThreshold {
			// .c9s shortening: create <shortened>.c9s directory with name.c9s and contents.c9r/dir.c9r
			shortened := shortenedName(encName)
			c9sPath := filepath.Join(storagePath, shortened)
			if err := os.MkdirAll(c9sPath, 0o755); err != nil {
				return fmt.Errorf("vault: creating .c9s directory %q: %w", c9sPath, err)
			}
			// name.c9s — full encrypted name
			if err := os.WriteFile(filepath.Join(c9sPath, "name.c9s"), []byte(encName), 0o644); err != nil {
				return fmt.Errorf("vault: writing name.c9s for %q: %w", name, err)
			}

			if e.IsDir() {
				// dir.c9r — plaintext subDirID
				subDirID := uuid.NewString()
				if err := os.WriteFile(filepath.Join(c9sPath, "dir.c9r"), []byte(subDirID), 0o644); err != nil {
					return fmt.Errorf("vault: writing dir.c9r in .c9s for %q: %w", name, err)
				}
				subSrcPath := filepath.Join(dirPath, name)
				if err := v.encryptDir(subSrcPath, vaultDir, subDirID); err != nil {
					return err
				}
			} else {
				// contents.c9r — encrypted content
				contentsPath := filepath.Join(c9sPath, "contents.c9r")
				if err := v.encryptFile(filepath.Join(dirPath, name), contentsPath); err != nil {
					return fmt.Errorf("vault: encrypting contents.c9r for %q: %w", name, err)
				}
			}
		} else if e.IsDir() {
			// Subdirectory with short name: create <encName>/ with dir.c9r
			subDirID := uuid.NewString()
			encDirPath := filepath.Join(storagePath, encName)
			if err := os.MkdirAll(encDirPath, 0o755); err != nil {
				return fmt.Errorf("vault: creating directory %q: %w", encDirPath, err)
			}
			// dir.c9r holds the plaintext subDirID (NOT encrypted — reader treats it as plaintext)
			dirCPath := filepath.Join(encDirPath, "dir.c9r")
			if err := os.WriteFile(dirCPath, []byte(subDirID), 0o644); err != nil {
				return fmt.Errorf("vault: writing dir.c9r for %q: %w", name, err)
			}
			// Recursively encrypt the subdirectory contents
			subSrcPath := filepath.Join(dirPath, name)
			if err := v.encryptDir(subSrcPath, vaultDir, subDirID); err != nil {
				return err
			}
		} else {
			// File with short name: <encName> (already contains .c9r)
			encFilePath := filepath.Join(storagePath, encName)
			if err := v.encryptFile(filepath.Join(dirPath, name), encFilePath); err != nil {
				return fmt.Errorf("vault: encrypting file %q → %q: %w", name, encFilePath, err)
			}
		}
	}
	return nil
}

// writeDirIDFile encrypts dirID as the content of dirid.c9r in storagePath.
func (v *Vault) writeDirIDFile(storagePath, dirID string) error {
	path := filepath.Join(storagePath, "dirid.c9r")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return v.EncryptContent(f, strings.NewReader(dirID))
}

// encryptFile encrypts a single file from srcPath to dstPath.
func encryptFile(v *Vault, srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	return v.EncryptContent(dst, src)
}

// encryptFile is a convenience method wrapper for use inside encryptDir.
func (v *Vault) encryptFile(srcPath, dstPath string) error {
	return encryptFile(v, srcPath, dstPath)
}
