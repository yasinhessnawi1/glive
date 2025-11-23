#!/bin/bash
set -e

# GLive Installer for Linux and macOS
# Usage: curl -sSL https://raw.githubusercontent.com/yasinhessnawi1/glive/main/install.sh | bash

REPO="yasinhessnawi1/glive"
BINARY_NAME="glive"
INSTALL_DIR="/usr/local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Installing GLive...${NC}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo -e "${RED}Unsupported architecture: $ARCH${NC}"; exit 1 ;;
esac

case "$OS" in
    linux) OS="linux" ;;
    darwin) OS="darwin" ;;
    *) echo -e "${RED}Unsupported OS: $OS${NC}"; exit 1 ;;
esac

echo "Detected: $OS/$ARCH"

# Get latest release version
echo "Fetching latest release..."
LATEST_RELEASE=$(curl -sL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_RELEASE" ]; then
    echo -e "${YELLOW}No releases found. Building from source...${NC}"

    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Go is not installed. Please install Go from https://go.dev/dl/${NC}"
        exit 1
    fi

    # Clone and build
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"
    git clone --depth 1 "https://github.com/$REPO.git"
    cd glive/cmd/glive
    go build -o "$BINARY_NAME"

    # Install
    if [ -w "$INSTALL_DIR" ]; then
        mv "$BINARY_NAME" "$INSTALL_DIR/"
    else
        echo "Requesting sudo to install to $INSTALL_DIR..."
        sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
    fi

    # Cleanup
    rm -rf "$TMP_DIR"
else
    echo "Latest version: $LATEST_RELEASE"

    # Determine archive extension
    EXT="tar.gz"

    # Download URL
    VERSION_NO_V="${LATEST_RELEASE#v}"
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_RELEASE/glive_${VERSION_NO_V}_${OS}_${ARCH}.${EXT}"

    echo "Downloading from: $DOWNLOAD_URL"

    # Download and extract
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"

    if ! curl -sL "$DOWNLOAD_URL" -o "glive.${EXT}"; then
        echo -e "${RED}Failed to download release. Falling back to source build...${NC}"
        cd ..
        rm -rf "$TMP_DIR"

        TMP_DIR=$(mktemp -d)
        cd "$TMP_DIR"
        git clone --depth 1 "https://github.com/$REPO.git"
        cd glive/cmd/glive
        go build -o "$BINARY_NAME"

        if [ -w "$INSTALL_DIR" ]; then
            mv "$BINARY_NAME" "$INSTALL_DIR/"
        else
            sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
        fi
        rm -rf "$TMP_DIR"
    else
        tar -xzf "glive.${EXT}"

        # Install binary
        if [ -w "$INSTALL_DIR" ]; then
            mv "$BINARY_NAME" "$INSTALL_DIR/"
        else
            echo "Requesting sudo to install to $INSTALL_DIR..."
            sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
        fi

        # Cleanup
        cd ..
        rm -rf "$TMP_DIR"
    fi
fi

# Verify installation
if command -v glive &> /dev/null; then
    echo -e "${GREEN}GLive installed successfully!${NC}"
    echo ""
    glive --help 2>/dev/null || echo "Run 'glive --help' to get started"
else
    echo -e "${YELLOW}Installation complete. You may need to add $INSTALL_DIR to your PATH.${NC}"
    echo "Add this to your ~/.bashrc or ~/.zshrc:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi
