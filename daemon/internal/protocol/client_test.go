package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestClientAuthChangesDownload(t *testing.T) {
	var changesHits int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/device/token", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			DeviceToken string `json:"device_token"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.DeviceToken != "kfd_test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "jwt-123"})
	})
	mux.HandleFunc("GET /sync/changes", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer jwt-123" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		n := atomic.AddInt32(&changesHits, 1)
		if r.URL.Query().Get("since") == "0" && n == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"changes": []map[string]any{{"seq": 1, "op": "create", "node_id": "n1", "path": "a.txt", "content_hash": "h", "size": 1, "is_dir": false, "deleted": false}},
				"cursor":  1, "has_more": true,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"changes": []map[string]any{{"seq": 2, "op": "create", "node_id": "n2", "path": "b.txt", "content_hash": "h2", "size": 1, "is_dir": false, "deleted": false}},
			"cursor":  2, "has_more": false,
		})
	})
	mux.HandleFunc("GET /files/{id}/content", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("body-" + r.PathValue("id")))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(srv.URL, "kfd_test")
	ctx := context.Background()

	ch1, cur1, more1, err := c.Changes(ctx, 0, 500)
	if err != nil || len(ch1) != 1 || ch1[0].NodeID != "n1" || cur1 != 1 || !more1 {
		t.Fatalf("page1: %+v cur=%d more=%v err=%v", ch1, cur1, more1, err)
	}
	ch2, _, more2, err := c.Changes(ctx, 1, 500)
	if err != nil || len(ch2) != 1 || ch2[0].NodeID != "n2" || more2 {
		t.Fatalf("page2: %+v more=%v err=%v", ch2, more2, err)
	}

	var buf bytes.Buffer
	if err := c.Download(ctx, "n1", &buf); err != nil || buf.String() != "body-n1" {
		t.Fatalf("download: %q err=%v", buf.String(), err)
	}
}

func TestClientRefreshesOn401(t *testing.T) {
	var tokenIssued int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/device/token", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&tokenIssued, 1)
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "fresh"})
	})
	var first int32 = 1
	mux.HandleFunc("GET /sync/changes", func(w http.ResponseWriter, r *http.Request) {
		if atomic.CompareAndSwapInt32(&first, 1, 0) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"changes": []any{}, "cursor": 0, "has_more": false})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(srv.URL, "kfd_test")
	if _, _, _, err := c.Changes(context.Background(), 0, 500); err != nil {
		t.Fatalf("expected success after refresh: %v", err)
	}
	if tokenIssued < 2 {
		t.Fatalf("token should have been reissued on 401, issued %d times", tokenIssued)
	}
}
