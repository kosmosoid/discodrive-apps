# DiscoDrive (Android)

Full file client: pair + browse the whole storage (offline index), download/open/pin/upload,
mkdir/rename/move/delete. Files live in `/storage/emulated/0/DiscoDrive` (visible to any app).

## Build
1. Block B/1a: from `daemon/` — `mobile/build.sh` (needs `kfmobile.aar` of type `Browser`).
2. `cp ../../daemon/mobile/build/kfmobile.aar app/libs/kfmobile.aar`
3. Open `clients/android-discodrive/` in Android Studio, let it sync (accept the AGP/Gradle upgrade if prompted).
4. Pick a device, Run.

## Usage
1. Grant **All files access**.
2. Server (+ "Self-signed" if mkcert), Pair → confirm the code in the browser.
3. Navigate the tree; "New folder"/"Upload" menu; per file — Open/Download/Pin/Unpin/Remove local/Rename/Move/Delete.
4. KeePass opens its database from `/storage/emulated/0/DiscoDrive/...`.
