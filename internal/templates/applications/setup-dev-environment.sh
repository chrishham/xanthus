#!/bin/bash

#================================================================================#
#           IMPROVED DEVELOPMENT ENVIRONMENT SETUP SCRIPT                        #
#                                                                                #
# This script allows you to selectively install development tools, including     #
# Go, Node.js, Rust, Dagger, and command-line tools for Gemini and Claude.       #
#================================================================================#

set -e

# --- Configuration & Colors ---
C_RESET='\033[0m'
C_RED='\033[0;31m'
C_GREEN='\033[0;32m'
C_YELLOW='\033[0;33m'
C_BLUE='\033[0;34m'
C_CYAN='\033[0;36m'

# --- Utility Functions ---

# Print a formatted header
print_header() {
    echo -e "\n${C_BLUE}#=======================================================================${C_RESET}"
    echo -e "${C_CYAN}# $1${C_RESET}"
    echo -e "${C_BLUE}#=======================================================================${C_RESET}"
}

# Print a success message
print_success() {
    echo -e "${C_GREEN}✅ $1${C_RESET}"
}

# Print an info message
print_info() {
    echo -e "${C_YELLOW}🔧 $1${C_RESET}"
}

# Print an error message and exit
print_error() {
    echo -e "${C_RED}❌ ERROR: $1${C_RESET}" >&2
    exit 1
}

# Check if a command exists
command_exists() {
    command -v "$1" &> /dev/null
}

# --- Core Functions ---

# Function to check if running as root
check_root() {
    if [ "$EUID" -eq 0 ]; then
        print_error "This script should not be run as root! Please run as a regular user."
    fi
}

# Function to determine system architecture
get_arch() {
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64) ARCH="amd64" ;;
        aarch64 | arm64) ARCH="arm64" ;;
        *) print_error "Unsupported architecture: $ARCH" ;;
    esac
    echo "$ARCH"
}

# --- Installation Functions ---

install_system_packages() {
    print_header "Installing System Packages"
    print_info "Updating package list and installing essential tools..."
    sudo apt-get update
    sudo apt-get install -y build-essential ripgrep curl wget git vim nano \
        htop tree jq unzip zip python3 python3-pip fd-find bat docker.io
    print_success "System packages installed."
}

