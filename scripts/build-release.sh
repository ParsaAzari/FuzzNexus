#!/usr/bin/env bash
# ==============================================================================
# FuzzNexus Universal Distribution Pipeline
# Optimized release builds for supported desktop platforms
# Version: 1.0.0
# Developed by Parsa Azari (https://parsaazari.com)
# Copyright (c) 2026 Parsa Azari. Licensed under the MIT License.
# ==============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DESKTOP_DIR="$ROOT_DIR/fuzznexus-desktop"
DIST_DIR="$ROOT_DIR/dist"
STAGE_DIR="$DIST_DIR/.stage"
VERSION="1.0.0"
APP_NAME="FuzzNexus"

# Terminal Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'
BOLD='\033[1m'

echo -e "${CYAN}${BOLD}"
echo "=================================================================="
echo "      FuzzNexus Universal Production Pipeline                     "
echo "              (Optimized release builds)                          "
echo "                     Version: $VERSION                            "
echo "          Developed by Parsa Azari (parsaazari.com)               "
echo "=================================================================="
echo -e "${NC}"

TARGET="${1:-all}"
INSTALL_LOCAL=false
for arg in "$@"; do
    if [ "$arg" = "--install" ]; then
        INSTALL_LOCAL=true
    fi
done

# Ensure directories
mkdir -p "$DIST_DIR" "$STAGE_DIR"

# Ensure tools are in PATH
export PATH="$PATH:$HOME/go/bin:/opt/homebrew/bin:/usr/local/bin"

# Release compilation flags:
# -s: Omit symbol table
# -w: Omit DWARF debugging info
# -trimpath: Remove all local system paths from compiled binaries
LDFLAGS_RELEASE="-s -w -X main.version=$VERSION"

# ------------------------------------------------------------------------------
# Helper: Generate complete 10-layer Apple ICNS with iconutil
# ------------------------------------------------------------------------------
generate_apple_icns() {
    local SRC_PNG="$DESKTOP_DIR/build/appicon.png"
    local TARGET_ICNS="$1"
    local ICONSET_DIR
    ICONSET_DIR=$(mktemp -d /tmp/FuzzNexus_icns.XXXXXX.iconset)

    python3 -c "
import os
from PIL import Image

src = Image.open('$SRC_PNG')
specs = [
    ('icon_16x16.png', 16),
    ('icon_16x16@2x.png', 32),
    ('icon_32x32.png', 32),
    ('icon_32x32@2x.png', 64),
    ('icon_128x128.png', 128),
    ('icon_128x128@2x.png', 256),
    ('icon_256x256.png', 256),
    ('icon_256x256@2x.png', 512),
    ('icon_512x512.png', 512),
    ('icon_512x512@2x.png', 1024),
]
for name, sz in specs:
    src.resize((sz, sz), Image.Resampling.LANCZOS).save(os.path.join('$ICONSET_DIR', name), 'PNG')
"
    iconutil -c icns "$ICONSET_DIR" -o "$TARGET_ICNS"
    rm -rf "$ICONSET_DIR"
}

