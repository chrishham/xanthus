sudo apt install build-essential -y
sudo apt install ripgrep

curl https://get.volta.sh | bash

#!/bin/bash

# Ensure the script is idempotent
VOLTA_HOME_LINE='export VOLTA_HOME="$HOME/.volta"'
PATH_LINE='export PATH="$VOLTA_HOME/bin:$PATH"'

echo "🔧 Setting up Volta environment in ~/.bashrc"

# Add VOLTA_HOME to ~/.bashrc if not already present
grep -qxF "$VOLTA_HOME_LINE" ~/.bashrc || echo "$VOLTA_HOME_LINE" >> ~/.bashrc
grep -qxF "$PATH_LINE" ~/.bashrc || echo "$PATH_LINE" >> ~/.bashrc

# Reload .bashrc
echo "🔄 Reloading ~/.bashrc"
source ~/.bashrc

# Check if Volta is installed
if command -v volta >/dev/null 2>&1; then
    echo "✅ Volta is installed: $(volta --version)"
else
    echo "❌ Volta is not found in PATH. Please re-login or restart your shell."
fi

volta install node

npm install -g @anthropic-ai/claude-code
npm install -g ccusage


install-go-latest.sh

#!/bin/bash


set -euo pipefail


echo "📦 Detecting system architecture..."


ARCH=$(uname -m)
case "$ARCH" in
 x86_64) ARCH="amd64" ;;
 aarch64 | arm64) ARCH="arm64" ;;
 *) echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
esac


echo "⚙️ Architecture detected: $ARCH"


echo "📦 Fetching latest stable Go version..."


# Fetch list of stable Go versions for linux and your arch
VERSIONS=$(curl -s https://go.dev/dl/ | grep -oP "go[0-9]+\.[0-9]+(\.[0-9]+)?\.linux-${ARCH}\.tar\.gz" | sed -E "s/\.linux-${ARCH}\.tar\.gz//" | sort -Vr)


LATEST_VERSION=$(echo "$VERSIONS" | head -n1)


if [ -z "$LATEST_VERSION" ]; then
 echo "❌ Could not determine the latest Go version for architecture $ARCH."
 exit 1
fi


TARFILE="${LATEST_VERSION}.linux-${ARCH}.tar.gz"
DOWNLOAD_URL="https://go.dev/dl/${TARFILE}"


echo "🔽 Downloading $TARFILE from $DOWNLOAD_URL ..."
curl -LO "$DOWNLOAD_URL"


echo "🧹 Removing old Go installation (if any)..."
sudo rm -rf /usr/local/go


echo "📦 Extracting Go to /usr/local..."
sudo tar -C /usr/local -xzf "$TARFILE"
rm "$TARFILE"


echo "⚙️ Adding Go to PATH..."
PROFILE_FILE="$HOME/.bashrc"
GO_LINE='export PATH=$PATH:/usr/local/go/bin'


if ! grep -Fxq "$GO_LINE" "$PROFILE_FILE"; then
 echo "$GO_LINE" >> "$PROFILE_FILE"
 echo "✅ Added Go to PATH in $PROFILE_FILE"
else
 echo "ℹ️ Go path already present in $PROFILE_FILE"
fi


echo "🔄 Reloading shell config..."
source "$PROFILE_FILE"


echo "✅ Go installation complete!"
go version