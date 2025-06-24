# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Xanthus is a Go web application that helps developers deploy K3s clusters on Hetzner VPS instances with automated SSL configuration through Cloudflare. It's a desktop tool that provides a complete infrastructure automation solution.

## Architecture

- **Backend**: Go 1.24.4 using Gin web framework
- **Frontend**: HTMX + Tailwind CSS + Alpine.js (no SPA framework)
- **Storage**: Cloudflare KV for all persistent configuration
- **Infrastructure**: Hetzner Cloud VPS with automated K3s deployment
- **SSL**: Cloudflare Origin Server certificates with full SSL automation
- **Authentication**: Single Cloudflare API token for all operations

## Development Commands

```bash
# Development with CSS watching
make dev                    # Builds CSS and starts Go server
make css-watch             # Watch Tailwind CSS changes

# Production builds
make build                 # Build optimized binary with assets
make assets                # Build CSS and JS assets only
make css                   # Build production CSS only

# Code quality
make test                  # Run Go tests
make lint                  # Format and vet Go code

# Cleanup
make clean                 # Remove build artifacts
```

## Key Services Architecture

The application is structured around four core services in `internal/services/`:

### CloudflareService (`cloudflare.go`)
- DNS zone management and SSL certificate provisioning
- Origin Server certificate generation with 15-year validity
- SSL mode configuration (Full strict) and page rules
- Handles both domain management and certificate lifecycle

### KVService (`kv.go`) 
- Cloudflare KV namespace management and data persistence
- Encrypted storage for sensitive data (Hetzner API keys)
- SSL certificate and VPS configuration storage
- Key patterns: `domain:{domain}:ssl_config`, `vps:{id}:config`, `config:ssl:csr`

### HetznerService (`hetzner.go`)
- VPS provisioning with Ubuntu 24.04 and K3s installation
- SSH key management with smart reuse logic
- Cloud-init configuration for automated server setup
- Power management operations (start/stop/reboot)

### SSH Key Management (`ssh.go`)
- Single RSA key architecture for both SSL certificates and SSH access
- Content-based key matching to prevent duplicates
- Automatic key reuse across VPS instances
- Key naming pattern: `xanthus-key-{unix_timestamp}`

## Frontend Architecture

- **Templates**: Server-rendered HTML in `web/templates/`
- **Interactivity**: HTMX for AJAX interactions, Alpine.js for reactive components
- **Styling**: Tailwind CSS with production optimization (scans templates for used classes)
- **Assets**: JavaScript vendors copied from node_modules to `web/static/js/vendor/`

## Data Storage Patterns

All configuration stored in Cloudflare KV with structured key patterns:

- SSL configs: `domain:{domain}:ssl_config` 
- VPS configs: `vps:{server_id}:config`
- CSR data: `config:ssl:csr` (shared RSA key for all operations)
- Encrypted API keys: `config:hetzner:api_key`

## Development Workflow

1. **CSS Development**: Use `make css-watch` for live Tailwind compilation
2. **Backend Changes**: Use `make dev` to rebuild and restart server
3. **Testing**: Run `make test` and `make lint` before commits
4. **Production**: Use `make build` for optimized binary with minified assets

## Security Considerations

- All sensitive data encrypted with AES-256-GCM using Cloudflare token as key
- Single RSA key architecture reduces attack surface
- HTTPS enforcement through Cloudflare proxy
- Rate limiting and input validation on all endpoints
- No local storage of credentials or certificates

## Key Integration Points

- VPS creation includes automatic SSL certificate deployment to `/opt/xanthus/ssl/`
- K3s configured with Cloudflare Origin certificates for ingress
- SSH access uses same private key as SSL certificates
- VPS status updates are currently manual (automatic polling is a planned feature)