#!/bin/sh
# Markdown in the Middle - Install Script
# Downloads the latest release binary for your OS and architecture.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/rickcrawford/markdowninthemiddle/main/install.sh | sh
#
# Environment variables:
#   INSTALL_DIR  - Directory to install to (default: /usr/local/bin on Unix, %USERPROFILE%\bin on Windows)
#   VERSION      - Specific version to install (default: latest)
#   GITHUB_REPO  - Repository (default: rickcrawford/markdowninthemiddle)

set -e

GITHUB_REPO="${GITHUB_REPO:-rickcrawford/markdowninthemiddle}"
BINARY_NAME="markdowninthemiddle"

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "linux" ;;
        Darwin*)    echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *)          echo "unsupported" ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64" ;;
        arm64|aarch64)  echo "arm64" ;;
        armv7l)         echo "arm" ;;
        *)              echo "unsupported" ;;
    esac
}

# Get latest version from GitHub API
get_latest_version() {
    if command -v curl > /dev/null 2>&1; then
        curl -fsSL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | \
            grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget > /dev/null 2>&1; then
        wget -qO- "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | \
            grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        echo "Error: curl or wget is required" >&2
        exit 1
    fi
}

# Download a file
download() {
    url="$1"
    output="$2"
    if command -v curl > /dev/null 2>&1; then
        curl -fsSL -o "$output" "$url"
    elif command -v wget > /dev/null 2>&1; then
        wget -qO "$output" "$url"
    else
        echo "Error: curl or wget is required" >&2
        exit 1
    fi
}

main() {
    OS=$(detect_os)
    ARCH=$(detect_arch)

    if [ "$OS" = "unsupported" ] || [ "$ARCH" = "unsupported" ]; then
        echo "Error: Unsupported platform: $(uname -s) $(uname -m)" >&2
        exit 1
    fi

    VERSION="${VERSION:-$(get_latest_version)}"
    if [ -z "$VERSION" ]; then
        echo "Error: Could not determine latest version" >&2
        exit 1
    fi

    # Strip leading 'v' for filename if present
    VERSION_NUM="${VERSION#v}"

    EXT="tar.gz"
    if [ "$OS" = "windows" ]; then
        EXT="zip"
        BINARY_NAME="${BINARY_NAME}.exe"
    fi

    FILENAME="${BINARY_NAME}_${VERSION_NUM}_${OS}_${ARCH}.${EXT}"
    DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${FILENAME}"

    # Determine install directory
    if [ -n "$INSTALL_DIR" ]; then
        TARGET_DIR="$INSTALL_DIR"
    elif [ "$OS" = "windows" ]; then
        TARGET_DIR="${USERPROFILE}/bin"
    else
        TARGET_DIR="/usr/local/bin"
    fi

    TMPDIR=$(mktemp -d)
    trap 'rm -rf "$TMPDIR"' EXIT

    echo "Downloading ${BINARY_NAME} ${VERSION} for ${OS}/${ARCH}..."
    download "$DOWNLOAD_URL" "${TMPDIR}/${FILENAME}"

    echo "Extracting..."
    if [ "$EXT" = "zip" ]; then
        if command -v unzip > /dev/null 2>&1; then
            unzip -q "${TMPDIR}/${FILENAME}" -d "$TMPDIR"
        else
            echo "Error: unzip is required to extract .zip files" >&2
            exit 1
        fi
    else
        tar -xzf "${TMPDIR}/${FILENAME}" -C "$TMPDIR"
    fi

    echo "Installing to ${TARGET_DIR}/${BINARY_NAME}..."
    mkdir -p "$TARGET_DIR"

    if [ -w "$TARGET_DIR" ]; then
        cp "${TMPDIR}/${BINARY_NAME}" "${TARGET_DIR}/${BINARY_NAME}"
        chmod +x "${TARGET_DIR}/${BINARY_NAME}"
    else
        echo "Elevated permissions required to install to ${TARGET_DIR}"
        sudo cp "${TMPDIR}/${BINARY_NAME}" "${TARGET_DIR}/${BINARY_NAME}"
        sudo chmod +x "${TARGET_DIR}/${BINARY_NAME}"
    fi

    echo ""
    echo "${BINARY_NAME} ${VERSION} installed to ${TARGET_DIR}/${BINARY_NAME}"
    echo ""
    echo "Run '${BINARY_NAME} --help' to get started."
}

main
