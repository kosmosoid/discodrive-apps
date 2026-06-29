package vault

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVersionMacFormula: verify the versionMac formula against a known-good value.
// Opens a real vault, takes its macKey, and checks:
// HMAC-SHA256(macKey, BE32(999)) == base64decode("hU08fq2tiYrDqTpjUTE4Nx1wy0lPh7rarqazsbuuG3g=")
func TestVersionMacFormula(t *testing.T) {
	v, err := Open("testdata/cmvault", "password123")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	wantB64 := "hU08fq2tiYrDqTpjUTE4Nx1wy0lPh7rarqazsbuuG3g="
	want, err := base64.StdEncoding.DecodeString(wantB64)
	if err != nil {
		t.Fatalf("decode want: %v", err)
	}

	got := computeVersionMac(v.macKey, 999)
	if !bytes.Equal(got, want) {
		t.Fatalf("versionMac mismatch\ngot:  %x\nwant: %x\nTried: HMAC-SHA256(macKey, BE32(999))", got, want)
	}
	t.Logf("versionMac formula verified: HMAC-SHA256(macKey, BE32(version=999))")
}

// TestCreateOpenRoundTrip: Create → Open returns the same Vault (encKey/macKey match).
func TestCreateOpenRoundTrip(t *testing.T) {
	dir := t.TempDir()
	v1, err := Create(dir, "testpassword")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	v2, err := Open(dir, "testpassword")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !bytes.Equal(v1.encKey, v2.encKey) {
		t.Fatalf("encKey mismatch\nv1: %x\nv2: %x", v1.encKey, v2.encKey)
	}
	if !bytes.Equal(v1.macKey, v2.macKey) {
		t.Fatalf("macKey mismatch\nv1: %x\nv2: %x", v1.macKey, v2.macKey)
	}
}

// TestCreateWrongPassword: Create → Open(wrong) = ErrWrongPassword.
func TestCreateWrongPassword(t *testing.T) {
	dir := t.TempDir()
	if _, err := Create(dir, "correct"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	_, err := Open(dir, "wrong")
	if err != ErrWrongPassword {
		t.Fatalf("expected ErrWrongPassword, got: %v", err)
	}
}

// TestEncryptNameRoundTrip: EncryptName → DecryptName == original.
func TestEncryptNameRoundTrip(t *testing.T) {
	dir := t.TempDir()
	v, err := Create(dir, "pw")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	cases := []struct {
		name     string
		parentID string
	}{
		{"hello.txt", ""},
		{"document.pdf", "some-dir-id-123"},
		{"файл с пробелами.md", ""},
		{"verylongfilename_with_extra_chars_abc.txt", "parent-uuid-here"},
	}

	for _, tc := range cases {
		enc, err := v.EncryptName(tc.name, tc.parentID)
		if err != nil {
			t.Fatalf("EncryptName(%q, %q): %v", tc.name, tc.parentID, err)
		}
		if !strings.HasSuffix(enc, ".c9r") {
			t.Fatalf("EncryptName(%q): result %q does not end with .c9r", tc.name, enc)
		}
		dec, err := v.DecryptName(enc, tc.parentID)
		if err != nil {
			t.Fatalf("DecryptName(%q, %q): %v", enc, tc.parentID, err)
		}
		if dec != tc.name {
			t.Fatalf("round-trip failed: got %q, want %q", dec, tc.name)
		}
	}
}

// TestEncryptNameIsDeterministic: same inputs → same output name.
func TestEncryptNameIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	v, err := Create(dir, "pw")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	enc1, _ := v.EncryptName("test.txt", "parent")
	enc2, _ := v.EncryptName("test.txt", "parent")
	if enc1 != enc2 {
		t.Fatalf("EncryptName is not deterministic: %q vs %q", enc1, enc2)
	}
}

// TestEncryptNameParentBinding: different parentDirIDs produce different encrypted names.
func TestEncryptNameParentBinding(t *testing.T) {
	dir := t.TempDir()
	v, err := Create(dir, "pw")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	enc1, _ := v.EncryptName("same.txt", "parent-a")
	enc2, _ := v.EncryptName("same.txt", "parent-b")
	if enc1 == enc2 {
		t.Fatalf("EncryptName must produce different names for different parentDirID")
	}
}

