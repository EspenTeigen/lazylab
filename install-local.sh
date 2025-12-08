#!/bin/bash
set -e

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

main() {
    echo "╭───────────────────────────────────────╮"
    echo "│       LazyLab Local Installer         │"
    echo "╰───────────────────────────────────────╯"
    echo ""

    if ! command -v go &> /dev/null; then
        error "Go is required. Install from https://go.dev"
    fi

    info "Building lazylab..."
    go build -o lazylab ./cmd/lazylab || error "Build failed"

    info "Installing to ${INSTALL_DIR}..."
    mkdir -p "$INSTALL_DIR"
    mv lazylab "$INSTALL_DIR/lazylab"
    chmod +x "$INSTALL_DIR/lazylab"

    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        warn "${INSTALL_DIR} is not in your PATH"
        echo ""
        echo "Add this to your shell config:"
        echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
        echo ""
    fi

    info "Installed to ${INSTALL_DIR}/lazylab"
    echo ""
    info "Run 'lazylab' to start!"
}

main "$@"
