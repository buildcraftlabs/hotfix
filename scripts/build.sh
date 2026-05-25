#!/usr/bin/env bash
set -euo pipefail

# ─────────────────────────────────────────────────────────────
# MCP Killer — Build Script
# BuildCraft Labs · 2025
# ─────────────────────────────────────────────────────────────

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"
DIST_DIR="$PROJECT_ROOT/dist"
BUILD_DIR="$PROJECT_ROOT/.build/apple/Products/Release"
APP_NAME="Hotfix"
DISPLAY_NAME="Hotfix"
APP_BUNDLE="$DIST_DIR/$APP_NAME.app"
DMG_NAME="MCP Killer"
DMG_OUT="$DIST_DIR/$DMG_NAME.dmg"
ICON_DIR="$PROJECT_ROOT/icon"
SVG_SOURCE="$ICON_DIR/AppIcon.svg"
ICONSET_DIR="$ICON_DIR/AppIcon.iconset"
ICNS_OUT="$ICON_DIR/AppIcon.icns"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

log()    { echo -e "${CYAN}[MCP Killer]${NC} $*"; }
ok()     { echo -e "${GREEN}[✓]${NC} $*"; }
warn()   { echo -e "${YELLOW}[!]${NC} $*"; }
die()    { echo -e "${RED}[✗] $*${NC}" >&2; exit 1; }

# ─────────────────────────────────────────────────────────────
# 1. Prerequisites check
# ─────────────────────────────────────────────────────────────
log "Checking prerequisites..."

if ! command -v xcode-select &>/dev/null; then
    die "Xcode command line tools not found. Run: xcode-select --install"
fi

XCODE_PATH=$(xcode-select -p 2>/dev/null || true)
if [[ -z "$XCODE_PATH" ]]; then
    die "Xcode command line tools path not set. Run: xcode-select --install"
fi

if ! command -v swift &>/dev/null; then
    die "swift compiler not found. Install Xcode or Xcode Command Line Tools."
fi

SWIFT_VERSION=$(swift --version 2>&1 | head -1)
ok "Swift found: $SWIFT_VERSION"

# ─────────────────────────────────────────────────────────────
# 2. Build universal binary
# ─────────────────────────────────────────────────────────────
log "${BOLD}Building universal binary (arm64 + x86_64)...${NC}"
cd "$PROJECT_ROOT"

swift build -c release --arch arm64 --arch x86_64 2>&1 || {
    # Fallback: build for native arch only if universal fails (e.g. CI without Rosetta)
    warn "Universal build failed, falling back to native arch..."
    swift build -c release
    BUILD_DIR="$PROJECT_ROOT/.build/release"
}

BINARY_PATH=""
# Search for the built binary
for candidate in \
    "$PROJECT_ROOT/.build/apple/Products/Release/$APP_NAME" \
    "$PROJECT_ROOT/.build/release/$APP_NAME"; do
    if [[ -f "$candidate" ]]; then
        BINARY_PATH="$candidate"
        break
    fi
done

if [[ -z "$BINARY_PATH" ]]; then
    die "Could not locate built binary. Check build output above."
fi

ok "Binary built: $BINARY_PATH"

# ─────────────────────────────────────────────────────────────
# 3. Generate icon (ICNS)
# ─────────────────────────────────────────────────────────────
log "Generating app icon..."

mkdir -p "$ICONSET_DIR"

# Try rsvg-convert first (librsvg, highest quality)
if command -v rsvg-convert &>/dev/null; then
    log "Using rsvg-convert for SVG→PNG conversion"
    for size in 16 32 64 128 256 512 1024; do
        rsvg-convert -w "$size" -h "$size" "$SVG_SOURCE" -o "$ICONSET_DIR/icon_${size}x${size}.png"
    done
elif command -v qlmanage &>/dev/null; then
    log "Using qlmanage for SVG→PNG conversion (install librsvg for better quality)"
    # qlmanage renders at a fixed size and names files differently
    TMPDIR_ICONS=$(mktemp -d)
    qlmanage -t -s 1024 -o "$TMPDIR_ICONS" "$SVG_SOURCE" 2>/dev/null || true
    # Find the rendered file
    RENDERED=$(find "$TMPDIR_ICONS" -name "*.png" | head -1)
    if [[ -n "$RENDERED" ]]; then
        for size in 16 32 64 128 256 512 1024; do
            sips -z "$size" "$size" "$RENDERED" --out "$ICONSET_DIR/icon_${size}x${size}.png" &>/dev/null
        done
        rm -rf "$TMPDIR_ICONS"
    else
        warn "qlmanage failed to render SVG. Creating placeholder icon."
        # Create a simple placeholder using sips with a colored square
        python3 -c "
import struct, zlib

def create_png(size, r, g, b):
    def chunk(name, data):
        c = zlib.crc32(name + data) & 0xffffffff
        return struct.pack('>I', len(data)) + name + data + struct.pack('>I', c)
    ihdr = struct.pack('>IIBBBBB', size, size, 8, 2, 0, 0, 0)
    row = b'\\x00' + bytes([r, g, b] * size)
    raw = row * size
    idat = zlib.compress(raw)
    data = b'\\x89PNG\\r\\n\\x1a\\n'
    data += chunk(b'IHDR', ihdr)
    data += chunk(b'IDAT', idat)
    data += chunk(b'IEND', b'')
    return data

