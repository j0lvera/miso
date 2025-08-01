#!/bin/bash
#
# This script is for installing the miso CLI tool.
# It determines the user's OS and architecture, then downloads the
# appropriate binary from the latest GitHub release.

set -e

main() {
  # Define repository and binary details
  REPO_OWNER="j0lvera"
  REPO_NAME="miso"
  BINARY_NAME="miso"
  INSTALL_DIR="$HOME/.${BINARY_NAME}"

  # Detect OS and architecture
  OS="$(uname -s)"
  ARCH="$(uname -m)"

  # Map OS and arch to the naming convention used by GoReleaser
  case $OS in
    Linux) OS_NAME="linux" ;;
    Darwin) OS_NAME="darwin" ;;
    *)
      echo "Error: Unsupported OS: $OS"
      exit 1
      ;;
  esac

  case $ARCH in
    x86_64 | amd64) ARCH_NAME="amd64" ;;
    arm64 | aarch64) ARCH_NAME="arm64" ;;
    *)
      echo "Error: Unsupported architecture: $ARCH"
      exit 1
      ;;
  esac

  # Determine the version to install (latest by default)
  if [ -n "$1" ]; then
    VERSION="$1"
  else
    # Get the latest release tag from the GitHub API
    LATEST_RELEASE_URL="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"

    # Use GITHUB_TOKEN for authenticated requests to avoid rate limits, especially in CI
    if [ -n "${GITHUB_TOKEN:-}" ]; then
      RELEASE_JSON=$(curl -s -H "Authorization: Bearer ${GITHUB_TOKEN}" "$LATEST_RELEASE_URL")
    else
      RELEASE_JSON=$(curl -s "$LATEST_RELEASE_URL")
    fi

    # Check if a release was found
    if echo "$RELEASE_JSON" | grep -q '"message": "Not Found"'; then
      echo "Error: No releases found for ${REPO_OWNER}/${REPO_NAME}."
      echo "Please create a release on GitHub before using the install script."
      exit 1
    fi

    VERSION=$(echo "$RELEASE_JSON" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
      echo "Error: Could not determine the latest release version from the API response."
      exit 1
    fi
  fi

  # GoReleaser uses the version without the 'v' prefix in the artifact name
  VERSION_FOR_FILENAME=${VERSION#v}

  # Construct the download URL
  TARBALL_NAME="${BINARY_NAME}_${VERSION_FOR_FILENAME}_${OS_NAME}_${ARCH_NAME}.tar.gz"
  DOWNLOAD_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/${TARBALL_NAME}"

  echo "Installing ${BINARY_NAME} version ${VERSION} for ${OS_NAME}/${ARCH_NAME}..."

  # Create a temporary directory for the download
  TMP_DIR=$(mktemp -d)
  trap 'rm -rf -- "$TMP_DIR"' EXIT

  # Download and extract the binary
  echo "Downloading from ${DOWNLOAD_URL}"
  curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_DIR}/${TARBALL_NAME}"
  tar -xzf "${TMP_DIR}/${TARBALL_NAME}" -C "${TMP_DIR}"

  # Install the binary
  mkdir -p "${INSTALL_DIR}/bin"
  mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/bin/"
  chmod +x "${INSTALL_DIR}/bin/${BINARY_NAME}"

  echo ""
  echo "${BINARY_NAME} was installed successfully to ${INSTALL_DIR}/bin/${BINARY_NAME}"

  # Add to PATH if not already there
  SHELL_PROFILE=""
  INSTALL_PATH_CMD=""
  case $SHELL in
    */bash)
      SHELL_PROFILE="$HOME/.bashrc"
      INSTALL_PATH_CMD="export PATH=\"${INSTALL_DIR}/bin:\$PATH\""
      ;;
    */zsh)
      SHELL_PROFILE="$HOME/.zshrc"
      INSTALL_PATH_CMD="export PATH=\"${INSTALL_DIR}/bin:\$PATH\""
      ;;
    */fish)
      SHELL_PROFILE="$HOME/.config/fish/config.fish"
      INSTALL_PATH_CMD="fish_add_path \"${INSTALL_DIR}/bin\""
      ;;
    *)
      SHELL_PROFILE="$HOME/.profile"
      INSTALL_PATH_CMD="export PATH=\"${INSTALL_DIR}/bin:\$PATH\""
      ;;
  esac

  if ! grep -q "${INSTALL_DIR}/bin" "$SHELL_PROFILE" 2>/dev/null; then
    echo "Adding ${BINARY_NAME} to your PATH in ${SHELL_PROFILE}..."
    if [[ "$SHELL" == */fish ]]; then
      mkdir -p "$(dirname "$SHELL_PROFILE")"
    fi
    echo "" >> "$SHELL_PROFILE"
    echo "# miso CLI" >> "$SHELL_PROFILE"
    echo "$INSTALL_PATH_CMD" >> "$SHELL_PROFILE"
    echo ""
    echo "Please restart your shell or run 'source ${SHELL_PROFILE}' to apply the changes."
  else
    echo "${BINARY_NAME} is already in your PATH."
  fi
}

main "$@"
