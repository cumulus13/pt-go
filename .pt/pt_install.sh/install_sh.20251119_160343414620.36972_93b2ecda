#!/usr/bin/env bash
#
# PT - Clipboard to File Tool Installer
# Author: Hadi Cahyadi <cumulus13@gmail.com>
# Repository: https://github.com/cumulus13/pt-go
# License: MIT
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/cumulus13/pt-go/main/install.sh | bash
#   wget -qO- https://raw.githubusercontent.com/cumulus13/pt-go/main/install.sh | bash
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

# Configuration
REPO="cumulus13/pt-go"
BINARY_NAME="pt"
INSTALL_DIR="/usr/local/bin"
TEMP_DIR=$(mktemp -d)
VERSION="latest"

# Cleanup on exit
cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

# Print functions
print_header() {
    echo ""
    echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${RESET}"
    echo -e "${CYAN}║${RESET}  ${BOLD}PT - Clipboard to File Tool Installer${RESET}              ${CYAN}║${RESET}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${RESET}"
    echo ""
}

print_step() {
    echo -e "${BLUE}▶${RESET} $1"
}

print_success() {
    echo -e "${GREEN}✓${RESET} $1"
}

print_error() {
    echo -e "${RED}✗${RESET} $1" >&2
}

print_warning() {
    echo -e "${YELLOW}⚠${RESET} $1"
}

print_info() {
    echo -e "${CYAN}ℹ${RESET} $1"
}

# Detect OS and architecture
detect_platform() {
    print_step "Detecting platform..."
    
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case "$OS" in
        linux*)
            OS="linux"
            ;;
        darwin*)
            OS="darwin"
            ;;
        mingw*|msys*|cygwin*)
            OS="windows"
            BINARY_NAME="pt.exe"
            ;;
        *)
            print_error "Unsupported operating system: $OS"
            exit 1
            ;;
    esac
    
    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l|armv6l)
            ARCH="arm"
            ;;
        i386|i686)
            ARCH="386"
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    print_success "Platform detected: ${BOLD}${OS}/${ARCH}${RESET}"
}

# Check if running with sufficient privileges
check_privileges() {
    if [ "$OS" != "windows" ]; then
        if [ "$EUID" -eq 0 ]; then
            print_warning "Running as root. Binary will be installed to ${INSTALL_DIR}"
        elif [ -w "$INSTALL_DIR" ]; then
            print_success "Have write permission to ${INSTALL_DIR}"
        else
            print_warning "No write permission to ${INSTALL_DIR}"
            print_info "Will attempt to use sudo for installation"
            NEED_SUDO=true
        fi
    fi
}

# Check dependencies
check_dependencies() {
    print_step "Checking dependencies..."
    
    local missing_deps=()
    
    # Check for required commands
    for cmd in curl tar; do
        if ! command -v "$cmd" &> /dev/null; then
            missing_deps+=("$cmd")
        fi
    done
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        print_error "Missing required dependencies: ${missing_deps[*]}"
        print_info "Please install them using your package manager:"
        print_info "  Ubuntu/Debian: sudo apt-get install ${missing_deps[*]}"
        print_info "  Fedora/RHEL:   sudo dnf install ${missing_deps[*]}"
        print_info "  macOS:         brew install ${missing_deps[*]}"
        exit 1
    fi
    
    print_success "All dependencies found"
}

# Check if Go is installed
check_go() {
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        print_success "Go ${GO_VERSION} found"
        return 0
    else
        print_warning "Go not found (optional for building from source)"
        return 1
    fi
}

# Get latest version from GitHub
get_latest_version() {
    print_step "Fetching latest version..."
    
    VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$VERSION" ]; then
        print_warning "Could not fetch latest version, will build from source"
        return 1
    fi
    
    print_success "Latest version: ${BOLD}${VERSION}${RESET}"
    return 0
}

