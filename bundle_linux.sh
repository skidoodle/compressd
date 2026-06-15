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

SYSTEM_AVIF=$(ldconfig -p | grep libavif.so | head -n 1 | awk -F '=> ' '{print $2}')
if [ -n "$SYSTEM_AVIF" ] && [ -f "$SYSTEM_AVIF" ]; then
    echo "Force-queuing system libavif: $SYSTEM_AVIF"
    echo "$SYSTEM_AVIF" >> "$QUEUE_FILE"
fi

VIPS_VERSION=$(pkg-config vips --modversion | cut -d. -f1,2)
MOD_SUBDIR="vips-modules-$VIPS_VERSION"

VIPS_MODULE_DIR=$(pkg-config vips --variable=pluginsdir 2>/dev/null || echo "")
echo "pkg-config vips pluginsdir: $VIPS_MODULE_DIR"
if [ -z "$VIPS_MODULE_DIR" ] || [ ! -d "$VIPS_MODULE_DIR" ]; then
    echo "Checking fallback dirs for vips $VIPS_VERSION..."
    for dir in "/usr/lib/$(uname -m)-linux-gnu/vips-plugins-$VIPS_VERSION" "/usr/lib/vips-plugins-$VIPS_VERSION" "/usr/lib/$(uname -m)-linux-gnu/$MOD_SUBDIR" "/usr/lib/$MOD_SUBDIR"; do
        echo "Checking $dir"
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
    mkdir -p "$LIB_DIR/$MOD_SUBDIR"
    cp "$VIPS_MODULE_DIR"/*.so "$LIB_DIR/$MOD_SUBDIR/" 2>/dev/null || true
    for module in "$LIB_DIR/$MOD_SUBDIR"/*.so; do
        if [ -f "$module" ]; then
            echo "Bundling module: $(basename "$module")"
            echo "$(realpath "$module")" >> "$QUEUE_FILE"
            patchelf --set-rpath '$ORIGIN/..' --force-rpath "$module"
        fi
    done
else
    echo "Warning: libvips plugins directory not found. AVIF/HEIF support might be missing if it's a module."
fi

HEIF_PLUGIN_DIR=""
for d in \
    "/usr/lib/$(uname -m)-linux-gnu/libheif/plugins" \
    "/usr/lib/$(uname -m)-linux-gnu/libheif" \
    "/usr/lib/libheif/plugins" \
    "/usr/lib/libheif"; do
    if [ -d "$d" ] && (ls "$d"/libheif-*.so* "$d"/libheif_*.so* >/dev/null 2>&1); then
        HEIF_PLUGIN_DIR="$d"
        break
    fi
done

if [ -z "$HEIF_PLUGIN_DIR" ]; then
    HEIF_PLUGIN_DIR=$(find /usr/lib \( -name "libheif-*.so*" -o -name "libheif_*.so*" \) -print -quit 2>/dev/null | xargs dirname 2>/dev/null || echo "")
fi

if [ -n "$HEIF_PLUGIN_DIR" ] && [ -d "$HEIF_PLUGIN_DIR" ]; then
    echo "Found libheif plugins in $HEIF_PLUGIN_DIR"
    mkdir -p "$LIB_DIR/libheif"
    cp -v "$HEIF_PLUGIN_DIR"/libheif-*.so* "$HEIF_PLUGIN_DIR"/libheif_*.so* "$LIB_DIR/libheif/" 2>/dev/null || true
    for plugin in "$LIB_DIR/libheif"/*.so*; do
        if [ -f "$plugin" ] && [ ! -L "$plugin" ]; then
            echo "Bundling heif plugin: $(basename "$plugin")"
            echo "$(realpath "$plugin")" >> "$QUEUE_FILE"
            patchelf --set-rpath '$ORIGIN/..' --force-rpath "$plugin"
        fi
    done
else
    echo "Note: libheif plugins directory not found. This is normal if libheif is built without plugins."
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

ln -sf . "$LIB_DIR/x86_64-linux-gnu" 2>/dev/null || true
ln -sf . "$LIB_DIR/aarch64-linux-gnu" 2>/dev/null || true
ln -sf . "$LIB_DIR/lib64" 2>/dev/null || true

echo "Bundling complete."
