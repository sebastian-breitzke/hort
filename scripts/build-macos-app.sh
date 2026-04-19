#!/usr/bin/env bash
#
# Wrap a hort binary into a macOS .app bundle and zip it.
# Usage: build-macos-app.sh <binary> <icns> <version> <out-zip>
#
# The .app is a CLI-style bundle (LSUIElement=true, no Dock icon). Its primary
# purpose is icon visibility in Finder/Spotlight and providing a notarization
# target. Day-to-day invocation is still `hort` from PATH (Homebrew symlink).
set -euo pipefail

if [[ $# -ne 4 ]]; then
    echo "usage: $0 <binary> <icns> <version> <out-zip>" >&2
    exit 2
fi

BINARY="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
ICNS="$(cd "$(dirname "$2")" && pwd)/$(basename "$2")"
VERSION="$3"
OUT_ZIP="$4"

[[ -f "$BINARY" ]] || { echo "binary not found: $BINARY" >&2; exit 1; }
[[ -f "$ICNS" ]]   || { echo "icns not found: $ICNS" >&2; exit 1; }

# Resolve OUT_ZIP to absolute path (may not exist yet)
mkdir -p "$(dirname "$OUT_ZIP")"
OUT_ZIP="$(cd "$(dirname "$OUT_ZIP")" && pwd)/$(basename "$OUT_ZIP")"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

APP="$WORK/hort.app"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp "$BINARY" "$APP/Contents/MacOS/hort"
chmod +x "$APP/Contents/MacOS/hort"
cp "$ICNS" "$APP/Contents/Resources/hort.icns"

cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>en</string>
    <key>CFBundleExecutable</key>
    <string>hort</string>
    <key>CFBundleIconFile</key>
    <string>hort</string>
    <key>CFBundleIdentifier</key>
    <string>de.s16e.hort</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>hort</string>
    <key>CFBundleDisplayName</key>
    <string>Hort</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundleVersion</key>
    <string>${VERSION}</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.15</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
PLIST

python3 - "$WORK" "$OUT_ZIP" <<'PY'
import os, stat, sys, zipfile
src_dir, out_zip = sys.argv[1], sys.argv[2]
with zipfile.ZipFile(out_zip, "w", zipfile.ZIP_DEFLATED) as zf:
    for root, _, files in os.walk(src_dir):
        for f in files:
            p = os.path.join(root, f)
            rel = os.path.relpath(p, src_dir)
            zi = zipfile.ZipInfo.from_file(p, rel)
            # Preserve +x on the binary
            mode = os.stat(p).st_mode
            zi.external_attr = (mode & 0xFFFF) << 16
            with open(p, "rb") as fh:
                zf.writestr(zi, fh.read(), zipfile.ZIP_DEFLATED)
PY
echo "wrote $OUT_ZIP"
