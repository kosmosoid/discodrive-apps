# DiscoDrive — native macOS client (standalone)

A windowed app for accessing DiscoDrive files: displays the full tree, fetches content
**on demand**, and lets you **keep individual files local**. No Finder integration.

## Why standalone and not File Provider

Finder integration (File Provider) requires a sandbox extension, and App Sandbox under
a **free** Apple ID is rejected by the macOS kernel (AMFI: ad-hoc/local signing + sandbox
= denied). A paid Apple Developer account ($99) is required. So files live in an app
window rather than in Finder. A File Provider variant is parked outside this repo
until a paid Apple Developer account is available.

## Requirements

- Xcode 26+, [XcodeGen](https://github.com/yonaskolb/XcodeGen) (`brew install xcodegen`).
- Set your `DEVELOPMENT_TEAM` in `project.yml` (Xcode → Settings → Accounts → Team ID), or
  switch signing to "Sign to Run Locally" in Xcode.

## Build and run

```bash
cd clients/macos
xcodegen generate
xcodebuild -project DiscoDrive.xcodeproj -scheme DiscoDrive -derivedDataPath build build
open build/Build/Products/Debug/DiscoDrive.app
```
Or open `DiscoDrive.xcodeproj` in Xcode (scheme **DiscoDrive**) and click Run.

## Pairing

On first launch, enter your DiscoDrive server address → click "Connect" → confirm the
device in the browser that opens. The token is saved in Keychain.

## What it can do (v1)

- Browse the full file tree (folder tree + list view, navigate up to the "DiscoDrive" root).
- Download a file on double-click (on-demand) and open it in an external app.
- "Keep local" (📌) — pinned copies are never evicted.
- "Free up space" — clears non-pinned copies.
- Multilingual UI (language is stored on the server and synced across devices).

## Out of scope for v1

File upload/editing on the server, encrypted vault (Cryptomator), iOS app,
notarization. iOS will reuse the `clients/DiscoKit` core.

## Distribution

A release without a paid account is signed locally → Gatekeeper will require
"right-click → Open" on first launch. Notarization is unavailable without $99.