install_go() {
    if command_exists go; then
        print_success "Go is already installed: $(go version)"
        return
    fi
    print_header "Installing Go (Latest Stable)"
    local ARCH
    ARCH=$(get_arch)

    print_info "Fetching the latest Go version..."
    GO_VERSION=$(curl -s https://go.dev/dl/ | grep -oP 'go([0-9]+\.[0-9]+\.[0-9]+)\.linux-'"$ARCH"'\.tar\.gz' | head -n 1 | sed -E 's/go(.*)\.linux-.*\.tar\.gz/\1/')
    if [ -z "$GO_VERSION" ]; then
        print_error "Could not determine the latest Go version."
    fi
    
    local TARFILE="go${GO_VERSION}.linux-${ARCH}.tar.gz"
    local DOWNLOAD_URL="https://go.dev/dl/${TARFILE}"

    print_info "Downloading Go v${GO_VERSION} for ${ARCH}..."
    cd /tmp
    curl -LO "$DOWNLOAD_URL"

    print_info "Extracting Go to /usr/local..."
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "${TARFILE}"
    rm "${TARFILE}"

    if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
        print_info "Adding Go to your PATH in ~/.bashrc"
        cat >> ~/.bashrc << 'EOF'

# Go Environment Variables
export PATH="$PATH:/usr/local/go/bin"
export GOPATH="$HOME/go"
export PATH="$PATH:$GOPATH/bin"
EOF
    fi
    print_success "Go v${GO_VERSION} installed successfully."
}

install_nodejs() {
    # Source Volta if it exists to make node command available
    [ -s "$HOME/.volta/volta.sh" ] && source "$HOME/.volta/volta.sh"

    if command_exists node; then
        print_success "Node.js is already installed: $(node -v)"
        return
    fi

    print_header "Installing Node.js (via Volta)"
    print_info "Installing Volta (Node.js version manager)..."
    curl https://get.volta.sh | bash
    # Add Volta to PATH for the current session
    export VOLTA_HOME="$HOME/.volta"
    export PATH="$VOLTA_HOME/bin:$PATH"
    
    print_info "Installing Node.js LTS (currently 20.x)..."
    volta install node@20
    print_success "Node.js installed."

    # Ensure Volta is in bashrc
    if ! grep -q "VOLTA_HOME" ~/.bashrc; then
        echo -e '\n# Volta Environment\nexport VOLTA_HOME="$HOME/.volta"\nexport PATH="$VOLTA_HOME/bin:$PATH"' >> ~/.bashrc
    fi
}

install_rust() {
    if command_exists rustc; then
        print_success "Rust is already installed: $(rustc --version)"
        return
    fi
    print_header "Installing Rust"
    print_info "Installing rustup..."
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y

    # Add cargo to PATH for the current session
    export PATH="$HOME/.cargo/bin:$PATH"

    if ! grep -q 'source "$HOME/.cargo/env"' ~/.bashrc; then
         echo 'source "$HOME/.cargo/env"' >> ~/.bashrc
    fi
    print_success "Rust installed successfully."
}

install_npm_ai_clis() {
    print_header "Installing AI CLIs (Claude & Gemini)"
    
    # AI CLIs require Node.js, so install it if it's not present
    if ! command_exists npm; then
        print_info "Node.js/npm not found. Installing it first..."
        install_nodejs
    fi
    
    print_info "Installing Claude Code and Gemini CLI via npm..."
    npm install -g @anthropic-ai/claude-code
    npm install -g @google/gemini-cli
    
    print_success "AI CLIs installed."
}

install_python_packages() {
    print_header "Installing Python Development Tools"
    
    print_info "Installing pipx for managing Python applications..."
    sudo apt-get update
    sudo apt-get install -y pipx
    
    # Add pipx to the path and run it
    pipx ensurepath
    
    # Tools to be installed with pipx
    local python_tools=(
        "black"
        "flake8"
        "mypy"
        "pytest"
        "jupyter"
    )

    print_info "Installing Python tools with pipx..."
    for tool in "${python_tools[@]}"; do
        pipx install "$tool"
    done
    
    print_success "Python development tools installed via pipx."
    print_info "Libraries like 'requests' or 'pandas' should be installed in a project's virtual environment."
}

install_lazygit() {
    if command_exists lazygit; then
        print_success "lazygit is already installed."
        return
    fi
    print_header "Installing lazygit"
    print_info "Fetching the latest lazygit version..."
    LAZYGIT_VERSION=$(curl -s "https://api.github.com/repos/jesseduffield/lazygit/releases/latest" | grep -Po '"tag_name": "v\K[^"]*')
    if [ -z "$LAZYGIT_VERSION" ]; then
        print_error "Could not determine the latest lazygit version."
    fi
    
    local ARCH_RAW
    ARCH_RAW=$(uname -m)
    local ARCH_LAZYGIT
    case "$ARCH_RAW" in
        x86_64) ARCH_LAZYGIT="x86_64" ;;
        aarch64 | arm64) ARCH_LAZYGIT="arm64" ;;
        *) print_error "Unsupported architecture for lazygit: $ARCH_RAW" ;;
    esac

    local TARFILE="lazygit_${LAZYGIT_VERSION}_Linux_${ARCH_LAZYGIT}.tar.gz"
    print_info "Downloading lazygit v${LAZYGIT_VERSION}..."
    curl -Lo lazygit.tar.gz "https://github.com/jesseduffield/lazygit/releases/latest/download/${TARFILE}"
    
    tar xf lazygit.tar.gz lazygit
    sudo install lazygit /usr/local/bin
    rm lazygit.tar.gz lazygit
    print_success "lazygit installed successfully."
}

