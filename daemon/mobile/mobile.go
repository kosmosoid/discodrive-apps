// Package mobile is the gomobile-bound facade over the sync core. Every exported type and
// function signature stays within gomobile's bindable type set (no context.Context,
// time.Duration, maps or non-byte slices).
package mobile

import (
	"context"
	"os"
	"sync"
	"time"

	"discodrive.org/daemon/internal/engine"
	"discodrive.org/daemon/internal/index"
	"discodrive.org/daemon/internal/protocol"
)

// setInsecure opts the whole process into accepting self-signed TLS (dev / LAN-IP). A mobile
// process serves one server config, so a process-global flag (the same DISCODRIVE_INSECURE_TLS
// the desktop reads) is acceptable.
func setInsecure(insecure bool) {
	if insecure {
		_ = os.Setenv("DISCODRIVE_INSECURE_TLS", "1")
	}
}

// Pairing carries what the app needs to complete device pairing.
type Pairing struct {
	VerificationURL string // absolute URL the app opens in a browser
	UserCode        string // code the user confirms in the browser
	DeviceCode      string // opaque; pass to PairAwait
	IntervalSeconds int    // poll interval hint
}

// PairBegin starts device pairing. deviceKind is "ios" or "android". Call off the UI thread.
func PairBegin(serverURL, deviceName, deviceKind string, insecureTLS bool) (*Pairing, error) {
	setInsecure(insecureTLS)
	p, err := protocol.PairInit(context.Background(), serverURL, deviceName, deviceKind)
	if err != nil {
		return nil, err
	}
	return &Pairing{
		VerificationURL: serverURL + p.VerificationURI,
		UserCode:        p.UserCode,
		DeviceCode:      p.DeviceCode,
		IntervalSeconds: p.Interval,
	}, nil
}

// PairAwait blocks until the user approves, returning the device token. Call off the UI thread.
func PairAwait(serverURL, deviceCode string, intervalSeconds int, insecureTLS bool) (string, error) {
	setInsecure(insecureTLS)
	if intervalSeconds <= 0 {
		intervalSeconds = 2
	}
	return protocol.PairPoll(context.Background(), serverURL, deviceCode, time.Duration(intervalSeconds)*time.Second)
}

// Status is a snapshot the app renders. The state machine is driven by SyncOnce.
type Status struct {
	State        string // "idle" | "syncing" | "offline"
	LastSyncUnix int64  // 0 if never synced successfully
	LastError    string // last SyncOnce error text, "" if none
}

// Client is a sync handle for one paired device + one sync folder.
type Client struct {
	client *protocol.Client
	eng    *engine.Engine
	idx    *index.Index

	mu     sync.Mutex
	status Status
}

// New builds a sync client. syncDir is the app-sandbox folder to mirror; stateDBPath is a
// writable path for the local index (sqlite). Both come from the app's sandbox.
func New(serverURL, deviceToken, syncDir, stateDBPath string, insecureTLS bool) (*Client, error) {
	setInsecure(insecureTLS)
	if err := os.MkdirAll(syncDir, 0o755); err != nil {
		return nil, err
	}
	idx, err := index.Open(stateDBPath)
	if err != nil {
		return nil, err
	}
	client := protocol.New(serverURL, deviceToken)
	eng := engine.New(client, idx, syncDir)
	return &Client{client: client, eng: eng, idx: idx, status: Status{State: "idle"}}, nil
}

// SyncOnce runs one sync pass. Blocks; call off the UI thread. Concurrent calls serialize.
func (c *Client) SyncOnce() error {
	c.mu.Lock()
	c.status.State = "syncing"
	c.mu.Unlock()

	err := c.syncPass(context.Background())

	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		c.status.State = "offline"
		c.status.LastError = err.Error()
		return err
	}
	c.status.State = "idle"
	c.status.LastError = ""
	c.status.LastSyncUnix = time.Now().Unix()
	return nil
}

// syncPass mirrors internal/syncer.(*Syncer).SyncOnce. It is duplicated here (instead of
// importing syncer) so the mobile bind does not pull in syncer's fsnotify dependency. Keep the
// two in sync: on a scope-epoch change reconcile and skip push; otherwise push then pull.
func (c *Client) syncPass(ctx context.Context) error {
	epoch, err := c.client.SyncMeta(ctx)
	if err != nil {
		return err
	}
	last, err := c.eng.ScopeEpoch()
	if err != nil {
		return err
	}
	if epoch != last {
		return c.eng.ResetForScope(ctx, epoch)
	}
	if err := c.eng.PushLocal(ctx, c.client); err != nil {
		return err
	}
	return c.eng.PullOnce(ctx)
}

// Status returns a copy of the tracked state.
func (c *Client) Status() *Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.status
	return &s
}

// Close releases the local index. The Client must not be used afterwards.
func (c *Client) Close() error {
	return c.idx.Close()
}
