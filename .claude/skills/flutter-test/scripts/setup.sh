#!/bin/bash
set -e

# Install fac if not already installed
if command -v fac &>/dev/null; then
  exit 0
fi

echo "Installing FAC (Flutter Agent Connect)..."

ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

case "$ARCH" in
  aarch64|arm64) ARCH="arm64" ;;
  x86_64)        ARCH="amd64" ;;
esac

BINARY="fac-${OS}-${ARCH}"
URL="https://github.com/torbenkeller/flutter-agent-connect/releases/latest/download/${BINARY}"

curl -fsSL "$URL" -o /usr/local/bin/fac 2>/dev/null || \
  curl -fsSL "$URL" -o /tmp/fac && mv /tmp/fac /usr/local/bin/fac

chmod +x /usr/local/bin/fac
echo "FAC installed successfully"
