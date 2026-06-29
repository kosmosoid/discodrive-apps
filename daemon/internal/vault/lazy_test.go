package vault

import (
	"bytes"
	"testing"
)

func TestOpenWithSourceCmvault(t *testing.T) {
	src := OSIO{Root: "testdata/cmvault"}

	v, err := OpenWithSource(src, "password123")
	if err != nil {
		t.Fatalf("OpenWithSource: %v", err)
	}
	if v == nil {
		t.Fatal("OpenWithSource returned a nil vault")
	}

	if _, err := OpenWithSource(src, "wrong"); err != ErrWrongPassword {
		t.Fatalf("expected ErrWrongPassword, got %v", err)
	}
}

func TestLazyReadCmvault(t *testing.T) {
	src := OSIO{Root: "testdata/cmvault"}
	v, err := OpenWithSource(src, "password123")
	if err != nil {
		t.Fatalf("OpenWithSource: %v", err)
	}

	want := []byte("Привет, мой верный друг!")
	if !walkFind(t, v, src, "", want) {
		t.Fatalf("did not find a file with content %q via the lazy API", want)
	}
}

func TestLazyWriteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	v, err := Create(dir, "pw")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	sink := OSIO{Root: dir}

	// write a file to root
	data := []byte("hello lazy")
	if err := v.WriteFile(sink, "", "a.txt", data); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// ListDir root shows a.txt
	if !listNames(t, v, sink, "")["a.txt"] {
		t.Fatalf("a.txt is not visible at root")
	}

	// ReadFile a.txt == data
	if got := readByName(t, v, sink, "", "a.txt"); !bytes.Equal(got, data) {
		t.Fatalf("ReadFile a.txt = %q, expected %q", got, data)
	}

	// MakeDir sub
	subID, err := v.MakeDir(sink, "", "sub")
	if err != nil {
		t.Fatalf("MakeDir: %v", err)
	}
	if !listNames(t, v, sink, "")["sub"] {
		t.Fatalf("sub is not visible at root")
	}

	// write into sub, visible via ListDir(subID)
	subData := []byte("inside sub")
	if err := v.WriteFile(sink, subID, "b.txt", subData); err != nil {
		t.Fatalf("WriteFile sub: %v", err)
	}
	if got := readByName(t, v, sink, subID, "b.txt"); !bytes.Equal(got, subData) {
		t.Fatalf("ReadFile sub/b.txt = %q, expected %q", got, subData)
	}

	// Remove a.txt → gone
	if err := v.Remove(sink, "", "a.txt"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if listNames(t, v, sink, "")["a.txt"] {
		t.Fatalf("a.txt is still visible after Remove")
	}
}

func listNames(t *testing.T, v *Vault, src Source, dirID string) map[string]bool {
	entries, err := v.ListDir(src, dirID)
	if err != nil {
		t.Fatalf("ListDir(%q): %v", dirID, err)
	}
	m := map[string]bool{}
	for _, e := range entries {
		m[e.Name] = true
	}
	return m
}

func readByName(t *testing.T, v *Vault, src Source, dirID, name string) []byte {
	entries, err := v.ListDir(src, dirID)
	if err != nil {
		t.Fatalf("ListDir(%q): %v", dirID, err)
	}
	for _, e := range entries {
		if e.Name == name && !e.IsDir {
			data, err := v.ReadFile(src, e.FileStoragePath)
			if err != nil {
				t.Fatalf("ReadFile(%q): %v", name, err)
			}
			return data
		}
	}
	t.Fatalf("file %q not found in dirID=%q", name, dirID)
	return nil
}

// walkFind recursively lists dirID via the lazy API and returns true if any
// file decrypts to content containing want.
func walkFind(t *testing.T, v *Vault, src Source, dirID string, want []byte) bool {
	entries, err := v.ListDir(src, dirID)
	if err != nil {
		t.Fatalf("ListDir(%q): %v", dirID, err)
	}
	for _, e := range entries {
		if e.IsDir {
			if walkFind(t, v, src, e.DirID, want) {
				return true
			}
			continue
		}
		data, err := v.ReadFile(src, e.FileStoragePath)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", e.FileStoragePath, err)
		}
		if bytes.Contains(data, want) {
			return true
		}
	}
	return false
}
