# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running
- `make dev` - Start development server with CSS compilation
- `make build` - Build production binary (includes asset compilation)
- `make build-windows` - Cross-compile for Windows 64-bit
- `go run cmd/xanthus/main.go` - Run directly without Make

### Asset Management
- `make css` - Compile CSS for production (minified)
- `make css-watch` - Watch and recompile CSS during development
- `make assets` - Build all frontend assets (CSS + JS vendors)
- `npm run build-assets` - Alternative asset build command

### Testing Strategy

**Quick Tests (< 5 minutes):**
- `make test` - Unit and integration tests (excludes E2E)
- `make test-unit` - Unit tests only
- `make test-integration` - Integration tests only

**Comprehensive Testing:**
- `make test-e2e` - End-to-end tests in mock mode
- `make test-e2e-live` - E2E tests with real Hetzner/Cloudflare APIs (costs money)
- `make test-everything` - All tests including E2E

**Specialized Test Suites:**
- `make test-e2e-vps` - VPS lifecycle tests
- `make test-e2e-ssl` - SSL certificate management
- `make test-e2e-apps` - Application deployment
- `make test-e2e-ui` - UI integration tests
- `make test-e2e-perf` - Performance tests
- `make test-e2e-security` - Security tests
- `make test-e2e-dr` - Disaster recovery tests

**Test Environment Variables:**
- `E2E_TEST_MODE=mock|live` - Test mode (default: mock)
- `TEST_HETZNER_API_KEY` - For live Hetzner API tests
- `TEST_CLOUDFLARE_TOKEN` - For live Cloudflare API tests
- `TEST_CLOUDFLARE_ACCOUNT_ID` - Cloudflare account
- `TEST_DOMAIN` - Test domain (default: test.xanthus.local)

### Code Quality
- `make lint` - Format and vet Go code
- `make test-coverage` - Generate coverage reports (coverage.html)
- `make clean` - Remove build artifacts and test files

## Architecture Overview

Xanthus is a **web-based infrastructure management platform** for deploying and managing applications on Hetzner VPS instances with Cloudflare DNS integration. It's built as a full-stack Go application using Gin web framework with HTMX frontend.

### Core Components

**Backend (Go):**
- **Gin** web framework with middleware-based architecture
- **Handler-Service-Model** pattern with domain separation
- **Structured testing** with unit, integration, and E2E tiers
- **Template-driven** HTML rendering with custom functions

**Frontend Stack:**
- **HTMX** for dynamic interactions without complex JavaScript
- **Alpine.js** for client-side reactivity
- **Tailwind CSS** for styling
- **SweetAlert2** for notifications

### Directory Structure

```
cmd/xanthus/           # Application entry point
internal/              # Private application code
├── handlers/          # HTTP handlers by domain
│   ├── applications/  # App deployment handlers
│   └── vps/          # VPS management handlers
├── middleware/        # Auth, logging, etc.
├── models/           # Data structures
├── router/           # Route configuration
├── services/         # Business logic
├── templates/        # Configuration templates
└── utils/            # Shared utilities
web/                  # Frontend assets
├── static/           # CSS, JS, images
└── templates/        # HTML templates
tests/                # Comprehensive test suite
├── unit/             # Component tests
├── integration/      # Cross-component tests
└── integration/e2e/  # End-to-end scenarios
```

### Key Integrations

- **Hetzner Cloud** - VPS provisioning and management
- **Cloudflare** - DNS and SSL certificate management
- **K3s** - Kubernetes orchestration on provisioned VPS
- **Helm** - Application deployment via charts
- **SSH** - Server configuration and management

### Application Deployment Architecture

Xanthus uses a **configuration-driven application deployment system** that supports multiple application types through a unified pipeline:

**Configuration Structure:**
```
configs/applications/           # Application definitions
├── code-server.yaml           # VS Code in browser
├── argocd.yaml               # GitOps CD tool
└── [future-apps].yaml        # Extensible for new applications

internal/templates/applications/ # Helm values templates
├── code-server.yaml           # Code-server deployment template
├── argocd.yaml               # ArgoCD deployment template
└── [future-apps].yaml        # Templates for new applications
```

**Deployment Flow (All Applications):**
1. **Load Configuration** - Read app definition from `configs/applications/`
2. **Version Resolution** - Fetch latest version (GitHub API, Helm repos, etc.)
3. **Repository Setup** - Clone GitHub repos or add Helm repositories
4. **Values Generation** - Process templates with placeholder substitution
5. **Helm Deployment** - Deploy using generated values and chart configuration

**Supported Application Types:**

| Application | Repository Type | Version Source | Special Features |
|------------|----------------|----------------|-----------------|
| **Code-Server** | GitHub clone | GitHub API | VS Code settings, init containers |
| **ArgoCD** | Helm repository | Helm API* | RBAC, multi-component deployment |

*Note: Helm version fetching not yet implemented, defaults to "latest"

**Key Features:**
- **Template-driven deployment** - No hardcoded application logic
- **Dynamic chart handling** - Supports both GitHub and Helm repositories
- **Placeholder substitution** - `{{VERSION}}`, `{{SUBDOMAIN}}`, `{{DOMAIN}}`, etc.
- **Extensible architecture** - Add new applications via YAML configuration
- **Unified pipeline** - Same deployment flow for all application types

### Application Namespace Structure

Xanthus uses a **type-based namespace organization** for clean resource management:

