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

DEPS=$(ldd "$BINARY" | grep "=> /" | awk '{print $3}')

for dep in $DEPS; do
    libname=$(basename "$dep")
    if is_excluded "$libname"; then
        echo "Skipping system library: $libname"
    else
        echo "Bundling: $libname"
        cp -L "$dep" "$LIB_DIR/"
    fi
done

echo "Setting RPATH to \$ORIGIN/lib"
patchelf --set-rpath '$ORIGIN/lib' "$BINARY"

echo "Bundling complete."
