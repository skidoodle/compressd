#!/bin/sh
set -e

REPO="skidoodle/compressd"
BINARY="compressd"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo "Checking requirements..."
for cmd in curl tar sudo; do
    if ! command -v $cmd >/dev/null 2>&1; then
        echo "${RED}Error: $cmd is not installed.${NC}"
        exit 1
    fi
done

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "${RED}Unsupported architecture: $ARCH${NC}"; exit 1 ;;
esac

if [ "$ARCH" != "amd64" ]; then
    echo "${RED}This script currently only supports amd64 binaries.${NC}"
    exit 1
fi

if [ "$OS" != "linux" ]; then
    echo "${RED}This script is for Linux only.${NC}"
    exit 1
fi

echo "Fetching latest release information..."
LATEST_TAG=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
    echo "${RED}Failed to fetch latest release tag.${NC}"
    exit 1
fi

URL="https://github.com/$REPO/releases/download/$LATEST_TAG/compressd-linux-amd64.tar.gz"

TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

echo "Downloading $BINARY $LATEST_TAG for $OS-$ARCH..."
if ! curl -L "$URL" -o compressd.tar.gz; then
    echo "${RED}Failed to download release.${NC}"
    exit 1
fi

echo "Extracting..."
tar -xzf compressd.tar.gz

echo "Installing $BINARY to /usr/local/libexec/compressd/..."
sudo mkdir -p /usr/local/libexec/compressd
sudo cp $BINARY /usr/local/libexec/compressd/
if [ -d "lib" ]; then
    sudo cp -r lib /usr/local/libexec/compressd/
fi
sudo chmod +x /usr/local/libexec/compressd/$BINARY

echo "Creating symlink in /usr/local/bin/..."
sudo ln -sf /usr/local/libexec/compressd/$BINARY /usr/local/bin/$BINARY

echo "${GREEN}Successfully installed $BINARY to /usr/local/bin/${NC}"

cd - > /dev/null
rm -rf "$TMP_DIR"
