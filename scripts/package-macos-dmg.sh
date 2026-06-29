#!/usr/bin/env bash
# Package the built Wails macOS app into a drag-to-Applications .dmg.
# Usage: scripts/package-macos-dmg.sh [version]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
APP="$ROOT/daemon/cmd/discodrive-wails/build/bin/discodrive-wails.app"
VERSION="${1:-$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$APP/Contents/Info.plist")}"
DIST="$ROOT/dist"
DMG="$DIST/DiscoDrive-$VERSION-macos.dmg"

[ -d "$APP" ] || { echo "ERROR: build first — $APP missing"; exit 1; }
mkdir -p "$DIST"
rm -f "$DMG"

STAGE="$(mktemp -d)"
cp -R "$APP" "$STAGE/DiscoDrive.app"
ln -s /Applications "$STAGE/Applications"

hdiutil create -volname "DiscoDrive" -srcfolder "$STAGE" -ov -format UDZO "$DMG"
rm -rf "$STAGE"
echo "built → $DMG"