**Namespace Design:**
- **All code-server applications** deploy to the `code-server` namespace
- **All ArgoCD applications** deploy to the `argocd` namespace
- **Future application types** will follow the same pattern (e.g., `grafana`, `jenkins`)

**Benefits:**
- ✅ **Clean organization**: Applications grouped by type instead of individual namespaces
- ✅ **Better resource management**: Easier to monitor and manage applications by type
- ✅ **Simplified operations**: Consistent namespace structure across deployments
- ✅ **Reduced namespace proliferation**: No more one-namespace-per-application

**Examples:**
```bash
# All code-server instances in one namespace
kubectl get pods -n code-server
# my-codeserver-app-123
# dev-codeserver-app-456
# test-codeserver-app-789

# All ArgoCD instances in one namespace  
kubectl get pods -n argocd
# prod-argocd-app-111
# staging-argocd-app-222
```

### Handler Architecture

Handlers are organized by domain with clear separation:
- **VPS handlers** (`internal/handlers/vps/`) - Server lifecycle, configuration, metadata
- **Application handlers** (`internal/handlers/applications/`) - App deployment and management
- **Core handlers** (`internal/handlers/`) - Auth, DNS, terminal, pages

Each handler follows dependency injection pattern through the `RouteConfig` struct in `cmd/xanthus/main.go:51-61`.

### Development Patterns

**Configuration Management:**
- YAML-based application configurations in `configs/applications/`
- Template-driven Helm values in `internal/templates/applications/`
- Template-driven server setup in `internal/templates/`
- Environment-based feature flags for testing

**Application Development:**
- New applications added via YAML configuration (no code changes required)
- Template-based deployment with placeholder substitution
- Unified deployment pipeline for all application types
- Version management through GitHub API or Helm repositories

**Password Management:**
- **Intelligent password retrieval** - Attempts KV store first, falls back to VPS if not found
- **Automatic password caching** - Stores retrieved passwords in KV for faster future access
- **Multi-application support** - Works with both code-server and ArgoCD applications
- **Robust error handling** - Gracefully handles missing passwords and connection issues

**Error Handling:**
- Structured error responses for API endpoints
- User-friendly error messages in web interface
- Comprehensive logging for debugging

**Security:**
- Middleware-based authentication
- Trusted proxy configuration
- Input validation and sanitization

## UI Features

### Auto-Refresh System

The applications page includes an **intelligent auto-refresh system** for real-time status monitoring:

**Auto-Refresh Features:**
- **Enabled by default** with 30-second intervals for optimal balance between freshness and performance
- **Visual countdown timer** showing time until next refresh ("Next: 29s")
- **Smart visibility detection** - automatically pauses when tab is hidden, resumes when visible
- **Concurrent request protection** - prevents multiple simultaneous refresh requests
- **Graceful error handling** - handles network issues and authentication failures

**Visual Indicators:**
- **Toggle button**: Shows "Auto-refresh ON/OFF" status
- **Pulsing green dot**: Indicates active auto-refresh
- **Live countdown**: Real-time countdown to next refresh
- **Status badges**: Color-coded application status indicators

**Status Color Mapping:**
- 🟢 **Running/deployed** - Green badge (application is healthy and accessible)
- 🔵 **Deploying/Creating** - Blue badge (deployment in progress)
- 🟡 **pending** - Yellow badge (waiting for resources or dependencies)
- 🔴 **Failed/failed** - Red badge (deployment or runtime failure)
- ⚪ **Not Deployed** - Gray badge (application not found or removed)

**Performance Optimizations:**
- **Background refresh** - Updates data without loading spinners during auto-refresh
- **Network-aware** - Automatically stops on repeated network failures
- **Resource-efficient** - 30-second intervals prevent server overload
- **Page focus detection** - Reduces unnecessary requests when user is away

**Usage:**
```javascript
// Auto-refresh controls in applications.html
toggleAutoRefresh()     // Enable/disable auto-refresh
refreshApplications()   // Manual refresh with loading indicator
refreshApplicationsQuietly() // Background refresh without UI feedback
```

## Testing Philosophy

The codebase uses a **three-tier testing approach**:

1. **Unit tests** - Test individual components in isolation
2. **Integration tests** - Test component interactions without external dependencies
3. **End-to-end tests** - Test complete workflows with mock or live external services

**Important:** E2E tests in live mode create real infrastructure resources and incur costs. Always use mock mode for development and CI/CD.

## Common Development Workflows

**Starting Development:**
```bash
make dev  # Starts server with CSS watching
```

**Before Committing:**
```bash
make lint         # Format and check code
make test         # Run fast test suite
make test-coverage # Verify coverage
```

**Feature Testing:**
```bash
make test-e2e     # Test with mocked external services
```

**Production Build:**
```bash
make build        # Creates bin/xanthus executable
```

**Adding New Applications:**
1. Create YAML configuration in `configs/applications/new-app.yaml`
2. Create Helm values template in `internal/templates/applications/new-app.yaml`
3. Application automatically available through the unified deployment pipeline
4. No code changes required - configuration-driven architecture

## Vps investigation

You can ssh to the vps at any time to investigate issues by using the @xanthus-key.pem and ssh -i xanthus-key.pem root@91.99.236.70

You can login to the app with the CLOUDFARE_API_TOKEN found at .env

Always use instructions at @logic/curl-commands.md to interact with the app, check changes and iterate.

## Important!!!

All *.go files shouldn't exceed the 500 lines limit. If you encounter such a case please tell me and suggest a refactor plan.