// TestEncryptContentRoundTrip: EncryptContent → DecryptContent == original.
func TestEncryptContentRoundTrip(t *testing.T) {
	dir := t.TempDir()
	v, err := Create(dir, "pw")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	cases := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"small", []byte("Hello, Cryptomator!")},
		{"exact32k", bytes.Repeat([]byte{0xAB}, 32*1024)},
		{"over32k", bytes.Repeat([]byte{0xCD}, 32*1024+100)},
		{"multi-chunk", bytes.Repeat([]byte{0xEF}, 3*32*1024+777)},
	}

	for _, tc := range cases {
		var encrypted bytes.Buffer
		if err := v.EncryptContent(&encrypted, bytes.NewReader(tc.data)); err != nil {
			t.Fatalf("[%s] EncryptContent: %v", tc.name, err)
		}

		var decrypted bytes.Buffer
		if err := v.DecryptContent(&decrypted, &encrypted); err != nil {
			t.Fatalf("[%s] DecryptContent: %v", tc.name, err)
		}

		if !bytes.Equal(decrypted.Bytes(), tc.data) {
			t.Fatalf("[%s] round-trip failed: got %d bytes, want %d bytes", tc.name, decrypted.Len(), len(tc.data))
		}
	}
}

// TestEncryptContentHeaderSize: verify that an encrypted empty file is exactly the header size (68B).
func TestEncryptContentHeaderSize(t *testing.T) {
	dir := t.TempDir()
	v, err := Create(dir, "pw")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var buf bytes.Buffer
	if err := v.EncryptContent(&buf, bytes.NewReader(nil)); err != nil {
		t.Fatalf("EncryptContent(empty): %v", err)
	}
	// Header: headerNonce(12) + AES-GCM(encKey, headerNonce, 0xFF*8||contentKey(32), nil) + tag(16)
	// payload = 8 + 32 = 40B, GCM overhead = 16B → ct = 56B. Total: 12 + 56 = 68B.
	if buf.Len() != 68 {
		t.Fatalf("empty file: expected a 68B header, got %d", buf.Len())
	}
}

// TestEncryptContentChunkAAD: encrypted chunks must have the correct AAD (chunk index + headerNonce).
// Verified indirectly: swapping chunk order must cause DecryptContent to fail.
func TestEncryptContentChunkAAD(t *testing.T) {
	dir := t.TempDir()
	v, err := Create(dir, "pw")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Data for two full chunks
	data := bytes.Repeat([]byte{0x01}, 2*32*1024)
	var encrypted bytes.Buffer
	if err := v.EncryptContent(&encrypted, bytes.NewReader(data)); err != nil {
		t.Fatalf("EncryptContent: %v", err)
	}

	encBytes := encrypted.Bytes()
	// Format: header(68) + chunk0(12+32768+16) + chunk1(12+32768+16)
	headerSize := 68
	chunkFrameSize := 12 + 32*1024 + 16

	if len(encBytes) != headerSize+2*chunkFrameSize {
		t.Fatalf("unexpected size: %d (expected %d)", len(encBytes), headerSize+2*chunkFrameSize)
	}

	// Swap chunk 0 and chunk 1
	tampered := make([]byte, len(encBytes))
	copy(tampered[:headerSize], encBytes[:headerSize])
	copy(tampered[headerSize:headerSize+chunkFrameSize], encBytes[headerSize+chunkFrameSize:])
	copy(tampered[headerSize+chunkFrameSize:], encBytes[headerSize:headerSize+chunkFrameSize])

	var out bytes.Buffer
	if err := v.DecryptContent(&out, bytes.NewReader(tampered)); err == nil {
		t.Fatal("DecryptContent must fail when chunks are swapped, but it did not")
	}
}

