# Xanthus

> **Self-hosted infrastructure management platform** for deploying applications on cloud VPS instances with automated DNS/SSL management.

Xanthus is a configuration-driven infrastructure management platform that simplifies the deployment of applications on cloud VPS instances. It provides automated DNS and SSL certificate management, making it easy to deploy and manage applications without complex setup procedures.

## 🚀 Features

- **Configuration-Driven Deployment** - Deploy applications using simple YAML configurations
- **Multi-Cloud VPS Support** - Works with Hetzner Cloud and Oracle Cloud
- **Automated DNS/SSL Management** - Seamless integration with Cloudflare for DNS and SSL certificates
- **Kubernetes Orchestration** - Uses K3s for reliable application deployment
- **Self-Updating Platform** - Manage Xanthus versions through the web interface
- **Application Catalog** - Pre-configured applications ready for one-click deployment
- **Web-Based Management** - Intuitive UI for managing infrastructure and applications

## 📦 Installation

### Option 1: Download Pre-built Binaries

Download the latest release for your platform from the [GitHub Releases page](https://github.com/chrishham/xanthus/releases):

- **Linux AMD64**: `xanthus-linux-amd64.tar.gz`
- **Linux ARM64**: `xanthus-linux-arm64.tar.gz`
- **Windows**: `xanthus-windows-amd64.zip`
- **macOS Intel**: `xanthus-macos-intel.tar.gz`
- **macOS Apple Silicon**: `xanthus-macos-arm64.tar.gz`

### Option 2: Docker

```bash
docker run -d \
  --name xanthus \
  -p 8081:8081 \
  -v xanthus-data:/data \
  ghcr.io/chrishham/xanthus:latest
```

### Option 3: Build from Source

```bash
git clone https://github.com/chrishham/xanthus.git
cd xanthus
make build
./bin/xanthus
```

## 🛠️ Quick Start

1. **Start Xanthus**:
   ```bash
   ./xanthus
   ```

2. **Access Web Interface**:
   Open http://localhost:8081 in your browser

3. **Configure Cloud Providers**:
   - Add your Hetzner Cloud API key
   - Add your Cloudflare API token
   - Configure your domain settings

4. **Deploy Your First Application**:
   - Choose from the application catalog
   - Select your target VPS or create a new one
   - Deploy with one click

## 📋 Development

### Prerequisites

- Go 1.24+
- Node.js 18+
- Make

### Development Commands

```bash
# Start development server
make dev

# Run tests
make test

# Run linter
make lint

# Build for production
make build

# Build for all platforms
make build-all
```

### Testing

```bash
# Quick tests (unit + integration)
make test

# End-to-end tests (mock mode)
make test-e2e

# All tests including E2E
make test-everything
```

## 🏗️ Architecture

Xanthus follows a clean **Handler-Service-Model (HSM)** architecture:

- **Handlers** - HTTP request processing and routing
- **Services** - Business logic and external service integration
- **Models** - Data structures and validation

### Key Components

- **Application Catalog** - YAML-based application definitions
- **VPS Management** - Multi-cloud VPS provisioning and management
- **DNS/SSL Automation** - Cloudflare integration for domain management
- **Kubernetes Integration** - K3s deployment with Helm charts
- **Version Management** - Self-updating capabilities with rollback support

For detailed architecture documentation, see:
- [Handlers Architecture](internal/handlers/README.md)
- [Services Architecture](internal/services/README.md)
- [Models Architecture](internal/models/README.md)

## 🚀 Creating Releases

### For Maintainers

Create a new release using the automated release system:

```bash
# Create a new release (replace with your version)
make release VERSION=v1.0.0
```

This command will:
1. **Run tests** to ensure code quality
2. **Create a Git tag** with the specified version
3. **Push to GitHub** to trigger the automated release workflow
4. **GitHub Actions will automatically**:
   - Build multi-architecture Docker images (AMD64 + ARM64)
   - Create cross-platform binaries (Windows, macOS, Linux)
   - Publish to GitHub Container Registry
   - Create GitHub Release with downloadable assets

### Re-releasing (Fixing Issues)

If you need to fix issues in an existing release without bumping the version:

```bash
# Fix your issues first
git add .
git commit -m "fix: critical bug in v1.0.0"

# Re-release the same version with fixes
make re-release VERSION=v1.0.0
```

The `re-release` command will:
1. **Run tests** to ensure the fixes work
2. **Force-update the existing Git tag** to point to the current commit
3. **Trigger GitHub Actions** to rebuild and replace all artifacts
4. **Update Docker images and binaries** without changing version numbers

This is perfect for:
- 🐛 **Critical bug fixes**
- 🔒 **Security patches** 
- 📝 **Documentation updates**
- 🏗️ **Build improvements**

### Version Format

- **Stable releases**: `v1.0.0`, `v1.1.0`, `v2.0.0`
- **Pre-releases**: `v1.0.0-rc.1`, `v1.0.0-beta.1`
- **Patch releases**: `v1.0.1`, `v1.0.2`

### Release Artifacts

After a successful release, artifacts will be available at:
- **Docker Images**: `ghcr.io/chrishham/xanthus:v1.0.0`
- **Binaries**: [GitHub Releases page](https://github.com/chrishham/xanthus/releases)
- **Container Registry**: Multi-architecture images automatically built

### Release Strategy

- **Development builds**: Only run CI tests, no artifacts created
- **Tagged releases**: Full build pipeline with multi-platform artifacts
- **Automated quality gates**: Tests must pass before release creation
- **Security scanning**: All Docker images scanned for vulnerabilities

### Troubleshooting Releases

**Q: Release workflow failed with "tag already exists"**
```bash
# Use re-release instead of release for existing versions
make re-release VERSION=v1.0.0
```

**Q: Docker build fails with "tailwindcss: not found"**
- This should be fixed in recent versions. If you encounter this, ensure your Dockerfile includes devDependencies during build.

**Q: How to cancel a failed release?**
```bash
# Delete the tag locally and remotely
git tag -d v1.0.0
git push origin --delete v1.0.0
```

**Q: Release assets are missing or incomplete**
- Check GitHub Actions workflow logs for errors
- Re-run the workflow from GitHub Actions UI, or use `make re-release`

## 📚 Documentation

- **[Architecture Overview](CLAUDE.md)** - Complete development guide
- **[Configuration System](configs/README.md)** - Application configuration
- **[Testing Strategy](tests/README.md)** - Testing approach and commands
- **[API Documentation](logic/curl-commands.md)** - Complete API reference

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite: `make test`
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🔗 Links

- [GitHub Repository](https://github.com/chrishham/xanthus)
- [Container Registry](https://ghcr.io/chrishham/xanthus)
- [Documentation](CLAUDE.md)
- [Issues](https://github.com/chrishham/xanthus/issues)