import os
iconset = '$ICONSET_DIR'
os.makedirs(iconset, exist_ok=True)
for size in [16, 32, 64, 128, 256, 512, 1024]:
    png = create_png(size, 20, 20, 22)
    with open(f'{iconset}/icon_{size}x{size}.png', 'wb') as f:
        f.write(png)
print('Placeholder icons created.')
"
    fi
else
    warn "Neither rsvg-convert nor qlmanage found. Creating minimal placeholder icons."
    python3 -c "
import struct, zlib, os

def create_png(size, r, g, b):
    def chunk(name, data):
        c = zlib.crc32(name + data) & 0xffffffff
        return struct.pack('>I', len(data)) + name + data + struct.pack('>I', c)
    ihdr = struct.pack('>IIBBBBB', size, size, 8, 2, 0, 0, 0)
    row = b'\\x00' + bytes([r, g, b] * size)
    raw = row * size
    idat = zlib.compress(raw)
    data = b'\\x89PNG\\r\\n\\x1a\\n'
    data += chunk(b'IHDR', ihdr)
    data += chunk(b'IDAT', idat)
    data += chunk(b'IEND', b'')
    return data

iconset = '$ICONSET_DIR'
os.makedirs(iconset, exist_ok=True)
for size in [16, 32, 64, 128, 256, 512, 1024]:
    png = create_png(size, 20, 20, 22)
    with open(f'{iconset}/icon_{size}x{size}.png', 'wb') as f:
        f.write(png)
print('Placeholder icons created.')
"
fi

# Rename to iconset convention (macOS expects @2x variants too)
cp "$ICONSET_DIR/icon_32x32.png"   "$ICONSET_DIR/icon_16x16@2x.png"  2>/dev/null || true
cp "$ICONSET_DIR/icon_64x64.png"   "$ICONSET_DIR/icon_32x32@2x.png"  2>/dev/null || true
cp "$ICONSET_DIR/icon_256x256.png" "$ICONSET_DIR/icon_128x128@2x.png" 2>/dev/null || true
cp "$ICONSET_DIR/icon_512x512.png" "$ICONSET_DIR/icon_256x256@2x.png" 2>/dev/null || true
cp "$ICONSET_DIR/icon_1024x1024.png" "$ICONSET_DIR/icon_512x512@2x.png" 2>/dev/null || true

# Remove the 1024 plain (not a standard iconset size)
rm -f "$ICONSET_DIR/icon_1024x1024.png"

# Assemble ICNS
iconutil -c icns "$ICONSET_DIR" -o "$ICNS_OUT" 2>/dev/null || warn "iconutil failed — app will use default icon"
[[ -f "$ICNS_OUT" ]] && ok "Icon assembled: $ICNS_OUT"

# ─────────────────────────────────────────────────────────────
# 4. Assemble app bundle
# ─────────────────────────────────────────────────────────────
log "Assembling app bundle..."

rm -rf "$APP_BUNDLE"
mkdir -p "$APP_BUNDLE/Contents/MacOS"
mkdir -p "$APP_BUNDLE/Contents/Resources"

# Copy binary
cp "$BINARY_PATH" "$APP_BUNDLE/Contents/MacOS/$APP_NAME"
chmod +x "$APP_BUNDLE/Contents/MacOS/$APP_NAME"

# Copy Info.plist
cp "$PROJECT_ROOT/Resources/Info.plist" "$APP_BUNDLE/Contents/Info.plist"

# Copy icon
if [[ -f "$ICNS_OUT" ]]; then
    cp "$ICNS_OUT" "$APP_BUNDLE/Contents/Resources/AppIcon.icns"
fi

ok "App bundle assembled: $APP_BUNDLE"

# ─────────────────────────────────────────────────────────────
# 5. Ad-hoc code sign
# ─────────────────────────────────────────────────────────────
log "Code signing (ad-hoc)..."
codesign --force --deep --sign - "$APP_BUNDLE" 2>&1 || warn "Code signing failed (app may still run with security exception)"
ok "Signed: $APP_BUNDLE"

# ─────────────────────────────────────────────────────────────
# 6. Create DMG
# ─────────────────────────────────────────────────────────────
log "Creating DMG..."

rm -f "$DMG_OUT"

# Create a temporary folder for DMG contents
DMG_STAGING=$(mktemp -d)
cp -R "$APP_BUNDLE" "$DMG_STAGING/"
ln -s /Applications "$DMG_STAGING/Applications"

# Create compressed DMG
hdiutil create \
    -volname "$DMG_NAME" \
    -srcfolder "$DMG_STAGING" \
    -ov \
    -format UDZO \
    -imagekey zlib-level=9 \
    "$DMG_OUT" 2>&1 || die "hdiutil failed to create DMG"

rm -rf "$DMG_STAGING"

ok "DMG created: $DMG_OUT"

# ─────────────────────────────────────────────────────────────
# Summary
# ─────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BOLD}  MCP Killer built successfully!${NC}"
echo -e "${BOLD}${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "  App:  ${CYAN}$APP_BUNDLE${NC}"
echo -e "  DMG:  ${CYAN}$DMG_OUT${NC}"
echo ""
echo -e "  Install: open \"$DMG_OUT\""
echo ""
