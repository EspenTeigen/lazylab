#!/bin/bash
set -e

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
REPO="EspenTeigen/lazylab"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$ARCH" in
        x86_64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac

    case "$OS" in
        linux) OS="linux" ;;
        darwin) OS="darwin" ;;
        *) error "Unsupported OS: $OS" ;;
    esac

    echo "${OS}_${ARCH}"
}

# Get latest release version from GitHub
get_latest_version() {
    curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install binary
install_binary() {
    PLATFORM=$(detect_platform)
    VERSION=$(get_latest_version)

    if [ -z "$VERSION" ]; then
        error "Could not determine latest version. Check https://github.com/${REPO}/releases"
    fi

    info "Installing lazylab ${VERSION} for ${PLATFORM}..."

    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/lazylab_${PLATFORM}.tar.gz"
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    info "Downloading from ${DOWNLOAD_URL}..."
    curl -sL "$DOWNLOAD_URL" -o "$TMP_DIR/lazylab.tar.gz" || error "Download failed"

    info "Extracting..."
    tar -xzf "$TMP_DIR/lazylab.tar.gz" -C "$TMP_DIR" || error "Extraction failed"

    info "Installing to ${INSTALL_DIR}..."
    mkdir -p "$INSTALL_DIR"
    mv "$TMP_DIR/lazylab" "$INSTALL_DIR/lazylab"
    chmod +x "$INSTALL_DIR/lazylab"

    info "Successfully installed lazylab to ${INSTALL_DIR}/lazylab"

    # Check if INSTALL_DIR is in PATH
    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        warn "${INSTALL_DIR} is not in your PATH"
        echo ""
        echo "Add this to your shell config (~/.bashrc, ~/.zshrc, etc.):"
        echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
        echo ""
    fi

    echo ""
    info "Run 'lazylab' to start!"
}

# Build from source (fallback)
install_from_source() {
    info "Building from source..."

    if ! command -v go &> /dev/null; then
        error "Go is required to build from source. Install from https://go.dev"
    fi

    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    info "Cloning repository..."
    git clone --depth 1 "https://github.com/${REPO}.git" "$TMP_DIR/lazylab" || error "Clone failed"

    info "Building..."
    cd "$TMP_DIR/lazylab"
    go build -o lazylab ./cmd/lazylab || error "Build failed"

    info "Installing to ${INSTALL_DIR}..."
    mkdir -p "$INSTALL_DIR"
    mv lazylab "$INSTALL_DIR/lazylab"
    chmod +x "$INSTALL_DIR/lazylab"

    info "Successfully installed lazylab to ${INSTALL_DIR}/lazylab"
}

# Main
main() {
    echo "╭───────────────────────────────────────╮"
    echo "│          LazyLab Installer            │"
    echo "╰───────────────────────────────────────╯"
    echo ""

    # Try binary install first, fall back to source
    if command -v curl &> /dev/null; then
        install_binary || install_from_source
    else
        warn "curl not found, building from source..."
        install_from_source
    fi
}

main "$@"
