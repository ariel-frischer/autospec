#!/bin/sh
# shellcheck disable=SC3043  # 'local' is widely supported in practice (dash, ash, busybox)
# AutoSpec Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/ariel-frischer/autospec/main/install.sh | sh
#
# Environment variables:
#   AUTOSPEC_INSTALL_DIR - Installation directory (default: /usr/local/bin)
#   AUTOSPEC_VERSION     - Specific version to install (default: latest)

set -e

# Configuration
GITHUB_REPO="ariel-frischer/autospec"
BINARY_NAME="autospec"
DEFAULT_INSTALL_DIR="/usr/local/bin"

# Colors (disabled if not a terminal)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# Logging functions
info() {
    printf "${BLUE}==>${NC} %s\n" "$1"
}

success() {
    printf "${GREEN}==>${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}Warning:${NC} %s\n" "$1"
}

error() {
    printf "${RED}Error:${NC} %s\n" "$1" >&2
    exit 1
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "Linux" ;;
        Darwin*) echo "Darwin" ;;
        MINGW*|MSYS*|CYGWIN*) error "Windows is not supported by this installer. Please use install.ps1 or download manually." ;;
        *) error "Unsupported operating system: $(uname -s)" ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "x86_64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) error "Unsupported architecture: $(uname -m)" ;;
    esac
}

# Check for required commands
check_dependencies() {
    for cmd in curl tar; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            error "Required command not found: $cmd"
        fi
    done
}

# Get latest release version from GitHub
get_latest_version() {
    local latest_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    local version

    version=$(curl -fsSL "$latest_url" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')

    if [ -z "$version" ]; then
        error "Failed to fetch latest version from GitHub. Check your internet connection."
    fi

    echo "$version"
}

# Download and verify checksum
download_and_verify() {
    local version="$1"
    local os="$2"
    local arch="$3"
    local tmp_dir="$4"

    # Construct archive name (matches goreleaser template)
    local archive_name="autospec_${version#v}_${os}_${arch}.tar.gz"
    local download_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/${archive_name}"
    local checksum_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/checksums.txt"

    info "Downloading ${archive_name}..."

    if ! curl -fsSL -o "${tmp_dir}/${archive_name}" "$download_url"; then
        error "Failed to download ${archive_name}. Check if version ${version} exists."
    fi

    # Try to verify checksum
    if curl -fsSL -o "${tmp_dir}/checksums.txt" "$checksum_url" 2>/dev/null; then
        info "Verifying checksum..."
        local expected_checksum
        expected_checksum=$(grep "${archive_name}" "${tmp_dir}/checksums.txt" | awk '{print $1}')

        if [ -n "$expected_checksum" ]; then
            local actual_checksum
            if command -v sha256sum >/dev/null 2>&1; then
                actual_checksum=$(sha256sum "${tmp_dir}/${archive_name}" | awk '{print $1}')
            elif command -v shasum >/dev/null 2>&1; then
                actual_checksum=$(shasum -a 256 "${tmp_dir}/${archive_name}" | awk '{print $1}')
            else
                warn "sha256sum/shasum not found, skipping checksum verification"
                echo "${tmp_dir}/${archive_name}"
                return
            fi

            if [ "$expected_checksum" != "$actual_checksum" ]; then
                error "Checksum verification failed!\nExpected: ${expected_checksum}\nActual: ${actual_checksum}"
            fi
            success "Checksum verified"
        else
            warn "Checksum not found for ${archive_name}, skipping verification"
        fi
    else
        warn "Could not download checksums.txt, skipping verification"
    fi

    echo "${tmp_dir}/${archive_name}"
}

# Extract binary from archive
extract_binary() {
    local archive_path="$1"
    local tmp_dir="$2"

    info "Extracting archive..."
    tar -xzf "$archive_path" -C "$tmp_dir"

    if [ ! -f "${tmp_dir}/${BINARY_NAME}" ]; then
        error "Binary '${BINARY_NAME}' not found in archive"
    fi

    echo "${tmp_dir}/${BINARY_NAME}"
}

# Install binary to destination
install_binary() {
    local binary_path="$1"
    local install_dir="$2"

    # Create install directory if it doesn't exist
    if [ ! -d "$install_dir" ]; then
        info "Creating directory ${install_dir}..."
        if ! mkdir -p "$install_dir" 2>/dev/null; then
            warn "Cannot create ${install_dir} without elevated privileges"
            info "Trying with sudo..."
            sudo mkdir -p "$install_dir"
        fi
    fi

    # Check if we can write to install directory
    if [ -w "$install_dir" ]; then
        mv "$binary_path" "${install_dir}/${BINARY_NAME}"
        chmod +x "${install_dir}/${BINARY_NAME}"
    else
        info "Elevated privileges required to install to ${install_dir}"
        sudo mv "$binary_path" "${install_dir}/${BINARY_NAME}"
        sudo chmod +x "${install_dir}/${BINARY_NAME}"
    fi
}

# Check if binary is in PATH
check_path() {
    local install_dir="$1"

    case ":$PATH:" in
        *":${install_dir}:"*) return 0 ;;
        *) return 1 ;;
    esac
}

