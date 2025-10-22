#!/usr/bin/env bash
set -euo pipefail

# Determine OS
OS=$(uname -s)
case "$OS" in
	Darwin)
		OS="darwin";;
	Linux)
		OS="linux";;
	*)
		echo "Unsupported OS: $OS" >&2; exit 1;;
esac

# Determine architecture
ARCH=$(uname -m)
case "$ARCH" in
	x86_64|amd64)
		ARCH="x86_64";;
	aarch64|arm64)
		ARCH="arm64";;
	*)
		echo "Unsupported architecture: $ARCH" >&2; exit 1;;
esac

# GitHub repo information
REPO_OWNER="coreeng"
REPO_NAME="corectl"

# Fetch latest release tag from GitHub API
echo "Fetching latest release information..."
LATEST_TAG=$(curl -s "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest" \
	| grep -oE '"tag_name": "([^"]+)"' | head -1 | cut -d '"' -f4)

if [[ -z "$LATEST_TAG" ]]; then
	echo "Failed to fetch latest release tag." >&2
	exit 1
fi

echo "Latest release: $LATEST_TAG"

# Construct download URL for the appropriate tarball
ASSET_NAME="corectl_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$LATEST_TAG/$ASSET_NAME"

# Create a temporary directory and ensure cleanup on exit
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

# Download and extract in one step
echo "Downloading and extracting $ASSET_NAME..."
curl -sL "$DOWNLOAD_URL" | tar -xz -C "$tmpdir"

# Install the binary to /usr/local/bin
echo "Installing corectl to /usr/local/bin (requires sudo)..."
sudo install -m 755 "$tmpdir/corectl" /usr/local/bin/corectl

# Verify installation
if ! /usr/local/bin/corectl version | grep -q "$LATEST_TAG"; then
	echo "corectl version mismatch or installation failed." >&2
	exit 1
fi

echo "corectl $LATEST_TAG installed and verified successfully!"

# Warn if the installed version is not the first in the path
active_corectl=$(command -v corectl 2>/dev/null || true)
if [[ -n "$active_corectl" && "$active_corectl" != "/usr/local/bin/corectl" ]]; then
	echo "" >&2
	echo "WARNING: When you run 'corectl', the version at $active_corectl will be used instead!" >&2
	echo "To use the newly installed version, either:" >&2
	echo "	1. Remove the old installation: rm $active_corectl" >&2
	echo "	2. Update your PATH to prioritize /usr/local/bin" >&2
	echo "	3. Use the absolute path: /usr/local/bin/corectl" >&2
	echo "" >&2
fi
