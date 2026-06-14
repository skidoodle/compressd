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
    "libnsl.so"
    "libcrypt.so"
    "libglib-2.0.so.0"
    "libdbus-1.so.3"
    "libasound.so.2"
    "libdrm.so.2"
    "libGL.so.1"
    "libX11.so.6"
    "libxcb.so.1"
    "libselinux.so.1"
    "libmount.so.1"
    "libblkid.so.1"
    "libacl.so.1"
    "libcap.so.2"
    "libattr.so.1"
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

echo "Finding dependencies for $BINARY..."
ALL_DEPS=$(ldd "$BINARY" | grep "=> /" | awk '{print $3}')

for dep in $ALL_DEPS; do
    libname=$(basename "$dep")
    if is_excluded "$libname"; then
        echo "Skipping system library: $libname"
    else
        if [ ! -f "$LIB_DIR/$libname" ]; then
            echo "Bundling: $libname"
            cp -L "$dep" "$LIB_DIR/"
            patchelf --set-rpath '$ORIGIN' --force-rpath "$LIB_DIR/$libname"
        fi
    fi
done

echo "Setting RPATH on binary to \$ORIGIN/lib"
patchelf --set-rpath '$ORIGIN/lib' --force-rpath "$BINARY"

echo "Bundling complete."