# ------------------------------------------------------------------------------
# 1. macOS Universal 2 (Apple Silicon + Intel) DMG
# ------------------------------------------------------------------------------
build_macos() {
    local APP_PATH="$DESKTOP_DIR/build/bin/$APP_NAME.app"
    echo -e "${YELLOW}==> [1/3] Building macOS Universal Bundle (arm64 + x86_64)...${NC}"

    generate_apple_icns "$DESKTOP_DIR/build/darwin/iconfile.icns"

    find "$DESKTOP_DIR" -name ".DS_Store" -delete 2>/dev/null || true
    xattr -cr "$DESKTOP_DIR" 2>/dev/null || true
    rm -rf "$APP_PATH"

    echo -e "${CYAN}--> Building optimized frontend (no source maps)...${NC}"
    (cd "$DESKTOP_DIR/frontend" && npm run build)

    echo -e "${CYAN}--> Compiling Universal Mach-O with release flags...${NC}"
    cd "$DESKTOP_DIR"
    wails build -platform "darwin/universal" -trimpath -ldflags="$LDFLAGS_RELEASE"

    if [ ! -d "$APP_PATH" ]; then
        echo -e "${RED}Error: App bundle not found at $APP_PATH${NC}"
        exit 1
    fi

    # Strip any remaining debug symbols from the universal Mach-O binary
    strip -S "$APP_PATH/Contents/MacOS/$APP_NAME" 2>/dev/null || true

    echo -e "${CYAN}--> Verifying Universal Mach-O slices...${NC}"
    file "$APP_PATH/Contents/MacOS/$APP_NAME"

    cp "$DESKTOP_DIR/build/darwin/iconfile.icns" "$APP_PATH/Contents/Resources/iconfile.icns"

    # Remove Finder metadata before signing so it cannot invalidate the bundle
    # signature.
    xattr -cr "$APP_PATH"
    codesign --force --deep -s - "$APP_PATH"
    touch "$APP_PATH"

    /System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/LaunchServices.framework/Versions/A/Support/lsregister -f "$APP_PATH" 2>/dev/null || true

    local DMG_OUTPUT="$DIST_DIR/${APP_NAME}-${VERSION}-macOS-Universal.dmg"
    rm -f "$DMG_OUTPUT"

    echo -e "${CYAN}--> Packaging compressed macOS Universal DMG...${NC}"
    local TMP_DMG_STAGE
    TMP_DMG_STAGE=$(mktemp -d /tmp/fn_dmg_stage.XXXXXX)
    ditto "$APP_PATH" "$TMP_DMG_STAGE/$APP_NAME.app"
    xattr -cr "$TMP_DMG_STAGE/$APP_NAME.app"
    codesign --force --deep -s - "$TMP_DMG_STAGE/$APP_NAME.app"

    create-dmg \
        --volname "$APP_NAME" \
        --volicon "$APP_PATH/Contents/Resources/iconfile.icns" \
        --background "$DESKTOP_DIR/build/darwin/dmg-background-light@2x.png" \
        --window-pos 250 150 \
        --window-size 660 420 \
        --icon-size 110 \
        --icon "$APP_NAME.app" 175 210 \
        --hide-extension "$APP_NAME.app" \
        --app-drop-link 485 210 \
        --no-internet-enable \
        "$DMG_OUTPUT" \
        "$TMP_DMG_STAGE" > /dev/null 2>&1 || true

    rm -rf "$TMP_DMG_STAGE"

    if [ -f "$DMG_OUTPUT" ]; then
        local SIZE
        SIZE=$(ls -lh "$DMG_OUTPUT" | awk '{print $5}')
        echo -e "${GREEN}✓ Created macOS Universal DMG: $DMG_OUTPUT ($SIZE)${NC}"
    else
        echo -e "${RED}Error: DMG creation failed${NC}"
        exit 1
    fi

    if [ "$INSTALL_LOCAL" = true ] || [ "$TARGET" = "mac" ] || [ "$TARGET" = "dmg" ]; then
        echo -e "${CYAN}--> Installing / Updating /Applications/$APP_NAME.app...${NC}"
        rm -rf "/Applications/$APP_NAME.app"
        ditto "$APP_PATH" "/Applications/$APP_NAME.app"
        xattr -cr "/Applications/$APP_NAME.app"
        /System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/LaunchServices.framework/Versions/A/Support/lsregister -f "/Applications/$APP_NAME.app" 2>/dev/null || true
        echo -e "${GREEN}✓ Successfully installed $APP_NAME.app in /Applications!${NC}"
    fi
}