install_dagger() {
    if command_exists dagger; then
        print_success "Dagger is already installed: $(dagger version)"
        return
    fi
    print_header "Installing Dagger"
    print_info "Installing Dagger CI/CD Engine..."
    mkdir -p "$HOME/.local/bin"
    curl -fsSL https://dl.dagger.io/dagger/install.sh | BIN_DIR=$HOME/.local/bin sh

    if ! grep -q '$HOME/.local/bin' ~/.bashrc; then
        print_info "Adding $HOME/.local/bin to your PATH."
        echo -e '\n# Add local binaries to path\nexport PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
    fi
    print_success "Dagger installed successfully."
}

setup_environment() {
    print_header "Finalizing Environment Setup"
    
    print_info "Creating standard development directories..."
    mkdir -p ~/workspace ~/go

    print_info "Configuring useful aliases in ~/.bashrc..."
    if ! grep -q "# Custom Development Aliases" ~/.bashrc; then
      cat >> ~/.bashrc << 'EOF'

# Custom Development Aliases
alias ll='ls -alF'
alias ..='cd ..'
alias py='python3'
alias pip='pip3'
alias gst='git status'
alias gco='git checkout'
alias dps='docker ps'
EOF
    fi
    print_success "Environment setup is complete."
}

# --- Main Execution ---

main() {
    check_root
    
    cat << "EOF"

🚀 Development Environment Setup 🚀

Select the tools you want to install. You can choose multiple options.
Example: To install Go, Dagger, and AI Tools, enter: 2 5 7

  [1] Base System & Dev Tools (Recommended First Run)
  [2] Go (Latest)
  [3] Node.js (via Volta)
  [4] Rust
  [5] AI CLIs (Claude & Gemini via npm)
  [6] Python Dev Packages (flake8, pytest, etc.)
  [7] Dagger (CI/CD Engine)
  [8] lazygit (TUI for Git)
  [9] Setup Aliases & Dirs
  [10] ALL OF THE ABOVE

EOF

    read -p "Enter your choice(s): " -a choices
    
    if [ -z "${choices[*]}" ]; then
        print_error "No selection made. Exiting."
    fi

    for choice in "${choices[@]}"; do
        case $choice in
            1) install_system_packages ;;
            2) install_go ;;
            3) install_nodejs ;;
            4) install_rust ;;
            5) install_npm_ai_clis ;;
            6) install_python_packages ;;
            7) install_dagger ;;
            8) install_lazygit ;;
            9) setup_environment ;;
            10)
                install_system_packages; install_go; install_nodejs; install_rust;
                install_npm_ai_clis; install_python_packages; install_dagger;
                install_lazygit; setup_environment;
                break # No need to process other choices if 'all' is selected
                ;;
            *) echo -e "${C_YELLOW}⚠️ Ignoring invalid choice: $choice${C_RESET}" ;;
        esac
    done

    # --- Completion Summary ---
    print_header "Setup Summary"
    echo -e "${C_GREEN}🎉 All selected tasks are complete!${C_RESET}\n"
    echo "Verification:"
    command_exists go && echo -e "  - Go:        ${C_GREEN}Installed ($(go version))${C_RESET}"
    command_exists node && echo -e "  - Node.js:   ${C_GREEN}Installed ($(node -v))${C_RESET}"
    command_exists rustc && echo -e "  - Rust:      ${C_GREEN}Installed ($(rustc --version))${C_RESET}"
    command_exists dagger && echo -e "  - Dagger:    ${C_GREEN}Installed ($(dagger version))${C_RESET}"
    command_exists lazygit && echo -e "  - lazygit:   ${C_GREEN}Installed${C_RESET}"
    command_exists gemini && echo -e "  - Gemini CLI:  ${C_GREEN}Installed (npm package)${C_RESET}"
    command_exists claude-code && echo -e "  - Claude Code: ${C_GREEN}Installed (npm package)${C_RESET}"
    echo ""
    print_info "To apply all changes, please restart your terminal or run:"
    echo -e "${C_CYAN}  source ~/.bashrc${C_RESET}"
}

main "$@"
