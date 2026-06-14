#!/bin/sh
set -e

REPO="skidoodle/compressd"
BINARY="compressd"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

if [ "$ARCH" != "amd64" ]; then
    echo "This script currently only supports amd64 binaries."
    exit 1
fi

if [ "$OS" = "darwin" ]; then
    echo "macOS binaries are not yet provided in prebuilt releases. Please use 'go install' or 'brew install vips && go build'."
    exit 1
fi

LATEST_TAG=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
    echo "Failed to fetch latest release tag."
    exit 1
fi

URL="https://github.com/$REPO/releases/download/$LATEST_TAG/compressd-linux-amd64.tar.gz"

echo "Downloading $BINARY $LATEST_TAG for $OS-$ARCH..."
curl -L "$URL" -o compressd.tar.gz
tar -xzf compressd.tar.gz

echo "Installing $BINARY to /usr/local/libexec/compressd/..."
sudo mkdir -p /usr/local/libexec/compressd
sudo cp $BINARY /usr/local/libexec/compressd/
sudo cp -r lib /usr/local/libexec/compressd/
sudo chmod +x /usr/local/libexec/compressd/$BINARY

echo "Creating symlink in /usr/local/bin/..."
sudo ln -sf /usr/local/libexec/compressd/$BINARY /usr/local/bin/$BINARY

echo "Successfully installed $BINARY to /usr/local/bin/"

rm -rf compressd.tar.gz lib/
