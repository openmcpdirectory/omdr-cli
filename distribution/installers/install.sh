#!/bin/sh
# OMDR CLI installer script for Unix-like systems

set -e

REPO="openmcpdirectory/omdr-cli"
BINARY_NAME="omdr"

get_latest_release() {
  curl --silent "https://api.github.com/repos/$REPO/releases/latest" |
    grep '"tag_name":' |
    sed -E 's/.*"([^"]+)".*/\1/'
}

detect_platform() {
  OS="$(uname -s)"
  ARCH="$(uname -m)"

  case "$OS" in
    Darwin)
      PLATFORM="Darwin"
      ;;
    Linux)
      PLATFORM="Linux"
      ;;
    *)
      echo "Unsupported OS: $OS"
      exit 1
      ;;
  esac

  case "$ARCH" in
    x86_64)
      ARCH_NAME="x86_64"
      ;;
    arm64|aarch64)
      ARCH_NAME="arm64"
      ;;
    *)
      echo "Unsupported architecture: $ARCH"
      exit 1
      ;;
  esac
}

main() {
  detect_platform
  
  VERSION="${1:-$(get_latest_release)}"
  VERSION="${VERSION#v}"
  
  FILENAME="${BINARY_NAME}_${PLATFORM}_${ARCH_NAME}.tar.gz"
  URL="https://github.com/$REPO/releases/download/v${VERSION}/${FILENAME}"
  
  echo "Downloading OMDR CLI v${VERSION} for ${PLATFORM} ${ARCH_NAME}..."
  
  TMPDIR=$(mktemp -d)
  trap "rm -rf $TMPDIR" EXIT
  
  curl -sL "$URL" | tar -xz -C "$TMPDIR"
  
  INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
  
  if [ -w "$INSTALL_DIR" ]; then
    mv "$TMPDIR/$BINARY_NAME" "$INSTALL_DIR/"
  else
    echo "Installing to $INSTALL_DIR requires sudo..."
    sudo mv "$TMPDIR/$BINARY_NAME" "$INSTALL_DIR/"
  fi
  
  chmod +x "$INSTALL_DIR/$BINARY_NAME"
  
  echo ""
  echo "OMDR CLI v${VERSION} installed successfully!"
  echo "Run 'omdr --help' to get started."
}

main "$@"