# Download and install prebuilt binary
install_prebuilt() {
    print_step "Downloading prebuilt binary..."
    
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/pt-${OS}-${ARCH}.tar.gz"
    
    cd "$TEMP_DIR"
    
    if curl -sL "$DOWNLOAD_URL" -o pt.tar.gz; then
        print_success "Downloaded binary"
        
        print_step "Extracting archive..."
        tar -xzf pt.tar.gz
        
        if [ -f "$BINARY_NAME" ]; then
            print_success "Extracted successfully"
            
            print_step "Installing to ${INSTALL_DIR}..."
            
            if [ "$NEED_SUDO" = true ]; then
                sudo install -m 755 "$BINARY_NAME" "${INSTALL_DIR}/${BINARY_NAME}"
            else
                install -m 755 "$BINARY_NAME" "${INSTALL_DIR}/${BINARY_NAME}"
            fi
            
            print_success "Installed to ${INSTALL_DIR}/${BINARY_NAME}"
            return 0
        else
            print_error "Binary not found in archive"
            return 1
        fi
    else
        print_warning "Prebuilt binary not available for ${OS}/${ARCH}"
        return 1
    fi
}

# Build and install from source
install_from_source() {
    print_step "Building from source..."
    
    if ! command -v go &> /dev/null; then
        print_error "Go is required to build from source"
        print_info "Please install Go from: https://golang.org/dl/"
        exit 1
    fi
    
    print_step "Cloning repository..."
    cd "$TEMP_DIR"
    
    if git clone --depth 1 "https://github.com/${REPO}.git" pt-repo 2>/dev/null; then
        cd pt-repo
    else
        # Fallback if git is not available
        print_warning "Git not found, downloading source archive..."
        curl -sL "https://github.com/${REPO}/archive/refs/heads/main.tar.gz" -o source.tar.gz
        tar -xzf source.tar.gz
        cd pt-go-main || cd pt-*
    fi
    
    print_success "Source code downloaded"
    
    print_step "Building binary..."
    
    # Try different possible locations for main.go
    if [ -f "pt/main.go" ]; then
        go build -o "$BINARY_NAME" -ldflags "-s -w" pt/main.go
    elif [ -f "main.go" ]; then
        go build -o "$BINARY_NAME" -ldflags "-s -w" main.go
    elif [ -f "cmd/pt/main.go" ]; then
        go build -o "$BINARY_NAME" -ldflags "-s -w" cmd/pt/main.go
    else
        print_error "Could not find main.go"
        exit 1
    fi
    
    if [ -f "$BINARY_NAME" ]; then
        print_success "Binary built successfully"
        
        print_step "Installing to ${INSTALL_DIR}..."
        
        if [ "$NEED_SUDO" = true ]; then
            sudo install -m 755 "$BINARY_NAME" "${INSTALL_DIR}/${BINARY_NAME}"
        else
            install -m 755 "$BINARY_NAME" "${INSTALL_DIR}/${BINARY_NAME}"
        fi
        
        print_success "Installed to ${INSTALL_DIR}/${BINARY_NAME}"
        return 0
    else
        print_error "Build failed"
        exit 1
    fi
}

# Verify installation
verify_installation() {
    print_step "Verifying installation..."
    
    if command -v pt &> /dev/null; then
        INSTALLED_VERSION=$(pt --version 2>&1 | head -n 1)
        print_success "PT installed successfully!"
        print_info "Version: ${INSTALLED_VERSION}"
        return 0
    else
        print_error "Installation verification failed"
        print_info "The binary was installed to ${INSTALL_DIR}/${BINARY_NAME}"
        print_info "Make sure ${INSTALL_DIR} is in your PATH"
        return 1
    fi
}

# Install delta (optional)
install_delta() {
    echo ""
    print_step "Checking for delta (required for diff feature)..."
    
    if command -v delta &> /dev/null; then
        DELTA_VERSION=$(delta --version | head -n 1)
        print_success "Delta already installed: ${DELTA_VERSION}"
        return 0
    fi
    
    print_warning "Delta not found"
    print_info "Delta is required for the diff feature (pt -d)"
    echo ""
    
    read -p "Would you like to see delta installation instructions? (y/N): " -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo ""
        print_info "Install delta using one of these methods:"
        echo ""
        echo "  ${BOLD}macOS:${RESET}"
        echo "    brew install git-delta"
        echo ""
        echo "  ${BOLD}Ubuntu/Debian:${RESET}"
        echo "    sudo apt install git-delta"
        echo ""
        echo "  ${BOLD}Fedora/RHEL:${RESET}"
        echo "    sudo dnf install git-delta"
        echo ""
        echo "  ${BOLD}Arch Linux:${RESET}"
        echo "    sudo pacman -S git-delta"
        echo ""
        echo "  ${BOLD}Windows (Chocolatey):${RESET}"
        echo "    choco install delta"
        echo ""
        echo "  ${BOLD}Windows (Scoop):${RESET}"
        echo "    scoop install delta"
        echo ""
        echo "  ${BOLD}Using Cargo:${RESET}"
        echo "    cargo install git-delta"
        echo ""
        echo "  ${BOLD}Or download from:${RESET}"
        echo "    https://github.com/dandavison/delta/releases"
        echo ""
    fi
}