# ------------------------------------------------------------------------------
# 2. Windows x86_64 Package
# ------------------------------------------------------------------------------
build_windows() {
    echo -e "\n${YELLOW}==> [2/3] Building Windows (x86_64) package...${NC}"

    cd "$DESKTOP_DIR"
    wails build -platform "windows/amd64" -trimpath -ldflags="$LDFLAGS_RELEASE"

    local EXE_SRC="$DESKTOP_DIR/build/bin/${APP_NAME}.exe"
    if [ ! -f "$EXE_SRC" ]; then
        echo -e "${RED}Error: Windows executable not found at $EXE_SRC${NC}"
        exit 1
    fi

    local ZIP_DIST="$DIST_DIR/${APP_NAME}-${VERSION}-Windows-x64.zip"
    rm -f "$ZIP_DIST"

    local TMP_WIN_DIR
    TMP_WIN_DIR=$(mktemp -d /tmp/fn_win.XXXXXX)
    cp "$EXE_SRC" "$TMP_WIN_DIR/${APP_NAME}.exe"

    cat << 'EOF' > "$TMP_WIN_DIR/README.txt"
FuzzNexus v1.0.0
Professional Fast Web Fuzzer & Security Assessment Platform
Developed by Parsa Azari (https://parsaazari.com)

Instructions:
1. Double-click FuzzNexus.exe to run.
2. Requirements: Windows 10/11 64-bit with Microsoft WebView2 Runtime.

Open source software licensed under the MIT License.
Copyright (c) 2026 Parsa Azari.
EOF

    (cd "$TMP_WIN_DIR" && zip -9 -q -r "$ZIP_DIST" .)
    rm -rf "$TMP_WIN_DIR"

    local SIZE_ZIP
    SIZE_ZIP=$(ls -lh "$ZIP_DIST" | awk '{print $5}')
    echo -e "${GREEN}✓ Created Windows package: $ZIP_DIST ($SIZE_ZIP)${NC}"
}

# ------------------------------------------------------------------------------
# 3. Linux x86_64 Desktop Package
# ------------------------------------------------------------------------------
build_linux() {
    echo -e "\n${YELLOW}==> [3/3] Building Linux (x86_64) desktop package...${NC}"

    cd "$ROOT_DIR"
    local CLI_OUTPUT_AMD64="$DIST_DIR/.stage_fuzznexus_linux_bin"
    GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$LDFLAGS_RELEASE" -o "$CLI_OUTPUT_AMD64" .

    local LINUX_STAGE
    LINUX_STAGE=$(mktemp -d /tmp/fn_linux.XXXXXX)
    local BUNDLE_DIR="$LINUX_STAGE/FuzzNexus-${VERSION}"
    mkdir -p "$BUNDLE_DIR/bin" "$BUNDLE_DIR/share/applications" "$BUNDLE_DIR/share/icons"

    cp "$CLI_OUTPUT_AMD64" "$BUNDLE_DIR/bin/fuzznexus"
    rm -f "$CLI_OUTPUT_AMD64"
    if [ -d "$DESKTOP_DIR/build/linux/icons" ]; then
        cp -R "$DESKTOP_DIR/build/linux/icons" "$BUNDLE_DIR/share/"
    fi
    if [ -f "$DESKTOP_DIR/build/linux/fuzznexus.desktop" ]; then
        cp "$DESKTOP_DIR/build/linux/fuzznexus.desktop" "$BUNDLE_DIR/share/applications/"
    fi

    cat << 'EOF' > "$BUNDLE_DIR/install.sh"
#!/usr/bin/env bash
set -e
TARGET_PREFIX="${HOME}/.local"
echo "Installing FuzzNexus to ${TARGET_PREFIX}..."
mkdir -p "${TARGET_PREFIX}/bin" "${TARGET_PREFIX}/share/applications" "${TARGET_PREFIX}/share/icons"
cp bin/fuzznexus "${TARGET_PREFIX}/bin/"
chmod +x "${TARGET_PREFIX}/bin/fuzznexus"
cp share/applications/fuzznexus.desktop "${TARGET_PREFIX}/share/applications/"
cp -R share/icons/* "${TARGET_PREFIX}/share/icons/"
echo "FuzzNexus installed successfully. Launch it with 'fuzznexus' or from your applications menu."
EOF
    chmod +x "$BUNDLE_DIR/install.sh"

    local TAR_DIST="$DIST_DIR/${APP_NAME}-${VERSION}-Linux-x86_64.tar.gz"
    rm -f "$TAR_DIST"
    (cd "$LINUX_STAGE" && GZIP="-9" tar -czf "$TAR_DIST" "FuzzNexus-${VERSION}")
    rm -rf "$LINUX_STAGE"

    local SIZE_TAR
    SIZE_TAR=$(ls -lh "$TAR_DIST" | awk '{print $5}')
    echo -e "${GREEN}✓ Created Linux package: $TAR_DIST ($SIZE_TAR)${NC}"
}

cleanup() {
    rm -rf "$STAGE_DIR"
    rm -rf "$DIST_DIR/cli" "$DIST_DIR/.stage" "$DIST_DIR/FuzzNexus-1.0.0-windows-amd64" 2>/dev/null || true
}

# ------------------------------------------------------------------------------
# Dispatcher
# ------------------------------------------------------------------------------
case "$TARGET" in
    mac|darwin|dmg)
        build_macos
        cleanup
        ;;
    win|windows|exe)
        build_windows
        cleanup
        ;;
    linux|tar)
        build_linux
        cleanup
        ;;
    all)
        rm -rf "$DIST_DIR"/*
        build_macos
        build_windows
        build_linux
        cleanup
        ;;
    *)
        echo -e "${RED}Usage: $0 [all|mac|win|linux] [--install]${NC}"
        exit 1
        ;;
esac

echo -e "\n${GREEN}${BOLD}Target distribution packages ready (Exactly 3 Files):${NC}"
ls -lh "$DIST_DIR"
