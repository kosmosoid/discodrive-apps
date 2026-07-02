# Changelog

All notable changes to this project are documented in this file.

## 0.0.2

### Security

- Hardening after an internal security audit: tightened validation of server-provided
  paths across the sync, desktop, and mobile file paths, and improved handling of the
  self-signed / insecure-TLS option.

## 0.0.1

First public release of DiscoDrive — a client for syncing with DiscoDrive server.

### Added

- Desktop app for macOS, Windows, and Linux.
- Sync daemon for darwin / linux / windows (amd64 and arm64).
- Android app (debug build).