# Print usage instructions
print_usage() {
    echo ""
    echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${RESET}"
    echo -e "${CYAN}║${RESET}  ${BOLD}Quick Start Guide${RESET}                                       ${CYAN}║${RESET}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${RESET}"
    echo ""
    echo -e "${BOLD}Basic Usage:${RESET}"
    echo "  pt <filename>               Write clipboard to file"
    echo "  pt + <filename>             Append clipboard to file"
    echo ""
    echo -e "${BOLD}Backup Management:${RESET}"
    echo "  pt -l <filename>            List all backups"
    echo "  pt -r <filename>            Restore backup (interactive)"
    echo "  pt -r <filename> --last     Restore most recent backup"
    echo ""
    echo -e "${BOLD}Diff Operations:${RESET}"
    echo "  pt -d <filename>            Compare with backup (interactive)"
    echo "  pt -d <filename> --last     Compare with most recent backup"
    echo ""
    echo -e "${BOLD}Information:${RESET}"
    echo "  pt --help                   Show detailed help"
    echo "  pt --version                Show version info"
    echo ""
    echo -e "${BOLD}Examples:${RESET}"
    echo "  ${CYAN}# Copy text, then save to file${RESET}"
    echo "  pt notes.txt"
    echo ""
    echo "  ${CYAN}# List all versions of a file${RESET}"
    echo "  pt -l notes.txt"
    echo ""
    echo "  ${CYAN}# Compare with last backup${RESET}"
    echo "  pt -d notes.txt --last"
    echo ""
    echo -e "${BOLD}Features:${RESET}"
    echo "  ✓ Automatic timestamped backups"
    echo "  ✓ Recursive file search (up to 10 levels)"
    echo "  ✓ Beautiful diff with delta integration"
    echo "  ✓ Interactive file selection"
    echo "  ✓ Production-ready security"
    echo ""
    echo -e "${BOLD}Documentation:${RESET}"
    echo "  https://github.com/${REPO}"
    echo ""
}

# Main installation function
main() {
    print_header
    
    # Platform detection
    detect_platform
    
    # Check privileges
    check_privileges
    
    # Check dependencies
    check_dependencies
    
    # Check for Go
    HAS_GO=false
    if check_go; then
        HAS_GO=true
    fi
    
    echo ""
    
    # Try to get latest version and install prebuilt binary
    if get_latest_version && install_prebuilt; then
        # Prebuilt installation successful
        :
    else
        # Fall back to building from source
        print_warning "Prebuilt binary not available, building from source..."
        echo ""
        
        if [ "$HAS_GO" = false ]; then
            print_error "Go is required to build from source"
            print_info "Please install Go from: https://golang.org/dl/"
            print_info "Or wait for prebuilt binaries to be available"
            exit 1
        fi
        
        install_from_source
    fi
    
    echo ""
    
    # Verify installation
    if verify_installation; then
        # Check for delta
        install_delta
        
        # Print success message
        echo ""
        echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${RESET}"
        echo -e "${GREEN}║${RESET}  ${BOLD}Installation Complete!${RESET} 🎉                             ${GREEN}║${RESET}"
        echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${RESET}"
        
        # Print usage
        print_usage
        
        # Final message
        echo -e "${CYAN}Thank you for installing PT!${RESET}"
        echo -e "Report issues: ${BLUE}https://github.com/${REPO}/issues${RESET}"
        echo ""
        
        exit 0
    else
        print_error "Installation completed but verification failed"
        print_info "Please check your PATH and try running: ${INSTALL_DIR}/${BINARY_NAME} --version"
        exit 1
    fi
}

# Run main function
main "$@"