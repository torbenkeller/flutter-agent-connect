#!/bin/bash
# Check if fac is installed, install if missing
if command -v fac &>/dev/null; then
  exit 0
fi

echo "FAC not found, installing..."
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$ARCH" in aarch64|arm64) ARCH="arm64" ;; x86_64) ARCH="amd64" ;; esac

curl -fsSL "https://github.com/torbenkeller/flutter-agent-connect/releases/latest/download/fac-${OS}-${ARCH}" \
  -o /usr/local/bin/fac && chmod +x /usr/local/bin/fac && echo "FAC installed" || \
  echo "Warning: Could not install FAC. Install manually or add the DevContainer feature."
