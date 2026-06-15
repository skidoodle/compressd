#!/bin/bash
set -e

BINARY=$1
LIB_DIR="lib"

mkdir -p "$LIB_DIR"

EXCLUDE_LIST=(
    "linux-vdso.so"
    "libpthread.so"
    "libdl.so"
    "libc.so"
    "libm.so"
    "librt.so"
    "ld-linux-x86-64.so"
    "libgcc_s.so"
    "libstdc++.so"
    "libresolv.so"
    "libutil.so"
)

function is_excluded() {
    local lib=$1
    for exclude in "${EXCLUDE_LIST[@]}"; do
        if [[ $lib == *"$exclude"* ]]; then
            return 0
        fi
    done
    return 1
}

PROCESSED_FILE=$(mktemp)
QUEUE_FILE=$(mktemp)

echo "$BINARY" > "$QUEUE_FILE"

SYSTEM_HEIF=$(ldconfig -p | grep libheif.so | head -n 1 | awk -F '=> ' '{print $2}')
if [ -n "$SYSTEM_HEIF" ] && [ -f "$SYSTEM_HEIF" ]; then
    echo "Force-queuing system libheif: $SYSTEM_HEIF"
    echo "$SYSTEM_HEIF" >> "$QUEUE_FILE"
else
    FORCED_HEIF="/usr/lib/$(uname -m)-linux-gnu/libheif.so"
    if [ -f "$FORCED_HEIF" ]; then
        echo "Force-queuing fallback libheif: $FORCED_HEIF"
        echo "$FORCED_HEIF" >> "$QUEUE_FILE"
    fi
fi

VIPS_MODULE_DIR=$(pkg-config vips --variable=pluginsdir 2>/dev/null || echo "")
if [ -z "$VIPS_MODULE_DIR" ]; then
    VIPS_VERSION=$(pkg-config vips --modversion | cut -d. -f1,2)
    for dir in "/usr/lib/$(uname -m)-linux-gnu/vips-plugins-$VIPS_VERSION" "/usr/lib/vips-plugins-$VIPS_VERSION"; do
        if [ -d "$dir" ]; then
            VIPS_MODULE_DIR="$dir"
            break
        fi
    done
fi

if [ -z "$VIPS_MODULE_DIR" ] || [ ! -d "$VIPS_MODULE_DIR" ]; then
    echo "Searching /usr for vips plugins..."
    VIPS_MODULE_DIR=$(find /usr/lib -name "vips-*.so" -print -quit | xargs dirname 2>/dev/null || echo "")
fi

if [ -n "$VIPS_MODULE_DIR" ] && [ -d "$VIPS_MODULE_DIR" ]; then
    echo "Found libvips modules in $VIPS_MODULE_DIR"
    mkdir -p "$LIB_DIR/vips-modules"
    cp "$VIPS_MODULE_DIR"/*.so "$LIB_DIR/vips-modules/" 2>/dev/null || true
    for module in "$LIB_DIR/vips-modules"/*.so; do
        if [ -f "$module" ]; then
            echo "Bundling module: $(basename "$module")"
            echo "$(realpath "$module")" >> "$QUEUE_FILE"
            patchelf --set-rpath '$ORIGIN/..' --force-rpath "$module"
        fi
    done
else
    echo "Warning: libvips plugins directory not found. AVIF/HEIF support might be missing if it's a module."
fi

while [ -s "$QUEUE_FILE" ]; do
    CURRENT=$(head -n 1 "$QUEUE_FILE")
    sed -i '1d' "$QUEUE_FILE"

    if grep -qF "$CURRENT" "$PROCESSED_FILE" 2>/dev/null; then
        continue
    fi
    echo "$CURRENT" >> "$PROCESSED_FILE"
    echo "Processing: $CURRENT"

    if [ ! -f "$CURRENT" ]; then
        DEPS=$(ldd "$BINARY" | grep "=> /" | grep "$(basename "$CURRENT")" | awk '{print $3}')
        for dep in $DEPS; do
            echo "Queuing dependency of $BINARY: $dep"
            echo "$dep" >> "$QUEUE_FILE"
        done
        continue
    fi

    echo "Finding dependencies for $CURRENT..."
    DEPS=$(ldd "$CURRENT" | grep "=> /" | awk '{print $3}')

    for dep in $DEPS; do
        libname=$(basename "$dep")
        if is_excluded "$libname"; then
            echo "Skipping excluded: $libname"
            continue
        fi

        if [ ! -f "$LIB_DIR/$libname" ]; then
            echo "Bundling and queuing: $libname (from $dep)"
            cp -L "$dep" "$LIB_DIR/"
            patchelf --set-rpath '$ORIGIN' --force-rpath "$LIB_DIR/$libname"
            echo "$(realpath "$LIB_DIR/$libname")" >> "$QUEUE_FILE"
        fi
    done

done

rm "$PROCESSED_FILE" "$QUEUE_FILE"

echo "Setting RPATH on binary to \$ORIGIN/lib"
patchelf --set-rpath '$ORIGIN/lib' --force-rpath "$BINARY"

echo "Bundling complete."
