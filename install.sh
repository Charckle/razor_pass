#!/usr/bin/env bash
set -euo pipefail

# Install rpass to a PATH directory.
# Default: ~/.local/bin  (user, no sudo)
# System:  PREFIX=/usr/local ./install.sh

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

PREFIX="${PREFIX:-$HOME/.local}"
BIN_DIR="${PREFIX}/bin"
TARGET="${BIN_DIR}/rpass"

if ! command -v go >/dev/null 2>&1; then
  echo "rpass: go is not installed. Install Go 1.25+ from https://go.dev/dl/" >&2
  exit 1
fi

echo "building rpass..."
go build -o "$ROOT/rpass" .

mkdir -p "$BIN_DIR"

if [[ -w "$BIN_DIR" ]] || [[ -w "$PREFIX" ]]; then
  install -m 755 "$ROOT/rpass" "$TARGET"
else
  echo "rpass: $BIN_DIR is not writable, trying sudo..."
  sudo install -m 755 "$ROOT/rpass" "$TARGET"
fi

echo "installed $TARGET"

case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *)
    echo
    echo "Note: $BIN_DIR is not on PATH. Add this to ~/.bashrc:"
    echo "  export PATH=\"$BIN_DIR:\$PATH\""
    echo "then open a new terminal, or run: hash -r"
    ;;
esac
