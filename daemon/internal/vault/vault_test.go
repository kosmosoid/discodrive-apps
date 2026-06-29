package vault

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Test 1: official RFC 3394 AES Key Wrap test vector (256-bit KEK, 256-bit plaintext)
// Source: https://www.rfc-editor.org/rfc/rfc3394
// KEK:  000102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F
// PT:   00112233445566778899AABBCCDDEEFF000102030405060708090A0B0C0D0E0F
// CT:   28C9F404C4B810F4CBCCB35CFB87F8263F5786E2D80ED326CBC7F0E71A99F43BFB988B9B7A02DD21
func TestAESKWRFC3394Vector(t *testing.T) {
	kek := []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
	}
	pt := []byte{
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
		0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff,
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
	}
	// Official RFC 3394 output (40 bytes = 5 semiblocks):
	wantCT := []byte{
		0x28, 0xc9, 0xf4, 0x04, 0xc4, 0xb8, 0x10, 0xf4,
		0xcb, 0xcc, 0xb3, 0x5c, 0xfb, 0x87, 0xf8, 0x26,
		0x3f, 0x57, 0x86, 0xe2, 0xd8, 0x0e, 0xd3, 0x26,
		0xcb, 0xc7, 0xf0, 0xe7, 0x1a, 0x99, 0xf4, 0x3b,
		0xfb, 0x98, 0x8b, 0x9b, 0x7a, 0x02, 0xdd, 0x21,
	}

	wrapped, err := aesKWWrap(kek, pt)
	if err != nil {
		t.Fatalf("wrap error: %v", err)
	}
	if !bytes.Equal(wrapped, wantCT) {
		t.Fatalf("wrap result mismatch\ngot:  %x\nwant: %x", wrapped, wantCT)
	}

	unwrapped, err := aesKWUnwrap(kek, wrapped)
	if err != nil {
		t.Fatalf("unwrap error: %v", err)
	}
	if !bytes.Equal(unwrapped, pt) {
		t.Fatalf("unwrap result mismatch\ngot:  %x\nwant: %x", unwrapped, pt)
	}
}

// Test 2: wrong password → ErrWrongPassword
func TestWrongPassword(t *testing.T) {
	v, err := Open("testdata/cmvault", "wrong")
	if err != ErrWrongPassword {
		t.Fatalf("expected ErrWrongPassword, got %v (v=%v)", err, v)
	}
}

// Test 3: DirIdHash("") → root directory path exists in the fixture
func TestDirIdHashRootExists(t *testing.T) {
	v, err := Open("testdata/cmvault", "password123")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	hash, err := v.DirIdHash("")
	if err != nil {
		t.Fatalf("DirIdHash: %v", err)
	}
	path := filepath.Join("testdata/cmvault", hash)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("DirIdHash(\"\") = %q, but path %q does not exist: %v", hash, path, err)
	}
}

// Test 4: interop — open a real Cryptomator vault and find the expected content
func TestInteropDecryptRealVault(t *testing.T) {
	v, err := Open("testdata/cmvault", "password123")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	outDir := t.TempDir()
	if err := v.DecryptTree("testdata/cmvault", outDir); err != nil {
		t.Fatalf("DecryptTree: %v", err)
	}

	want := []byte("Привет, мой верный друг!")
	found := false
	err = filepath.WalkDir(outDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if bytes.Equal(bytes.TrimSpace(data), want) || bytes.Contains(data, want) {
			found = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}
	if !found {
		// print decrypted file list for diagnostics
		t.Log("Decrypted files:")
		filepath.WalkDir(outDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			data, _ := os.ReadFile(path)
			t.Logf("  %s: %q", path[len(outDir):], data[:min(len(data), 80)])
			return nil
		})
		t.Fatalf("did not find a file with content %q", want)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