// TestEncryptTreeRoundTrip: main round-trip test — create tree → encrypt → decrypt → byte-for-byte.
func TestEncryptTreeRoundTrip(t *testing.T) {
	// Create the test tree
	srcDir := t.TempDir()
	mustWriteFile(t, filepath.Join(srcDir, "hello.txt"), []byte("Hello, World!"))
	mustWriteFile(t, filepath.Join(srcDir, "data.bin"), bytes.Repeat([]byte{0xDE, 0xAD, 0xBE, 0xEF}, 100))

	// Subdirectory with files
	subDir := filepath.Join(srcDir, "subdir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("MkdirAll subdir: %v", err)
	}
	mustWriteFile(t, filepath.Join(subDir, "nested.txt"), []byte("nested file content"))
	mustWriteFile(t, filepath.Join(subDir, "Привет.md"), []byte("Привет, мой верный друг!"))

	// Multi-chunk file (>32KiB)
	bigData := bytes.Repeat([]byte{0xAB}, 65*1024+321)
	mustWriteFile(t, filepath.Join(srcDir, "bigfile.bin"), bigData)

	// Create vault and encrypt
	vaultDir := t.TempDir()
	v, err := Create(vaultDir, "testpw")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := v.EncryptTree(srcDir, vaultDir); err != nil {
		t.Fatalf("EncryptTree: %v", err)
	}

	// Verify no plaintext filenames exist in vaultDir
	for _, name := range []string{"hello.txt", "data.bin", "bigfile.bin", "nested.txt", "Привет.md", "subdir"} {
		if fileExists(filepath.Join(vaultDir, name)) {
			t.Fatalf("vault contains an unencrypted name %q — only the d/ structure is allowed", name)
		}
	}

	// Decrypt into a new directory
	outDir := t.TempDir()
	if err := v.DecryptTree(vaultDir, outDir); err != nil {
		t.Fatalf("DecryptTree: %v", err)
	}

	// Verify byte-for-byte identity
	compareDirTrees(t, srcDir, outDir)
}

