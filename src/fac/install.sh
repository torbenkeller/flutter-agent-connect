#!/bin/bash
set -e

REPO="torbenkeller/flutter-agent-connect"
VERSION="${VERSION:-latest}"

echo "Installing FAC (Flutter Agent Connect)..."

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  aarch64|arm64) BINARY="fac-linux-arm64" ;;
  x86_64)        BINARY="fac-linux-amd64" ;;
  *)             echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Download from GitHub Releases
if [ "$VERSION" = "latest" ]; then
  DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY}"
else
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}"
fi

echo "Downloading ${BINARY} from ${DOWNLOAD_URL}..."
curl -fsSL "$DOWNLOAD_URL" -o /usr/local/bin/fac
chmod +x /usr/local/bin/fac

echo "FAC installed to /usr/local/bin/fac"
fac --help | head -1
