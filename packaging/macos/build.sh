#!/bin/bash
# Build a macOS .app bundle and .dmg disk image.
#
# Usage: ./packaging/macos/build.sh <binary> <version> <arch>
# Example: ./packaging/macos/build.sh r1ptt-darwin-arm64 1.0.0 arm64
#
# Requires: sips, iconutil, install_name_tool, codesign, hdiutil
# Optional: assets/icon.png (1024x1024) for app icon
#           packaging/macos/dmg-background.png (1320x800 @2x Retina)
#             — window is 660x400; app icon at (165,185), Applications at (495,185)

set -euo pipefail

BINARY="${1:?Usage: build.sh <binary> <version> <arch>}"
VERSION="${2:?Usage: build.sh <binary> <version> <arch>}"
VERSION="${VERSION#v}"  # strip leading v so filename is R1-Control-v1.0.0 not vv1.0.0
ARCH="${3:?Usage: build.sh <binary> <version> <arch>}"

APP_NAME="R1 Control"
APP_DIR="${APP_NAME}.app"
CONTENTS="${APP_DIR}/Contents"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "==> Building ${APP_NAME} v${VERSION} (${ARCH})"

# Clean
rm -rf "${APP_DIR}"

# Create .app structure
mkdir -p "${CONTENTS}/MacOS"
mkdir -p "${CONTENTS}/Resources"
mkdir -p "${CONTENTS}/Frameworks"

# Copy binary
cp "${BINARY}" "${CONTENTS}/MacOS/r1control"
chmod +x "${CONTENTS}/MacOS/r1control"

# Copy and patch Info.plist
sed "s/\${VERSION}/${VERSION}/g" "${SCRIPT_DIR}/Info.plist" > "${CONTENTS}/Info.plist"

# Generate .icns from icon.png if available
ICON_SRC="${PROJECT_DIR}/assets/icon.png"
if [ -f "${ICON_SRC}" ]; then
    echo "==> Generating icon.icns from assets/icon.png"
    ICONSET="${CONTENTS}/Resources/icon.iconset"
    mkdir -p "${ICONSET}"

    # Generate all required sizes
    for SIZE in 16 32 128 256 512; do
        sips -z ${SIZE} ${SIZE} "${ICON_SRC}" --out "${ICONSET}/icon_${SIZE}x${SIZE}.png" >/dev/null 2>&1
        DOUBLE=$((SIZE * 2))
        sips -z ${DOUBLE} ${DOUBLE} "${ICON_SRC}" --out "${ICONSET}/icon_${SIZE}x${SIZE}@2x.png" >/dev/null 2>&1
    done

    iconutil -c icns "${ICONSET}" -o "${CONTENTS}/Resources/icon.icns"
    rm -rf "${ICONSET}"
else
    echo "==> Warning: assets/icon.png not found, skipping icon"
fi

# Bundle libusb dylib
LIBUSB_PATH=""
for candidate in /opt/homebrew/lib/libusb-1.0.0.dylib /usr/local/lib/libusb-1.0.0.dylib; do
    if [ -f "${candidate}" ]; then
        LIBUSB_PATH="${candidate}"
        break
    fi
done

if [ -n "${LIBUSB_PATH}" ]; then
    echo "==> Bundling libusb from ${LIBUSB_PATH}"
    cp "${LIBUSB_PATH}" "${CONTENTS}/Frameworks/"

    # Get the install name of libusb as linked in the binary
    LIBUSB_ID=$(otool -L "${CONTENTS}/MacOS/r1control" | grep libusb | awk '{print $1}')

    if [ -n "${LIBUSB_ID}" ]; then
        # Rewrite the binary to load libusb from Frameworks/
        install_name_tool -change "${LIBUSB_ID}" \
            "@executable_path/../Frameworks/libusb-1.0.0.dylib" \
            "${CONTENTS}/MacOS/r1control"

        # Fix the dylib's own install name
        install_name_tool -id \
            "@executable_path/../Frameworks/libusb-1.0.0.dylib" \
            "${CONTENTS}/Frameworks/libusb-1.0.0.dylib"
    fi
else
    echo "==> Warning: libusb not found, not bundling"
fi

# Ad-hoc code sign
echo "==> Code signing (ad-hoc)"
codesign --force --deep -s - "${APP_DIR}"

# ── Create polished DMG via create-dmg ───────────────────────────────────────
DMG_NAME="R1-Control-v${VERSION}-macos-${ARCH}.dmg"
BG_IMAGE="${SCRIPT_DIR}/dmg-background.png"

echo "==> Creating ${DMG_NAME}"

# Remove any stale output so create-dmg doesn't prompt
rm -f "${DMG_NAME}"

CREATE_DMG_ARGS=(
    --volname "${APP_NAME}"
    --window-pos 200 120
    --window-size 660 400
    --icon-size 128
    --icon "${APP_NAME}.app" 165 185
    --app-drop-link 495 185
    --hide-extension "${APP_NAME}.app"
    --no-internet-enable
)

if [ -f "${BG_IMAGE}" ]; then
    # create-dmg needs a 1x image to set the window size and a @2x for Retina.
    # The source is 1320x800 (2x), so generate a 660x400 (1x) copy on the fly.
    BG_1X="/tmp/dmg-background-1x.png"
    BG_2X="/tmp/dmg-background@2x.png"
    sips -z 400 660 "${BG_IMAGE}" --out "${BG_1X}" >/dev/null
    cp "${BG_IMAGE}" "${BG_2X}"
    CREATE_DMG_ARGS+=(--background "${BG_1X}")
fi

create-dmg "${CREATE_DMG_ARGS[@]}" "${DMG_NAME}" "${APP_DIR}"

# Clean up temp background files
rm -f "${BG_1X:-}" "${BG_2X:-}"

echo "==> Done: ${DMG_NAME}"
