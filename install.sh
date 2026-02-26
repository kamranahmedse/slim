#!/bin/sh
set -e

REPO="kamranahmedse/slim"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  darwin) OS="darwin" ;;
  linux)  OS="linux" ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# Get latest version
TAG=$(curl -fsI "https://github.com/$REPO/releases/latest" | grep -i "^location:" | sed 's/.*tag\///' | tr -d '\r\n')
if [ -z "$TAG" ]; then
  echo "Failed to fetch latest version"
  exit 1
fi

VERSION="${TAG#v}"
FILENAME="slim_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/${TAG}/${FILENAME}"
CHECKSUM_URL="https://github.com/$REPO/releases/download/${TAG}/checksums.txt"

echo "Installing slim ${VERSION} (${OS}/${ARCH})..."

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -fL --progress-bar "$URL" -o "$TMP/$FILENAME"
curl -fsSL "$CHECKSUM_URL" -o "$TMP/checksums.txt"

# Verify checksum
if [ "$OS" = "darwin" ]; then
  (cd "$TMP" && grep "$FILENAME" checksums.txt | shasum -a 256 -c --quiet)
elif command -v sha256sum >/dev/null 2>&1; then
  (cd "$TMP" && grep "$FILENAME" checksums.txt | sha256sum -c --quiet)
else
  echo "Warning: cannot verify checksum (sha256sum/shasum not found)"
fi

tar -xzf "$TMP/$FILENAME" -C "$TMP"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP/slim" "$INSTALL_DIR/slim"
else
  sudo mv "$TMP/slim" "$INSTALL_DIR/slim"
fi

chmod +x "$INSTALL_DIR/slim"

echo "Installed slim to $INSTALL_DIR/slim"
slim version
