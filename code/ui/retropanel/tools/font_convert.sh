#!/usr/bin/env bash
#
# Convert TTF fonts to LVGL C arrays using lv_font_conv.
# Install: npm install -g lv_font_conv
#
# Fonts used:
#   - "Nixie One" — for numeric displays (48px)
#   - "Share Tech Mono" — for body text (18px)
#
# Download fonts from Google Fonts to assets/fonts/ first.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
ASSETS="$PROJECT_DIR/assets/fonts"
OUTPUT="$PROJECT_DIR/src/theme"

# Nixie display font — digits + degree + percent + dot
if [ -f "$ASSETS/NixieOne-Regular.ttf" ]; then
    echo "Converting Nixie One 48px..."
    lv_font_conv \
        --font "$ASSETS/NixieOne-Regular.ttf" \
        --bpp 4 \
        --size 48 \
        --range 0x20-0x7F \
        --format lvgl \
        --output "$OUTPUT/nixie_font.c" \
        --lv-include "lvgl.h"
    echo "  → $OUTPUT/nixie_font.c"
else
    echo "SKIP: $ASSETS/NixieOne-Regular.ttf not found"
    echo "  Download from: https://fonts.google.com/specimen/Nixie+One"
fi

# Mono body font — full ASCII
if [ -f "$ASSETS/ShareTechMono-Regular.ttf" ]; then
    echo "Converting Share Tech Mono 18px..."
    lv_font_conv \
        --font "$ASSETS/ShareTechMono-Regular.ttf" \
        --bpp 4 \
        --size 18 \
        --range 0x20-0x7F,0xB0 \
        --format lvgl \
        --output "$OUTPUT/mono_font.c" \
        --lv-include "lvgl.h"
    echo "  → $OUTPUT/mono_font.c"
else
    echo "SKIP: $ASSETS/ShareTechMono-Regular.ttf not found"
    echo "  Download from: https://fonts.google.com/specimen/Share+Tech+Mono"
fi

echo "Font conversion complete."
