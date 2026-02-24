#!/bin/bash
# Build a Linux AppImage.
#
# Usage: ./packaging/linux/build.sh <binary> <version>
# Example: ./packaging/linux/build.sh r1ptt-linux-amd64 1.0.0
#
# Requires: libusb-1.0 installed on the build system
# Downloads: appimagetool (if not already present)

set -euo pipefail

BINARY="${1:?Usage: build.sh <binary> <version>}"
VERSION="${2:?Usage: build.sh <binary> <version>}"
VERSION="${VERSION#v}"  # strip leading v so filename is R1-Control-v1.0.0 not vv1.0.0

APP_NAME="R1Control"
APPDIR="${APP_NAME}.AppDir"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "==> Building AppImage v${VERSION}"

# Clean
rm -rf "${APPDIR}"

# Create AppDir structure
mkdir -p "${APPDIR}/usr/bin"
mkdir -p "${APPDIR}/usr/lib"

# Copy binary
cp "${BINARY}" "${APPDIR}/usr/bin/r1control"
chmod +x "${APPDIR}/usr/bin/r1control"

# Copy AppRun
cp "${SCRIPT_DIR}/AppRun" "${APPDIR}/AppRun"
chmod +x "${APPDIR}/AppRun"

# Copy desktop file
cp "${SCRIPT_DIR}/r1control.desktop" "${APPDIR}/r1control.desktop"

# Copy icon (use assets/icon.png if available, otherwise create a placeholder)
if [ -f "${PROJECT_DIR}/assets/icon.png" ]; then
    cp "${PROJECT_DIR}/assets/icon.png" "${APPDIR}/icon.png"
else
    echo "==> Warning: assets/icon.png not found"
    # Create a minimal 1x1 PNG as placeholder
    printf '\x89PNG\r\n\x1a\n' > "${APPDIR}/icon.png"
fi

# Bundle libusb
LIBUSB_PATH=""
for candidate in \
    /usr/lib/x86_64-linux-gnu/libusb-1.0.so.0 \
    /usr/lib64/libusb-1.0.so.0 \
    /usr/lib/libusb-1.0.so.0; do
    if [ -f "${candidate}" ]; then
        LIBUSB_PATH="${candidate}"
        break
    fi
done

if [ -n "${LIBUSB_PATH}" ]; then
    echo "==> Bundling libusb from ${LIBUSB_PATH}"
    cp "${LIBUSB_PATH}" "${APPDIR}/usr/lib/"
else
    echo "==> Warning: libusb-1.0.so.0 not found, not bundling"
fi

# Get appimagetool if not present
APPIMAGETOOL="./appimagetool-x86_64.AppImage"
if [ ! -f "${APPIMAGETOOL}" ]; then
    echo "==> Downloading appimagetool"
    curl -sL -o "${APPIMAGETOOL}" \
        "https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage"
    chmod +x "${APPIMAGETOOL}"
fi

# Build AppImage
OUTPUT="R1-Control-v${VERSION}-linux-amd64.AppImage"
echo "==> Creating ${OUTPUT}"
ARCH=x86_64 "${APPIMAGETOOL}" "${APPDIR}" "${OUTPUT}"

echo "==> Done: ${OUTPUT}"
