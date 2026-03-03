#!/bin/bash

set -euo pipefail

REPO="rickcrawford/markdowninthemiddle"
INSTALL_DIR="${1:-.}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
if [ "$OS" = "darwin" ]; then
  OS="darwin"
elif [ "$OS" = "linux" ]; then
  OS="linux"
elif [ "$OS" = "mingw64_nt" ] || [ "$OS" = "msys_nt" ]; then
  OS="windows"
else
  echo "Unsupported OS: $OS" >&2
  exit 1
fi

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64 | amd64)
    ARCH="amd64"
    ;;
  aarch64 | arm64)
    ARCH="arm64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# Get latest release tag
echo "Fetching latest release from $REPO..."
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)
if [ -z "$LATEST" ]; then
  echo "Failed to fetch latest release" >&2
  exit 1
fi
echo "Latest version: $LATEST"

# Download and extract
if [ "$OS" = "windows" ]; then
  ARCHIVE="markdowninthemiddle-windows-amd64.zip"
  URL="https://github.com/$REPO/releases/download/$LATEST/$ARCHIVE"
  echo "Downloading $ARCHIVE from $URL..."
  curl -fsSL "$URL" -o "$ARCHIVE"
  unzip -o "$ARCHIVE" -d "$INSTALL_DIR"
  rm "$ARCHIVE"
  BINARY="$INSTALL_DIR/markdowninthemiddle/markdowninthemiddle.exe"
else
  ARCHIVE="markdowninthemiddle-$OS-$ARCH.tar.gz"
  URL="https://github.com/$REPO/releases/download/$LATEST/$ARCHIVE"
  echo "Downloading $ARCHIVE from $URL..."
  curl -fsSL "$URL" | tar -xz -C "$INSTALL_DIR" --strip-components=0
  BINARY="$INSTALL_DIR/markdowninthemiddle/markdowninthemiddle"
  chmod +x "$BINARY"
fi

echo "✓ Installed markdowninthemiddle to $BINARY"
echo "Run: $BINARY --help"
