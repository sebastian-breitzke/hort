#!/usr/bin/env bash
#
# Build a signed, notarized hort universal binary and distribute it as:
#   - dist/hort_<ver>_darwin_universal.tar.gz  (bare CLI binary, signed+notarized)
#   - dist/hort_<ver>_darwin_universal.dmg     (hort.app, signed+notarized+stapled)
#
# Requires (for notarization):
#   APPLE_ID, APPLE_TEAM_ID, APPLE_APP_SPECIFIC_PASSWORD
#   A Developer ID Application certificate in the default keychain.
#
# Usage:
#   ./scripts/build-macos-signed.sh <version>            # build + sign only
#   ./scripts/build-macos-signed.sh <version> --notarize # + notarize + DMG
set -euo pipefail

VERSION="${1:?usage: $0 <version> [--notarize]}"
shift || true

DO_NOTARIZE=0
for arg in "$@"; do
    case "$arg" in
        --notarize) DO_NOTARIZE=1 ;;
        *) echo "unknown flag: $arg" >&2; exit 1 ;;
    esac
done

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
ENT="$ROOT/scripts/entitlements.plist"
ICNS="$ROOT/icons/macos/hort.icns"
BIN="$DIST/hort"
APP="$DIST/hort.app"
TARBALL="$DIST/hort_${VERSION}_darwin_universal.tar.gz"
DMG="$DIST/hort_${VERSION}_darwin_universal.dmg"

mkdir -p "$DIST"

echo "==> building universal binary"
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o "$DIST/hort-amd64" "$ROOT/cmd/hort"
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w" -o "$DIST/hort-arm64" "$ROOT/cmd/hort"
lipo -create -output "$BIN" "$DIST/hort-amd64" "$DIST/hort-arm64"
rm "$DIST/hort-amd64" "$DIST/hort-arm64"
file "$BIN"

IDENTITY=$(security find-identity -v -p codesigning | grep "Developer ID Application" | head -1 | sed 's/.*"\(.*\)"/\1/')
if [[ -z "$IDENTITY" ]]; then
    echo "ERROR: no Developer ID Application certificate in keychain" >&2
    exit 1
fi
echo "==> signing as: $IDENTITY"

SIGN_OPTS=(--force --sign "$IDENTITY" --timestamp --options runtime --entitlements "$ENT")

echo "==> signing bare binary"
codesign "${SIGN_OPTS[@]}" "$BIN"
codesign --verify --verbose=2 --strict "$BIN"

echo "==> wrapping into hort.app"
bash "$ROOT/scripts/build-macos-app.sh" "$BIN" "$ICNS" "$VERSION" "$DIST/_app_staging.zip" >/dev/null
rm -rf "$APP"
(cd "$DIST" && unzip -q _app_staging.zip && rm _app_staging.zip)

echo "==> signing hort.app"
codesign "${SIGN_OPTS[@]}" "$APP/Contents/MacOS/hort"
codesign "${SIGN_OPTS[@]}" "$APP"
codesign --verify --verbose=2 --strict "$APP"

if [[ "$DO_NOTARIZE" == "1" ]]; then
    : "${APPLE_ID:?APPLE_ID not set}"
    : "${APPLE_TEAM_ID:?APPLE_TEAM_ID not set}"
    : "${APPLE_APP_SPECIFIC_PASSWORD:?APPLE_APP_SPECIFIC_PASSWORD not set}"

    echo "==> notarizing hort.app"
    NOTARY_ZIP="$DIST/_notary.zip"
    ditto -c -k --keepParent --sequesterRsrc "$APP" "$NOTARY_ZIP"
    xcrun notarytool submit "$NOTARY_ZIP" \
        --apple-id "$APPLE_ID" \
        --team-id "$APPLE_TEAM_ID" \
        --password "$APPLE_APP_SPECIFIC_PASSWORD" \
        --wait --timeout 30m
    rm "$NOTARY_ZIP"
    echo "==> stapling hort.app"
    xcrun stapler staple "$APP"
    xcrun stapler validate "$APP"
fi

echo "==> packaging bare binary: $TARBALL"
(cd "$DIST" && tar -czf "$(basename "$TARBALL")" hort)

if [[ "$DO_NOTARIZE" == "1" ]]; then
    echo "==> building DMG: $DMG"
    rm -f "$DMG"
    STAGING="$DIST/_dmg"
    rm -rf "$STAGING" && mkdir -p "$STAGING"
    cp -R "$APP" "$STAGING/"
    ln -s /Applications "$STAGING/Applications"
    hdiutil create -volname "hort" -srcfolder "$STAGING" -ov -format UDZO "$DMG"
    rm -rf "$STAGING"

    echo "==> notarizing DMG"
    xcrun notarytool submit "$DMG" \
        --apple-id "$APPLE_ID" \
        --team-id "$APPLE_TEAM_ID" \
        --password "$APPLE_APP_SPECIFIC_PASSWORD" \
        --wait --timeout 30m
    echo "==> stapling DMG"
    xcrun stapler staple "$DMG"
    xcrun stapler validate "$DMG"
fi

echo
echo "✓ tarball:  $TARBALL"
[[ "$DO_NOTARIZE" == "1" ]] && echo "✓ dmg:      $DMG"
echo "✓ binary:   $BIN"
echo "✓ app:      $APP"
