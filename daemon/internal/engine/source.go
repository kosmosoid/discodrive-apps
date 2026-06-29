// Package engine is the sync client core: fetches changes and applies them to disk.
package engine

import (
	"context"
	"io"
)

// Change is a single entry from the server change feed (mirrors the server's /sync/changes).
type Change struct {
	Seq         int64
	Op          string
	NodeID      string
	RelPath     string // path relative to the sync root (slash-separated)
	IsDir       bool
	Version     int64
	ContentHash string
	Size        int64
	Deleted     bool
}

// Source provides changes and file content (implemented by the HTTP protocol client).
type Source interface {
	Changes(ctx context.Context, since int64, limit int) (changes []Change, cursor int64, hasMore bool, err error)
	Download(ctx context.Context, nodeID string, w io.Writer) error
}
