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

### Handler Architecture

Handlers are organized by domain with clear separation:
- **VPS handlers** (`internal/handlers/vps/`) - Server lifecycle, configuration, metadata
- **Application handlers** (`internal/handlers/applications/`) - App deployment and management
- **Core handlers** (`internal/handlers/`) - Auth, DNS, terminal, pages

Each handler follows dependency injection pattern through the `RouteConfig` struct in `cmd/xanthus/main.go:51-61`.

### Development Patterns

**Configuration Management:**
- YAML-based application configurations in `configs/`
- Template-driven server setup in `internal/templates/`
- Environment-based feature flags for testing

**Error Handling:**
- Structured error responses for API endpoints
- User-friendly error messages in web interface
- Comprehensive logging for debugging

**Security:**
- Middleware-based authentication
- Trusted proxy configuration
- Input validation and sanitization

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