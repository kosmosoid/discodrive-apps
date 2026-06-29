# DiscoKit

Shared Swift core for native DiscoDrive clients. No UI — pure logic covered by
unit tests. Reused by the macOS app (`clients/macos`) and the iOS client (`clients/ios`).

## Modules

- **`APIClient`** — HTTP to the server: exchanges a device token for a JWT (with 401-retry),
  `/sync/changes` (delta + `allChanges` pagination), `/files/{id}/content` (`download`).
- **`IndexStore`** — local SQLite mirror of the node tree (GRDB): `apply(changes)`,
  `children(of:)`, `node(id:)`, cursor. `parent_id` is derived from `path` in `node_id`.
- **`LocalStore`** — local content copies on disk + `none/cached/pinned/stale` statuses,
  `store/pin/unpin/remove/evictCached`.
- **`Pairing`** — device-code flow (`start`/`poll`). **`KeychainToken`** — token and server
  address stored in Keychain.
- **`Models`** — `Node`, `RemoteChange`, `ChangesPage`.

## Tests

```bash
cd clients/DiscoKit
swift test
```
Network is mocked via `MockURLProtocol`; DB is an in-memory GRDB instance. No Apple account or signing required.
