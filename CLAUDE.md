# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 📚 Architecture Documentation

For detailed architecture information, see the following documentation:

- **[Handlers Architecture](internal/handlers/README.md)** - HTTP request processing, domain organization, dependency injection
- **[Services Architecture](internal/services/README.md)** - Business logic, external integrations, deployment orchestration  
- **[Models Architecture](internal/models/README.md)** - Data structures, validation, serialization patterns
- **[Configuration System](configs/README.md)** - YAML-driven app definitions, configuration-driven deployment
- **[Charts Architecture](charts/README.md)** - Local Helm charts, Kubernetes manifests, deployment templates
- **[Testing Strategy](tests/README.md)** - Three-tier testing, mock vs live modes, test organization
- **[Web Templates](web/templates/README.md)** - HTMX integration, Alpine.js components, UI patterns

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

## 🏗️ High-Level Architecture

Xanthus is a **configuration-driven infrastructure management platform** for deploying applications on cloud VPS instances with automated DNS/SSL management.

### Core Architecture Patterns
- **Handler-Service-Model (HSM)** - Clean separation of concerns
- **Configuration-Driven Deployment** - No code changes for new apps
- **Type-Based Namespaces** - Organized resource management
- **Three-Tier Testing** - Unit, integration, and E2E with mock/live modes

### Key Integrations
- **Hetzner Cloud & Oracle Cloud** - VPS provisioning
- **Cloudflare** - DNS and SSL certificate management  
- **K3s** - Kubernetes orchestration on VPS
- **Helm** - Application deployment via local charts
- **HTMX + Alpine.js** - Dynamic UI without complex JavaScript

### Quick Reference
- **Entry Point**: `cmd/xanthus/main.go:51-61` (RouteConfig dependency injection)
- **Application Creation**: `internal/services/application_service_core.go:124`
- **Deployment Pipeline**: `internal/services/application_service_deployment.go:14`
- **Handler Patterns**: See [Handlers README](internal/handlers/README.md)
- **Service Orchestration**: See [Services README](internal/services/README.md)

## 🚀 Key Features & Workflows

### Configuration-Driven Deployment
```
YAML Config → Template Processing → Helm Values → K8s Deployment
```
- **Add new apps** without code changes - see [Configuration System](configs/README.md)
- **Local Helm charts** for fast, reliable deployments - see [Charts Architecture](charts/README.md)
- **Template substitution** - `{{VERSION}}`, `{{SUBDOMAIN}}`, `{{DOMAIN}}`

### Application Namespace Strategy
- **Type-based namespaces** - All `code-server` apps in `code-server` namespace
- **Clean resource management** - Easier monitoring and operations
- **Consistent structure** - Predictable deployment patterns

### Auto-Refresh UI System
- **30-second intervals** with visual countdown timer
- **Smart pause/resume** when tab becomes hidden/visible
- **Background updates** without loading spinners
- **Network-aware** error handling

### Password Management
- **Intelligent retrieval** - KV store first, VPS fallback
- **Automatic caching** - Faster subsequent access
- **Encrypted storage** - Token-based encryption

## 🛠️ Common Development Workflows

### Quick Development Tasks
```bash
make dev              # Start development with CSS watching
make test            # Quick tests (< 5 minutes)
make lint            # Format and validate code
```

### Adding New Applications (No Code Required!)
1. **Create config**: `configs/applications/new-app.yaml` - see [Configuration System](configs/README.md)
2. **Create template**: `internal/templates/applications/new-app.yaml`
3. **Test deployment**: App automatically available in catalog

### Testing Strategy
```bash
make test-e2e        # E2E tests (mock mode, free)
make test-e2e-live   # E2E tests (real APIs, costs money)
make test-everything # Full test suite
```
See [Testing Strategy](tests/README.md) for comprehensive testing approach.

### Code Organization Guidelines
- **500-line limit** for Go files - refactor if exceeded
- **Handler-Service-Model** pattern - see [Handlers](internal/handlers/README.md) and [Services](internal/services/README.md)
- **Domain separation** - VPS handlers separate from application handlers

## 🔧 Debugging & Investigation

### VPS SSH Connection
**Dynamic connection** (don't hardcode IPs):
```bash
# 1. Get VPS details via API
curl -X GET "http://localhost:8081/vps" -b cookies.txt

# 2. Connect with provider-specific user
ssh -i xanthus-key.pem root@{hetzner_ip}      # Hetzner (root)
ssh -i xanthus-key.pem ubuntu@{oracle_ip}     # Oracle (ubuntu)
```

### Authentication & API Testing
- **Login**: Use `CLOUDFARE_API_TOKEN` from `.env`
- **API Reference**: See `logic/curl-commands.md` for complete API examples
- **Code-Server Flow**: See `logic/code-server-deployment-flow.md` for deployment details

### Performance Monitoring
- **Coverage**: `make test-coverage` → `coverage.html`
- **Auto-refresh**: 30-second intervals with smart pause/resume
- **Resource limits**: See [Charts Architecture](charts/README.md) for container limits

---

## 📖 Additional Resources

- **[Complete API Examples](logic/curl-commands.md)** - curl commands for all endpoints
- **[Code-Server Deployment Flow](logic/code-server-deployment-flow.md)** - Step-by-step deployment process
- **[Performance Findings](logic/performance-optimization-findings.md)** - Optimization insights

---

**⚠️ Important**: All Go files must stay under 500 lines. If exceeded, suggest refactoring plan.