// TestEncryptTreeEmptyFile: an empty file encrypts and decrypts correctly.
func TestEncryptTreeEmptyFile(t *testing.T) {
	srcDir := t.TempDir()
	mustWriteFile(t, filepath.Join(srcDir, "empty.txt"), []byte{})

	vaultDir := t.TempDir()
	v, err := Create(vaultDir, "pw")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := v.EncryptTree(srcDir, vaultDir); err != nil {
		t.Fatalf("EncryptTree: %v", err)
	}

	outDir := t.TempDir()
	if err := v.DecryptTree(vaultDir, outDir); err != nil {
		t.Fatalf("DecryptTree: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(outDir, "empty.txt"))
	if err != nil {
		t.Fatalf("ReadFile empty.txt: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("empty file after round-trip: %d bytes", len(got))
	}
}

// TestExistingReaderIntact: ensure existing reader tests are not broken.
// Integration test: opens a real Cryptomator vault.
func TestExistingReaderIntact(t *testing.T) {
	v, err := Open("testdata/cmvault", "password123")
	if err != nil {
		t.Fatalf("Open of the real vault: %v", err)
	}
	outDir := t.TempDir()
	if err := v.DecryptTree("testdata/cmvault", outDir); err != nil {
		t.Fatalf("DecryptTree of the real vault: %v", err)
	}
	want := []byte("Привет, мой верный друг!")
	found := false
	filepath.WalkDir(outDir, func(path string, d os.DirEntry, _ error) error {
		if d.IsDir() {
			return nil
		}
		data, _ := os.ReadFile(path)
		if bytes.Contains(data, want) {
			found = true
		}
		return nil
	})
	if !found {
		t.Fatal("did not find «Привет, мой верный друг!» in the decrypted real vault")
	}
}

// --- helpers ---

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile %q: %v", path, err)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func compareDirTrees(t *testing.T, want, got string) {
	t.Helper()
	// Collect all files from want
	err := filepath.WalkDir(want, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(want, path)
		wantData, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		gotPath := filepath.Join(got, rel)
		gotData, err := os.ReadFile(gotPath)
		if err != nil {
			t.Errorf("file %q is missing from the decrypted tree: %v", rel, err)
			return nil
		}
		if !bytes.Equal(wantData, gotData) {
			t.Errorf("file %q: content did not match (want %d bytes, got %d bytes)", rel, len(wantData), len(gotData))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir want: %v", err)
	}
	// Verify there are no extra files in got
	err = filepath.WalkDir(got, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(got, path)
		wantPath := filepath.Join(want, rel)
		if _, err := os.Stat(wantPath); os.IsNotExist(err) {
			t.Errorf("extra file in the decrypted tree: %q", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir got: %v", err)
	}
}

// computeVersionMac is a test helper (accessed directly because tests are in the same package).
func computeVersionMac(macKey []byte, version uint32) []byte {
	h := hmac.New(sha256.New, macKey)
	var vb [4]byte
	binary.BigEndian.PutUint32(vb[:], version)
	h.Write(vb[:])
	return h.Sum(nil)
}

// TestVersionMacRaw: sanity-check that computeVersionMac works — smoke test, not interop.
func TestVersionMacRaw(t *testing.T) {
	// Zero key, zero version — just a smoke test
	result := computeVersionMac(make([]byte, 32), 0)
	if len(result) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(result))
	}
}

// Verify that the io.Reader interface is accessible in tests
var _ io.Reader = (*bytes.Reader)(nil)

// longPlainName generates a plaintext filename long enough that the encrypted name exceeds 220 chars.
// AES-SIV adds a 16-byte tag; base64url is ~4/3 × len(ct). To reliably exceed 220:
// len(base64(ct)) + len(".c9r") > 220, i.e. len(base64(ct)) > 216.
// base64url len ≈ ceil(len(ct)*4/3). len(ct) = 16 + len(name).
// With len(name)=150: len(ct)=166, base64len=ceil(166*4/3)=ceil(221.3)=222 → total 222+4=226 > 220 ✓
func longPlainName(suffix string) string {
	return strings.Repeat("a", 150) + suffix
}

// TestC9SFileRoundTrip: file with long name → .c9s structure in vault → DecryptTree → byte-for-byte.
func TestC9SFileRoundTrip(t *testing.T) {
	srcDir := t.TempDir()
	longName := longPlainName(".txt")
	content := []byte("содержимое длинноимённого файла")
	mustWriteFile(t, filepath.Join(srcDir, longName), content)

	vaultDir := t.TempDir()
	v, err := Create(vaultDir, "pw")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := v.EncryptTree(srcDir, vaultDir); err != nil {
		t.Fatalf("EncryptTree: %v", err)
	}

	// Verify a .c9s directory exists in the vault
	encDir := findEncryptedRootDir(t, vaultDir)
	entries, err := os.ReadDir(encDir)
	if err != nil {
		t.Fatalf("ReadDir encDir: %v", err)
	}
	var foundC9S bool
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".c9s") && e.IsDir() {
			foundC9S = true
			// Verify name.c9s and contents.c9r are present
			c9sPath := filepath.Join(encDir, e.Name())
			if _, err := os.Stat(filepath.Join(c9sPath, "name.c9s")); err != nil {
				t.Errorf(".c9s folder without name.c9s: %v", err)
			}
			if _, err := os.Stat(filepath.Join(c9sPath, "contents.c9r")); err != nil {
				t.Errorf(".c9s folder without contents.c9r: %v", err)
			}
		}
	}
	if !foundC9S {
		t.Fatal("no .c9s folder found in the vault for the long file name")
	}

	// Decrypt and compare
	outDir := t.TempDir()
	if err := v.DecryptTree(vaultDir, outDir); err != nil {
		t.Fatalf("DecryptTree: %v", err)
	}

	gotData, err := os.ReadFile(filepath.Join(outDir, longName))
	if err != nil {
		t.Fatalf("ReadFile after DecryptTree: %v", err)
	}
	if !bytes.Equal(gotData, content) {
		t.Fatalf("content did not match: got %q, want %q", gotData, content)
	}
}

// TestC9SDirRoundTrip: directory with long name + file inside → .c9s dir.c9r → recursion → ok.
func TestC9SDirRoundTrip(t *testing.T) {
	srcDir := t.TempDir()
	longDirName := longPlainName("_dir")
	subDir := filepath.Join(srcDir, longDirName)
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	innerContent := []byte("файл внутри длинноимённой папки")
	mustWriteFile(t, filepath.Join(subDir, "inner.txt"), innerContent)

	vaultDir := t.TempDir()
	v, err := Create(vaultDir, "pw")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := v.EncryptTree(srcDir, vaultDir); err != nil {
		t.Fatalf("EncryptTree: %v", err)
	}

	// Check for .c9s with dir.c9r anywhere in the d/ tree (walk the whole tree
	// rather than guessing which storage folder is the root).
	var foundC9SDir bool
	_ = filepath.WalkDir(filepath.Join(vaultDir, "d"), func(p string, d os.DirEntry, err error) error {
		if err == nil && d.IsDir() && strings.HasSuffix(d.Name(), ".c9s") {
			if _, serr := os.Stat(filepath.Join(p, "dir.c9r")); serr == nil {
				foundC9SDir = true
			}
		}
		return nil
	})
	if !foundC9SDir {
		t.Fatal("did not find a .c9s with dir.c9r for the long-named folder")
	}

	// DecryptTree → verify content
	outDir := t.TempDir()
	if err := v.DecryptTree(vaultDir, outDir); err != nil {
		t.Fatalf("DecryptTree: %v", err)
	}
	gotData, err := os.ReadFile(filepath.Join(outDir, longDirName, "inner.txt"))
	if err != nil {
		t.Fatalf("ReadFile inner.txt: %v", err)
	}
	if !bytes.Equal(gotData, innerContent) {
		t.Fatalf("inner.txt content did not match")
	}
}

// TestC9SMixedTree: short and long names in one tree — round-trip ok.
func TestC9SMixedTree(t *testing.T) {
	srcDir := t.TempDir()
	// Short names
	mustWriteFile(t, filepath.Join(srcDir, "short.txt"), []byte("short"))
	// Long file name
	longFile := longPlainName(".bin")
	mustWriteFile(t, filepath.Join(srcDir, longFile), []byte("long file content"))
	// Short-named directory containing a long-named file
	subDir := filepath.Join(srcDir, "subdir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	longNested := longPlainName("_nested.txt")
	mustWriteFile(t, filepath.Join(subDir, longNested), []byte("nested long file"))
	// Long-named directory
	longDir := longPlainName("_folder")
	longSubDir := filepath.Join(srcDir, longDir)
	if err := os.MkdirAll(longSubDir, 0o755); err != nil {
		t.Fatalf("MkdirAll longDir: %v", err)
	}
	mustWriteFile(t, filepath.Join(longSubDir, "deep.txt"), []byte("deep content"))

	vaultDir := t.TempDir()
	v, err := Create(vaultDir, "pw")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := v.EncryptTree(srcDir, vaultDir); err != nil {
		t.Fatalf("EncryptTree: %v", err)
	}

	outDir := t.TempDir()
	if err := v.DecryptTree(vaultDir, outDir); err != nil {
		t.Fatalf("DecryptTree: %v", err)
	}

	compareDirTrees(t, srcDir, outDir)
}

// findEncryptedRootDir finds the root storage directory (d/XX/YYY) in vaultDir.
func findEncryptedRootDir(t *testing.T, vaultDir string) string {
	t.Helper()
	dDir := filepath.Join(vaultDir, "d")
	entries, err := os.ReadDir(dDir)
	if err != nil {
		t.Fatalf("ReadDir d/: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("d/ is empty")
	}
	// Walk d/XX/YYY to find the root directory (dirID="").
	// All dirs contain dirid.c9r; the root is the one with additional files.
	for _, e := range entries {
		subPath := filepath.Join(dDir, e.Name())
		subs, _ := os.ReadDir(subPath)
		for _, s := range subs {
			candidate := filepath.Join(subPath, s.Name())
			cs, _ := os.ReadDir(candidate)
			// Root directory contains dirid.c9r and encrypted files/folders
			if len(cs) > 1 {
				return candidate
			}
		}
	}
	t.Fatal("did not find the root directory in d/")
	return ""
}
