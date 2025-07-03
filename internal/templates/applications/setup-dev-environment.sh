#!/bin/bash

# Development Environment Setup Script for Code-Server
# This script installs additional development tools and packages
# Run this script manually when you need extra development tools

set -e

echo "🚀 Starting development environment setup..."
echo "This script will install additional development tools and packages."
echo "You can run this at any time to enhance your development environment."
echo ""

# Function to check if running as root
check_root() {
    if [ "$EUID" -eq 0 ]; then
        echo "❌ This script should not be run as root!"
        echo "Please run as the regular user (coder)"
        exit 1
    fi
}

# Function to install system packages (requires sudo)
install_system_packages() {
    echo "🔧 Installing additional system packages..."
    
    # Check if we need to install packages
    if ! command -v build-essential &> /dev/null || ! command -v rg &> /dev/null; then
        echo "Installing build-essential, ripgrep, and development tools..."
        sudo apt-get update
        sudo apt-get install -y build-essential ripgrep curl wget git vim nano \
            htop tree jq unzip zip python3 python3-pip docker.io
    else
        echo "✅ System packages already installed"
    fi
}

# Function to install Go
install_go() {
    if command -v go &> /dev/null; then
        echo "✅ Go already installed: $(go version)"
        return 0
    fi
    
    echo "🐹 Installing Go..."
    
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64) ARCH="amd64" ;;
        aarch64 | arm64) ARCH="arm64" ;;
        *) echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    
    # Use a specific stable Go version
    GO_VERSION="1.23.4"
    TARFILE="go${GO_VERSION}.linux-${ARCH}.tar.gz"
    DOWNLOAD_URL="https://go.dev/dl/${TARFILE}"
    
    echo "🔽 Downloading Go ${GO_VERSION}..."
    cd /tmp
    curl -LO "$DOWNLOAD_URL"
    
    if [ ! -f "${TARFILE}" ]; then
        echo "❌ Failed to download Go tarball"
        exit 1
    fi
    
    echo "📦 Extracting Go to /usr/local..."
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "${TARFILE}"
    rm "${TARFILE}"
    
    # Add Go to PATH if not already there
    if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
        echo 'export PATH="$PATH:/usr/local/go/bin"' >> ~/.bashrc
        echo 'export GOPATH="$HOME/go"' >> ~/.bashrc
        echo 'export PATH="$PATH:$GOPATH/bin"' >> ~/.bashrc
    fi
    
    echo "✅ Go installation complete"
}

# Function to install Node.js via Volta
install_nodejs() {
    if command -v node &> /dev/null; then
        echo "✅ Node.js already installed: $(node --version)"
        return 0
    fi
    
    echo "🌐 Installing Volta (Node.js version manager)..."
    
    if [ ! -d "$HOME/.volta" ]; then
        curl https://get.volta.sh | bash
    fi
    
    # Setup Volta environment
    export VOLTA_HOME="$HOME/.volta"
    export PATH="$VOLTA_HOME/bin:$PATH"
    
    echo "📦 Installing Node.js..."
    ~/.volta/bin/volta install node@20
    
    # Add Volta to PATH if not already there
    if ! grep -q "VOLTA_HOME" ~/.bashrc; then
        echo 'export VOLTA_HOME="$HOME/.volta"' >> ~/.bashrc
        echo 'export PATH="$VOLTA_HOME/bin:$PATH"' >> ~/.bashrc
    fi
    
    echo "✅ Node.js installation complete"
}

# Function to install global npm packages
install_npm_packages() {
    echo "🔧 Installing global npm packages..."
    
    # Ensure we have npm
    if ! command -v npm &> /dev/null; then
        echo "❌ npm not found. Please install Node.js first."
        return 1
    fi
    
    # Install Claude Code and ccusage
    npm install -g @anthropic-ai/claude-code ccusage
    
    echo "✅ Global npm packages installed"
}

# Function to install Python packages
install_python_packages() {
    echo "🐍 Installing Python packages..."
    
    # Install common Python development packages
    pip3 install --user black flake8 mypy pytest requests numpy pandas matplotlib jupyter
    
    echo "✅ Python packages installed"
}

# Function to install Rust
install_rust() {
    if command -v rustc &> /dev/null; then
        echo "✅ Rust already installed: $(rustc --version)"
        return 0
    fi
    
    echo "🦀 Installing Rust..."
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
    
    # Add Rust to PATH
    if ! grep -q "cargo/bin" ~/.bashrc; then
        echo 'export PATH="$HOME/.cargo/bin:$PATH"' >> ~/.bashrc
    fi
    
    echo "✅ Rust installation complete"
}

# Function to setup development directories
setup_directories() {
    echo "📁 Setting up development directories..."
    
    mkdir -p ~/workspace ~/go/src ~/go/bin ~/go/pkg ~/.local/share/code-server/User
    
    echo "✅ Development directories created"
}

