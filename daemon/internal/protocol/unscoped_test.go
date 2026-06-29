package protocol

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUnscopedOmitsScopeHeader(t *testing.T) {
	var scoped, unscoped string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/device/token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"token": "jwt"})
	})
	mux.HandleFunc("GET /sync/changes", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Discodrive-Scope") == "1" {
			scoped = "yes"
		} else {
			unscoped = "yes"
		}
		json.NewEncoder(w).Encode(map[string]any{"changes": []any{}, "cursor": 0, "has_more": false})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, _, _, _ = New(srv.URL, "kfd").Changes(context.Background(), 0, 100)
	_, _, _, _ = NewUnscoped(srv.URL, "kfd").Changes(context.Background(), 0, 100)
	// New() sends the header (sets scoped); NewUnscoped() omits it (server takes the else → sets unscoped).
	if scoped != "yes" {
		t.Fatal("scoped client must send the header")
	}
	if unscoped != "yes" {
		t.Fatal("unscoped client must omit the header (no request reached the no-header branch)")
	}
}
