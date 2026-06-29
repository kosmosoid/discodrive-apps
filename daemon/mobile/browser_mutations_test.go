package mobile

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserMutations(t *testing.T) {
	hits := map[string]string{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/device/token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"token": "jwt"})
	})
	mux.HandleFunc("POST /files/folder", func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		hits["folder"] = b["name"].(string)
		json.NewEncoder(w).Encode(map[string]any{"id": "nd", "name": b["name"], "is_dir": true, "version": 1})
	})
	mux.HandleFunc("POST /files/upload", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(1 << 20)
		hits["upload"] = r.FormValue("name")
		json.NewEncoder(w).Encode(map[string]any{"id": "nu", "name": r.FormValue("name"), "is_dir": false, "version": 1})
	})
	mux.HandleFunc("PATCH /files/{id}/rename", func(w http.ResponseWriter, r *http.Request) {
		hits["rename"] = r.PathValue("id")
		json.NewEncoder(w).Encode(map[string]any{"id": r.PathValue("id"), "name": "x", "is_dir": false, "version": 2})
	})
	mux.HandleFunc("PATCH /files/{id}/move", func(w http.ResponseWriter, r *http.Request) {
		hits["move"] = r.PathValue("id")
		json.NewEncoder(w).Encode(map[string]any{"id": r.PathValue("id"), "name": "x", "is_dir": false, "version": 2})
	})
	mux.HandleFunc("DELETE /files/{id}", func(w http.ResponseWriter, r *http.Request) {
		hits["delete"] = r.PathValue("id")
		w.WriteHeader(http.StatusNoContent)
	})
	// auto-Refresh after each mutation hits /sync/changes; return empty deltas.
	mux.HandleFunc("GET /sync/changes", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"changes": []any{}, "cursor": 0, "has_more": false})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	b, _ := NewBrowser(srv.URL, "kfd", t.TempDir(), filepath.Join(t.TempDir(), "i.db"), false)
	defer b.Close()

	if err := b.Mkdir("", "newdir"); err != nil || hits["folder"] != "newdir" {
		t.Fatalf("mkdir: %v %v", err, hits)
	}
	tmp := filepath.Join(t.TempDir(), "u.txt")
	os.WriteFile(tmp, []byte("x"), 0o644)
	if err := b.Upload(tmp, "p1"); err != nil || hits["upload"] != "u.txt" {
		t.Fatalf("upload: %v %v", err, hits)
	}
	if err := b.Rename("n9", "b.txt"); err != nil || hits["rename"] != "n9" {
		t.Fatalf("rename: %v %v", err, hits)
	}
	if err := b.Move("n9", "p2"); err != nil || hits["move"] != "n9" {
		t.Fatalf("move: %v %v", err, hits)
	}
	if err := b.Delete("n9"); err != nil || hits["delete"] != "n9" {
		t.Fatalf("delete: %v %v", err, hits)
	}
}
