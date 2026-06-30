#!/usr/bin/env bash
set -euo pipefail

APP="webcli"
REPO="yenya/webcli"
BINARY_DIR="${BINARY_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info()  { echo -e "${BLUE}info${NC} $1"; }
ok()    { echo -e "${GREEN}ok${NC}   $1"; }
error() { echo -e "${RED}error${NC} $1"; }

# Detect OS and architecture
detect_platform() {
    local os
    local arch

    case "$(uname -s)" in
        Linux)  os="linux"    ;;
        Darwin) os="darwin"   ;;
        MINGW*|MSYS*|CYGWIN*) os="windows" ;;
        *)      error "unsupported OS: $(uname -s)"; exit 1 ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *)      error "unsupported architecture: $(uname -m)"; exit 1 ;;
    esac

    echo "${os}_${arch}"
}

# Find latest version from GitHub API
get_latest_version() {
    if [ "$VERSION" != "latest" ]; then
        echo "$VERSION"
        return
    fi

    local url="https://api.github.com/repos/${REPO}/releases/latest"
    if command -v curl &>/dev/null; then
        curl -sL "$url" | grep '"tag_name"' | cut -d'"' -f4
    elif command -v wget &>/dev/null; then
        wget -qO- "$url" | grep '"tag_name"' | cut -d'"' -f4
    else
        error "need curl or wget"
        exit 1
    fi
}

# Determine install directory
determine_install_dir() {
    if [ -w "$BINARY_DIR" ]; then
        echo "$BINARY_DIR"
        return
    fi

    # Try common writable bin dirs
    for dir in "$HOME/.local/bin" "$HOME/bin" "$HOME/.cargo/bin"; do
        if [ -d "$dir" ] && [ -w "$dir" ]; then
            echo "$dir"
            return
        fi
    done

    # Create ~/.local/bin if it doesn't exist
    local fallback="$HOME/.local/bin"
    mkdir -p "$fallback"
    echo "$fallback"
}

install_via_go() {
    info "Installing via 'go install'..."
    if ! command -v go &>/dev/null; then
        error "go is not installed. Install Go from https://go.dev/dl/"
        exit 1
    fi

    local version_flag
    if [ "$VERSION" = "latest" ]; then
        version_flag="@latest"
    else
        version_flag="@${VERSION}"
    fi

    go install "github.com/${REPO}${version_flag}"
    ok "Installed! Make sure \$GOPATH/bin is in your PATH"
    echo ""
    echo "  $(command -v webcli || echo "\$GOPATH/bin/webcli")"
}

install_via_binary() {
    local platform
    platform="$(detect_platform)"
    local dest_dir
    dest_dir="$(determine_install_dir)"
    local binary_path="${dest_dir}/${APP}"

    info "Detected: ${platform}"
    info "Target:   ${binary_path}"

    local tag
    tag="$(get_latest_version)"
    info "Version:  ${tag}"

    local url="https://github.com/${REPO}/releases/download/${tag}/${APP}_${platform}.tar.gz"

    info "Downloading ${url}..."

    local tmpdir
    tmpdir="$(mktemp -d)"
    cd "$tmpdir"

    if command -v curl &>/dev/null; then
        curl -sL "$url" -o "${APP}.tar.gz"
    elif command -v wget &>/dev/null; then
        wget -q "$url" -O "${APP}.tar.gz"
    else
        error "need curl or wget"
        exit 1
    fi

    tar xzf "${APP}.tar.gz"
    chmod +x "${APP}"

    mv "${APP}" "${binary_path}"
    cd /
    rm -rf "$tmpdir"

    ok "Installed to ${binary_path}"
    echo ""
    echo "  ${binary_path}"
}

build_from_source() {
    info "Building from source..."

    if ! command -v go &>/dev/null; then
        error "go is not installed. Install Go from https://go.dev/dl/"
        exit 1
    fi

    local tmpdir
    tmpdir="$(mktemp -d)"
    cd "$tmpdir"

    git clone "https://github.com/${REPO}.git" .
    go build -ldflags="-s -w" -o "${APP}" .

    local dest_dir
    dest_dir="$(determine_install_dir)"
    local binary_path="${dest_dir}/${APP}"
    mv "${APP}" "${binary_path}"

    cd /
    rm -rf "$tmpdir"

    ok "Installed to ${binary_path}"
    echo ""
    echo "  ${binary_path}"
}

# --- main ---

echo ""
echo "  ${APP} — Web access & shared memory for AI agents"
echo "  ================================================="
echo ""

# Check if binary already exists
if command -v ${APP} &>/dev/null; then
    info "${APP} is already installed at $(command -v ${APP})"
    echo ""
    echo -n "  Reinstall? [y/N] "
    read -r REPLY
    if [ "$REPLY" != "y" ] && [ "$REPLY" != "Y" ]; then
        echo "  Skipping."
        exit 0
    fi
fi

# Try binary install first, fallback to go install, then source
if curl -sI "https://github.com/${REPO}/releases" 2>/dev/null | grep -q "200\|302"; then
    install_via_binary
else
    info "GitHub releases not available, trying go install..."
    if command -v go &>/dev/null; then
        install_via_go
    else
        build_from_source
    fi
fi

echo ""
info "Run '${APP} --help' to get started"
info "Run '${APP} config init' to create config"
info "Run '${APP} serve --transport sse' to start MCP server"
echo ""
