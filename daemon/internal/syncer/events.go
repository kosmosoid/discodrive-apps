package syncer

import (
	"bufio"
	"context"
	"strings"

	"discodrive.org/daemon/internal/protocol"
)

// listenEvents holds a GET /sync/events connection and calls notify on each data event.
func listenEvents(ctx context.Context, client *protocol.Client, notify func()) error {
	resp, err := client.Events(ctx)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if strings.HasPrefix(sc.Text(), "data:") {
			notify()
		}
	}
	return sc.Err()
}
