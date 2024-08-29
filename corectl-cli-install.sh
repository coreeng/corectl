#!/bin/bash

set -euo pipefail

# Check if the script is being run as root (UID 0)
if [ "$EUID" -ne 0 ]; then
  echo "This script must be run with sudo or as root."
  exit 1
fi

# =============================================================================
# Ensure no command-line arguments are passed
# =============================================================================

if [ "$#" -ne 0 ]; then
  echo "This script does not accept command-line arguments."
  echo "Specify the version using the CORECTL_VERSION environment variable."
  exit 1
fi

# =============================================================================
# Define helper functions
# =============================================================================

text_bold() {
  echo -e "\033[1m$1\033[0m"
}
text_title() {
  echo ""
  text_bold "$1"
  if [ -n "$2" ]; then echo "$2"; fi
}
text_title_error() {
    echo ""
    echo -e "\033[1;31m$1\033[00m"
}

# =============================================================================
# Define base variables
# =============================================================================

NAME="corectl"
GITHUB_REPO="coreeng/corectl"

# Debugging output to check if CORECTL_VERSION is set

# Determine the version to use based on CORECTL_VERSION environment variable
VERSION="${CORECTL_VERSION:-latest}"

echo "CORECTL_VERSION is set to: '${VERSION}'"
# =============================================================================
# Get the actual version if 'latest' is specified
# =============================================================================

if [ "$VERSION" == "latest" ]; then
  text_title "Fetching latest version..." ""
  VERSION=$(curl -s "https://api.github.com/repos/coreeng/corectl/releases/latest" | awk -F'"' '/"tag_name":/ {print $4}')
  if [ -z "$VERSION" ]; then
    echo "Failed to fetch the latest version." ""
    exit 1
  fi
  text_title "Latest version found" " $VERSION"
fi

DOWNLOAD_BASE_URL="https://github.com/$GITHUB_REPO/releases/download/$VERSION"

# =============================================================================
# Define binary list for supported OS & Arch
# - this is a map of "OS:Arch" -> "download binary name"
# - you can remove or add to this list as needed
# =============================================================================

OS_LIST=("Linux:x86_64" "Darwin:x86_64" "Darwin:arm64")
BINARIES=("corectl_Linux_x86_64.tar.gz" "corectl_Darwin_x86_64.tar.gz" "corectl_Darwin_arm64.tar.gz")

# =============================================================================
# Get the user's OS and Arch
# =============================================================================

OS="$(uname -s)"
ARCH="$(uname -m)"
SYSTEM="${OS}:${ARCH}"

# =============================================================================
# Match a binary to check if the system is supported
# =============================================================================

BINARY=""
for i in "${!OS_LIST[@]}"; do
  if [[ "${OS_LIST[$i]}" == "$SYSTEM" ]]; then
    BINARY="${BINARIES[$i]}"
    break
  fi
done

if [ -z "$BINARY" ]; then
  text_title_error "Error"
  echo " Unsupported OS or arch: ${SYSTEM}"
  echo ""
  exit 1
fi

# =============================================================================
# Set the default installation variables
# =============================================================================

INSTALL_DIR="/usr/local/bin"
DOWNLOAD_URL="$DOWNLOAD_BASE_URL/$BINARY"
CHECKSUM_FILE="corectl_${VERSION#v}_checksums.txt"
CHECKSUM_URL="$DOWNLOAD_BASE_URL/$CHECKSUM_FILE"
# =============================================================================
# Download the binary and the checksum file
# =============================================================================

text_title "Downloading Binary" "$DOWNLOAD_URL"
curl -LO --proto '=https' --tlsv1.2 -sSf "$DOWNLOAD_URL"
curl -LO --proto '=https' --tlsv1.2 -sSf "$CHECKSUM_URL"

# =============================================================================
# Verify the downloaded binary against the checksum
# =============================================================================

text_title "Verifying Checksum" ""

EXPECTED_CHECKSUM=$(grep "$BINARY" "$CHECKSUM_FILE" | awk '{print $1}')
ACTUAL_CHECKSUM=""
if [[ "${OS}" == "Linux" ]]; then
    ACTUAL_CHECKSUM=$(sha256sum "$BINARY" | awk '{print $1}')
else
    ACTUAL_CHECKSUM=$(shasum -a 256 "$BINARY" | awk '{print $1}')
fi

if [ "$EXPECTED_CHECKSUM" != "$ACTUAL_CHECKSUM" ]; then
  text_title_error "" "Checksum verification failed!"
  echo "Expected: $EXPECTED_CHECKSUM"
  echo "Actual: $ACTUAL_CHECKSUM"
  exit 1
fi

text_title "" "Checksum verification succeeded."

# =============================================================================
# Make binary executable and move to install directory with appropriate name
# =============================================================================

tar -xzf "$BINARY"

# Locate the executable
EXECUTABLE=$(tar -tzf "$BINARY" | grep -m 1 -E '^corectl$')

if [ -z "$EXECUTABLE" ]; then
    text_title_error "Executable 'corectl' not found in the tarball."
    exit 1
fi

# Move the executable to /usr/local/bin
sudo mv "$EXECUTABLE" "$INSTALL_DIR"
if [ $? -ne 0 ]; then
  text_title_error "Error: Failed to move the executable to $INSTALL_DIR"
  exit 1
fi

# Make the binary executable
sudo chmod +x $INSTALL_DIR/corectl
if [ $? -ne 0 ]; then
  text_title_error "Error: Failed to make corectl executable."
  exit 1
fi

# Clean up the downloaded files
rm "$BINARY" "$CHECKSUM_FILE"
if [ $? -ne 0 ]; then
  text_title_error "Error: Failed to remove the binary or checksum file."
  exit 1
fi


# Verify installation
if command -v corectl &> /dev/null; then
    text_title "corectl successfully installed!" ""
else
    text_title_error "corectl installation failed."
    exit 1
fi

# =============================================================================
# Display post install message
# =============================================================================

text_title "Installation Complete" " Run $NAME --help for more information"
echo ""
