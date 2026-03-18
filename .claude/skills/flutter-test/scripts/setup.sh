#!/bin/bash
set -e

# Skip if already installed
if command -v fac &>/dev/null; then
  exit 0
fi

SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"

# Detect platform
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$ARCH" in
  aarch64|arm64) ARCH="arm64" ;;
  x86_64)        ARCH="amd64" ;;
esac

BINARY="${SKILL_DIR}/bin/fac-${OS}-${ARCH}"

if [ -f "$BINARY" ]; then
  # Use bundled binary
  cp "$BINARY" /usr/local/bin/fac 2>/dev/null || cp "$BINARY" /tmp/fac
  chmod +x /usr/local/bin/fac 2>/dev/null || (chmod +x /tmp/fac && export PATH="/tmp:$PATH")
  echo "FAC installed from skill bundle"
else
  # Fallback: download from GitHub
  URL="https://github.com/torbenkeller/flutter-agent-connect/releases/latest/download/fac-${OS}-${ARCH}"
  curl -fsSL "$URL" -o /usr/local/bin/fac 2>/dev/null || \
    (curl -fsSL "$URL" -o /tmp/fac && chmod +x /tmp/fac)
  chmod +x /usr/local/bin/fac 2>/dev/null || true
  echo "FAC installed from GitHub release"
fi
