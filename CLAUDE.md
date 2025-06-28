# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Xanthus is a K3s deployment tool built with Go (Gin framework) and HTMX/Alpine.js frontend. It provides a web interface for managing VPS servers on Hetzner Cloud, configuring DNS through Cloudflare, and deploying applications via Helm to K3s clusters.

## Development Commands

### Building and Running
- `make dev` - Run development server with CSS build
- `make build` - Build production binary (creates `bin/xanthus`)
- `go run cmd/xanthus/main.go` - Run directly without make

### Assets and Styling
- `make css` - Build CSS for production (minified)
- `make css-watch` - Watch CSS changes during development
- `make assets` - Build all assets (CSS + JS vendor files)

### Testing and Code Quality
- `make test` - Run all structured tests (`./tests/...`)
- `make test-unit` - Run unit tests only (`./tests/unit/...`)
- `make test-integration` - Run integration tests only (`./tests/integration/...`)
- `make test-coverage` - Run tests with coverage report (generates `coverage.html`)
- `make test-all` - Run ALL tests including any legacy tests (`./...`)
- `make lint` - Format and vet Go code (`go fmt` + `go vet`)

### Cleaning
- `make clean` - Remove build artifacts and generated assets

## Architecture

### Handler-Based Architecture
The application uses a clean handler pattern with domain separation:

- **`internal/handlers/`** - Business logic organized by domain
  - `auth.go` - Authentication and health endpoints
  - `dns.go` - Cloudflare DNS management
  - `vps.go` - Hetzner VPS lifecycle management
  - `applications.go` - Helm application deployment
  - `terminal.go` - SSH terminal sessions
  - `pages.go` - Static page rendering

### Route Organization
Routes are grouped by domain in `internal/router/routes.go`:
- Public routes (login, health)
- Protected routes (main app functionality)
- API routes (future extensibility)

### Service Layer
- **`internal/services/`** - External API integrations
  - `hetzner.go` - Hetzner Cloud API
  - `cloudflare.go` - Cloudflare API
  - `helm.go` - Helm chart deployment
  - `ssh.go` - SSH connection management
  - `kv.go` - Cloudflare KV storage

### Utilities and Models
- **`internal/utils/`** - Reusable helper functions
  - `responses.go` - Standardized JSON responses
  - `cloudflare.go` - Cloudflare API utilities
  - `hetzner.go` - Hetzner API utilities
  - `crypto.go` - Encryption/decryption
  - `server.go` - Server utilities (port finding)

- **`internal/models/types.go`** - All data structures (Cloudflare, Hetzner, Application types)

### Frontend Structure
- **`web/templates/`** - HTML templates with HTMX integration
- **`web/static/`** - Static assets (CSS built with Tailwind, JS vendor files)

## Key Integration Points

### Authentication Flow
Authentication uses token-based validation with Cloudflare KV storage. The `middleware.AuthMiddleware()` protects all routes except login/health endpoints.

### VPS Lifecycle
1. Hetzner API validation and server creation
2. SSH key injection and cloud-init configuration
3. K3s cluster installation via SSH
4. DNS configuration through Cloudflare
5. Application deployment via Helm

### Helm Deployment System
Real K3s deployments with automatic ingress configuration, SSL termination, and complete application lifecycle management (install/upgrade/uninstall).

### Application Password Management
For code-server applications, passwords are encrypted and stored in Cloudflare KV storage with full lifecycle management:
- **Password Retrieval**: `GET /applications/:id/password` - retrieves current password for code-server apps
- **Password Change**: `POST /applications/:id/password` - updates password with minimum 8 characters
- **Storage**: Passwords encrypted using `utils.EncryptData()` with user's Cloudflare token as key
- **UI Features**: "Get Password" and "Change Password" buttons with modal dialogs and copy functionality
- **Security**: Only available for code-server applications, requires authentication

## Dependencies

### Go Modules
- `github.com/gin-gonic/gin` - Web framework
- `golang.org/x/crypto` - Cryptographic functions

### Frontend Assets
- `htmx.org` - Dynamic HTML interactions
- `alpinejs` - Client-side state management
- `sweetalert2` - User notifications
- `tailwindcss` - CSS framework

## Important Notes
- To ssh to the VPS you can use the @xanthus-ssh-key.pem with ssh -i xanthus-ssh-key.pem root@188.245.79.245
- Code Server installation info is at @plans/code-server.md
- Server automatically finds available port at startup
- All responses use standardized helpers from `utils/responses.go`
- Encryption keys and API tokens stored securely in Cloudflare KV
- SSH sessions managed with proper cleanup and security
- The codebase was recently refactored from a 3,120-line main.go to clean architecture (98% reduction)