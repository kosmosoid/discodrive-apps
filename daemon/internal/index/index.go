// Package index is the local state index for the sync client (SQLite via modernc).
package index

import (
	"database/sql"
	"errors"
	"strconv"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS nodes (
    node_id      TEXT PRIMARY KEY,
    rel_path     TEXT NOT NULL,
    is_dir       INTEGER NOT NULL,
    version      INTEGER NOT NULL,
    content_hash TEXT,
    size         INTEGER
);
CREATE TABLE IF NOT EXISTS local (
    node_id TEXT PRIMARY KEY,
    state   TEXT NOT NULL,
    version INTEGER NOT NULL,
    path    TEXT NOT NULL
);`

// Node is an index record for a node known to the client.
type Node struct {
	NodeID      string
	RelPath     string
	IsDir       bool
	Version     int64
	ContentHash string
	Size        int64
}

type Index struct{ db *sql.DB }

// Open opens (or creates) the index and applies the schema.
func Open(path string) (*Index, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// SQLite allows only one writer. Serialize all access through a single
	// connection so concurrent callers (e.g. the desktop UI firing pin/unpin/refresh
	// from separate goroutines) queue instead of failing with "database is locked".
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Index{db: db}, nil
}

func (i *Index) Close() error { return i.db.Close() }

// Cursor returns the last applied seq (0 if never synced).
func (i *Index) Cursor() (int64, error) {
	var v string
	err := i.db.QueryRow("SELECT value FROM meta WHERE key = 'cursor'").Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(v, 10, 64)
}

func (i *Index) SetCursor(seq int64) error {
	_, err := i.db.Exec(
		"INSERT INTO meta(key, value) VALUES('cursor', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		strconv.FormatInt(seq, 10))
	return err
}

// ScopeEpoch returns the last scope epoch the client reconciled to (0 if never set).
func (i *Index) ScopeEpoch() (int64, error) {
	var v string
	err := i.db.QueryRow("SELECT value FROM meta WHERE key = 'scope_epoch'").Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(v, 10, 64)
}

func (i *Index) SetScopeEpoch(epoch int64) error {
	_, err := i.db.Exec(
		"INSERT INTO meta(key, value) VALUES('scope_epoch', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		strconv.FormatInt(epoch, 10))
	return err
}

// Clear drops all known nodes and resets the cursor to 0, so the next pull rebuilds the
// tree from scratch. Used when the sync scope changes. The scope_epoch is left untouched
// (the caller sets it after a successful reconcile).
func (i *Index) Clear() error {
	tx, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // rolled back only if Commit didn't run
	if _, err := tx.Exec("DELETE FROM nodes"); err != nil {
		return err
	}
	if _, err := tx.Exec(
		"INSERT INTO meta(key, value) VALUES('cursor', '0') ON CONFLICT(key) DO UPDATE SET value = excluded.value"); err != nil {
		return err
	}
	return tx.Commit()
}

func (i *Index) Get(nodeID string) (Node, bool, error) {
	var n Node
	var isDir int
	var hash sql.NullString
	err := i.db.QueryRow(
		"SELECT node_id, rel_path, is_dir, version, content_hash, size FROM nodes WHERE node_id = ?", nodeID).
		Scan(&n.NodeID, &n.RelPath, &isDir, &n.Version, &hash, &n.Size)
	if errors.Is(err, sql.ErrNoRows) {
		return Node{}, false, nil
	}
	if err != nil {
		return Node{}, false, err
	}
	n.IsDir = isDir != 0
	n.ContentHash = hash.String
	return n, true, nil
}

// GetByPath returns the node at the given rel_path (slash-relative), if present.
func (i *Index) GetByPath(relPath string) (Node, bool, error) {
	var n Node
	var isDir int
	var hash sql.NullString
	err := i.db.QueryRow(
		"SELECT node_id, rel_path, is_dir, version, content_hash, size FROM nodes WHERE rel_path = ?", relPath).
		Scan(&n.NodeID, &n.RelPath, &isDir, &n.Version, &hash, &n.Size)
	if errors.Is(err, sql.ErrNoRows) {
		return Node{}, false, nil
	}
	if err != nil {
		return Node{}, false, err
	}
	n.IsDir = isDir != 0
	n.ContentHash = hash.String
	return n, true, nil
}

func (i *Index) Put(n Node) error {
	_, err := i.db.Exec(
		`INSERT INTO nodes(node_id, rel_path, is_dir, version, content_hash, size) VALUES(?,?,?,?,?,?)
		 ON CONFLICT(node_id) DO UPDATE SET
		   rel_path = excluded.rel_path, is_dir = excluded.is_dir, version = excluded.version,
		   content_hash = excluded.content_hash, size = excluded.size`,
		n.NodeID, n.RelPath, boolToInt(n.IsDir), n.Version, n.ContentHash, n.Size)
	return err
}

func (i *Index) Delete(nodeID string) error {
	_, err := i.db.Exec("DELETE FROM nodes WHERE node_id = ?", nodeID)
	return err
}

// All returns all known nodes (for diffing against disk state).
func (i *Index) All() ([]Node, error) {
	rows, err := i.db.Query("SELECT node_id, rel_path, is_dir, version, content_hash, size FROM nodes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Node
	for rows.Next() {
		var n Node
		var isDir int
		var hash sql.NullString
		if err := rows.Scan(&n.NodeID, &n.RelPath, &isDir, &n.Version, &hash, &n.Size); err != nil {
			return nil, err
		}
		n.IsDir = isDir != 0
		n.ContentHash = hash.String
		out = append(out, n)
	}
	return out, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// --- browse + local-copy support (mobile Browser facade) ---

// escapeLike escapes LIKE wildcards so paths with % or _ match literally.
func escapeLike(s string) string {
	var b []rune
	for _, c := range s {
		if c == '\\' || c == '%' || c == '_' {
			b = append(b, '\\')
		}
		b = append(b, c)
	}
	return string(b)
}

// Children returns the direct children of the folder at parentRelPath ("" = root). Direct means
// rel_path is exactly one path segment deeper than parentRelPath.
func (i *Index) Children(parentRelPath string) ([]Node, error) {
	var rows *sql.Rows
	var err error
	if parentRelPath == "" {
		rows, err = i.db.Query(`SELECT node_id, rel_path, is_dir, version, content_hash, size
			FROM nodes WHERE rel_path NOT LIKE '%/%' ORDER BY is_dir DESC, rel_path`)
	} else {
		p := escapeLike(parentRelPath)
		rows, err = i.db.Query(`SELECT node_id, rel_path, is_dir, version, content_hash, size
			FROM nodes WHERE rel_path LIKE ? ESCAPE '\' AND rel_path NOT LIKE ? ESCAPE '\'
			ORDER BY is_dir DESC, rel_path`, p+"/%", p+"/%/%")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Node
	for rows.Next() {
		var n Node
		var isDir int
		var hash sql.NullString
		if err := rows.Scan(&n.NodeID, &n.RelPath, &isDir, &n.Version, &hash, &n.Size); err != nil {
			return nil, err
		}
		n.IsDir = isDir != 0
		n.ContentHash = hash.String
		out = append(out, n)
	}
	return out, rows.Err()
}

// SetLocal records a downloaded copy (state "cached" or "pinned").
func (i *Index) SetLocal(nodeID, state string, version int64, path string) error {
	_, err := i.db.Exec(`INSERT INTO local(node_id,state,version,path) VALUES(?,?,?,?)
		ON CONFLICT(node_id) DO UPDATE SET state=excluded.state, version=excluded.version, path=excluded.path`,
		nodeID, state, version, path)
	return err
}

// LocalStatus returns the cache state ("" if none), whether it is stale vs serverVersion, and path.
func (i *Index) LocalStatus(nodeID string, serverVersion int64) (state string, stale bool, path string) {
	var v int64
	if i.db.QueryRow("SELECT state, version, path FROM local WHERE node_id=?", nodeID).Scan(&state, &v, &path) != nil {
		return "", false, ""
	}
	return state, v < serverVersion, path
}

// LocalPathOf returns the cached file path, or "".
func (i *Index) LocalPathOf(nodeID string) string {
	var p string
	if i.db.QueryRow("SELECT path FROM local WHERE node_id=?", nodeID).Scan(&p) != nil {
		return ""
	}
	return p
}

func (i *Index) DeleteLocal(nodeID string) error {
	_, err := i.db.Exec("DELETE FROM local WHERE node_id=?", nodeID)
	return err
}

// RelocateLocal updates the on-disk path recorded for a node's local copy, used when a
// rename/move changes the node's rel_path and the cached file is moved to match.
func (i *Index) RelocateLocal(nodeID, newPath string) error {
	_, err := i.db.Exec("UPDATE local SET path=? WHERE node_id=?", newPath, nodeID)
	return err
}

// ListPinned returns the node IDs marked pinned.
func (i *Index) ListPinned() []string {
	rows, err := i.db.Query("SELECT node_id FROM local WHERE state='pinned'")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			out = append(out, id)
		}
	}
	return out
}
