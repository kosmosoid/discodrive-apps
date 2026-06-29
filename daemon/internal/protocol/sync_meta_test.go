package protocol

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// SyncMeta returns the server's scope epoch, and every request carries the daemon's
// X-Discodrive-Scope header (so the server applies scoping only to the daemon).
func TestSyncMetaAndScopeHeader(t *testing.T) {
	var sawScopeHeader bool
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/device/token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "jwt-123"})
	})
	mux.HandleFunc("GET /sync/meta", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer jwt-123" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		sawScopeHeader = r.Header.Get("X-Discodrive-Scope") == "1"
		_ = json.NewEncoder(w).Encode(map[string]any{"scope_epoch": 7})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(srv.URL, "kfd_test")
	epoch, err := c.SyncMeta(context.Background())
	if err != nil {
		t.Fatalf("SyncMeta: %v", err)
	}
	if epoch != 7 {
		t.Fatalf("expected epoch 7, got %d", epoch)
	}
	if !sawScopeHeader {
		t.Fatal("daemon must send X-Discodrive-Scope: 1")
	}
}