# Main installation function
main() {
    echo ""
    printf '%s\n' "${GREEN}"
    echo '▄▀█ █ █ ▀█▀ █▀█ █▀ █▀█ █▀▀ █▀▀'
    echo '█▀█ █▄█  █  █▄█ ▄█ █▀▀ ██▄ █▄▄'
    printf '%s\n' "${NC}"
    echo "         Installer"
    echo ""

    # Check dependencies
    check_dependencies

    # Detect platform
    local os
    local arch
    os=$(detect_os)
    arch=$(detect_arch)
    info "Detected platform: ${os}/${arch}"

    # Determine version to install
    local version="${AUTOSPEC_VERSION:-}"
    if [ -z "$version" ]; then
        info "Fetching latest version..."
        version=$(get_latest_version)
    fi
    info "Installing version: ${version}"

    # Determine install directory
    local install_dir="${AUTOSPEC_INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"
    info "Install directory: ${install_dir}"

    # Create temporary directory
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    # Download and verify
    local archive_path
    archive_path=$(download_and_verify "$version" "$os" "$arch" "$tmp_dir")

    # Extract
    local binary_path
    binary_path=$(extract_binary "$archive_path" "$tmp_dir")

    # Install
    info "Installing to ${install_dir}..."
    install_binary "$binary_path" "$install_dir"

    echo ""
    success "Successfully installed ${BINARY_NAME} ${version} to ${install_dir}/${BINARY_NAME}"
    echo ""

    # Check PATH
    if check_path "$install_dir"; then
        success "${install_dir} is already in your PATH"
    else
        warn "${install_dir} is NOT in your PATH"
        echo ""
        echo "Add it to your shell config:"
        echo ""
        echo "    # Bash (~/.bashrc or ~/.bash_profile)"
        echo "    export PATH=\"${install_dir}:\$PATH\""
        echo ""
        echo "    # Zsh (~/.zshrc)"
        echo "    export PATH=\"${install_dir}:\$PATH\""
        echo ""
        echo "    # Fish (~/.config/fish/config.fish)"
        echo "    fish_add_path ${install_dir}"
        echo ""
        echo "Then reload your shell:"
        echo "    source ~/.bashrc   # Bash"
        echo "    source ~/.zshrc    # Zsh"
        echo "    exec fish          # Fish"
        echo ""
        echo "Or start a new terminal session."
        echo ""
    fi

    # Verify installation
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        echo "Verify installation:"
        echo ""
        echo "    ${BINARY_NAME} version"
        echo "    ${BINARY_NAME} doctor"
        echo ""
    fi

    echo "Get started:"
    echo ""
    echo "    ${BINARY_NAME} init           # Initialize configuration"
    echo "    ${BINARY_NAME} --help         # Show available commands"
    echo ""
    echo "Documentation: https://github.com/${GITHUB_REPO}"
    echo ""
}

# Run main
main "$@"