# Function to setup useful aliases
setup_aliases() {
    echo "🔧 Setting up useful aliases..."
    
    # Check if aliases are already in bashrc
    if ! grep -q "# Xanthus Development Aliases" ~/.bashrc; then
        cat >> ~/.bashrc << 'EOF'

# Xanthus Development Aliases
alias ll='ls -alF'
alias la='ls -A'
alias l='ls -CF'
alias ..='cd ..'
alias ...='cd ../..'

# Development shortcuts
alias gst='git status'
alias gco='git checkout'
alias gc='git commit'
alias gp='git push'
alias gl='git log --oneline'

# Docker shortcuts
alias dps='docker ps'
alias dimg='docker images'
alias dlog='docker logs'

# Python shortcuts
alias py='python3'
alias pip='pip3'

EOF
    fi
    
    echo "✅ Aliases configured"
}

# Function to install additional tools
install_additional_tools() {
    echo "🛠️ Installing additional development tools..."
    
    # Install lazygit
    if ! command -v lazygit &> /dev/null; then
        echo "Installing lazygit..."
        LAZYGIT_VERSION=$(curl -s "https://api.github.com/repos/jesseduffield/lazygit/releases/latest" | grep -Po '"tag_name": "v\K[^"]*')
        curl -Lo lazygit.tar.gz "https://github.com/jesseduffield/lazygit/releases/latest/download/lazygit_${LAZYGIT_VERSION}_Linux_x86_64.tar.gz"
        tar xf lazygit.tar.gz lazygit
        sudo install lazygit /usr/local/bin
        rm lazygit.tar.gz lazygit
    fi
    
    # Install fd (find alternative)
    if ! command -v fd &> /dev/null; then
        echo "Installing fd..."
        sudo apt-get install -y fd-find
    fi
    
    # Install bat (cat alternative)
    if ! command -v bat &> /dev/null; then
        echo "Installing bat..."
        sudo apt-get install -y bat
    fi
    
    echo "✅ Additional tools installed"
}

# Function to show completion message
show_completion() {
    echo ""
    echo "🎉 Development environment setup complete!"
    echo ""
    echo "Installed tools and packages:"
    echo "  📦 System packages: build-essential, ripgrep, curl, wget, git, vim, nano, htop, tree, jq"
    echo "  🐹 Go: $(go version 2>/dev/null || echo 'Not installed')"
    echo "  🌐 Node.js: $(node --version 2>/dev/null || echo 'Not installed')"
    echo "  🐍 Python: $(python3 --version 2>/dev/null || echo 'Not installed')"
    echo "  🦀 Rust: $(rustc --version 2>/dev/null || echo 'Not installed')"
    echo "  🛠️ Additional tools: lazygit, fd, bat"
    echo ""
    echo "To activate the new environment, run:"
    echo "  source ~/.bashrc"
    echo ""
    echo "You can run this script again anytime to install additional tools or update existing ones."
}

# Main execution
main() {
    echo "🚀 Xanthus Code-Server Development Environment Setup"
    echo "=================================================="
    echo ""
    
    check_root
    
    echo "Select what you want to install:"
    echo "1) Everything (recommended)"
    echo "2) System packages only"
    echo "3) Go only"
    echo "4) Node.js only"
    echo "5) Python packages only"
    echo "6) Rust only"
    echo "7) Additional tools only"
    echo "8) Custom selection"
    echo ""
    read -p "Enter your choice (1-8): " choice
    
    case $choice in
        1)
            install_system_packages
            install_go
            install_nodejs
            install_npm_packages
            install_python_packages
            install_rust
            setup_directories
            setup_aliases
            install_additional_tools
            ;;
        2)
            install_system_packages
            ;;
        3)
            install_go
            ;;
        4)
            install_nodejs
            install_npm_packages
            ;;
        5)
            install_python_packages
            ;;
        6)
            install_rust
            ;;
        7)
            install_additional_tools
            ;;
        8)
            echo "Custom selection mode:"
            read -p "Install system packages? (y/n): " sys && [[ $sys == "y" ]] && install_system_packages
            read -p "Install Go? (y/n): " go && [[ $go == "y" ]] && install_go
            read -p "Install Node.js? (y/n): " node && [[ $node == "y" ]] && install_nodejs && install_npm_packages
            read -p "Install Python packages? (y/n): " python && [[ $python == "y" ]] && install_python_packages
            read -p "Install Rust? (y/n): " rust && [[ $rust == "y" ]] && install_rust
            read -p "Install additional tools? (y/n): " tools && [[ $tools == "y" ]] && install_additional_tools
            ;;
        *)
            echo "Invalid choice. Exiting."
            exit 1
            ;;
    esac
    
    setup_directories
    setup_aliases
    show_completion
}

# Run main function
main "$@"