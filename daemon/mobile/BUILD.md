# Building the mobile bindings

Artifacts: `mobile/build/Kfmobile.xcframework` (iOS) and `mobile/build/kfmobile.aar` (Android).

## Toolchain setup (once, macOS)

    go install golang.org/x/mobile/cmd/gomobile@latest
    go install golang.org/x/mobile/cmd/gobind@latest
    gomobile init

iOS: requires Xcode (`xcode-select -p` must resolve).

Android: requires the SDK + NDK.

    export ANDROID_HOME="$HOME/Library/Android/sdk"
    export ANDROID_NDK_HOME="$ANDROID_HOME/ndk/<version>"

Install the NDK via Android Studio or `sdkmanager "ndk;<version>" "platforms;android-21"`.

## Building

From the `daemon/` directory:

    mobile/build.sh

For a single platform only — use the matching `gomobile bind` line from `build.sh`.

## If the bind fails

- `modernc.org/sqlite` on arm64: confirm the error is actually from it; swap the index.
- `golang.org/x/mobile` incompatible with Go 1.25: install a compatible gomobile version.
