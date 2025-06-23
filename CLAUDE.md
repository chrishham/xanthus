# Xanthus - K3s Deployment Tool

## Project Overview
Xanthus is a Go web application that helps developers deploy their applications to a K3s cluster on Hetzner VPS instances. The app uses Gin for the web framework, HTMX for frontend interactions, Tailwind CSS for styling, and Alpine.js for reactive components.

## Architecture
- **Backend**: Go with Gin web framework
- **Frontend**: HTMX + Tailwind CSS + Alpine.js
- **Deployment**: Single executable for desktop use
- **Storage**: Cloudflare KV for settings persistence
- **Infrastructure**: Hetzner VPS with K3s cluster

## Key Features
1. **Cloudflare DNS Management**: Automatic SSL certificate provisioning and DNS configuration
2. **Hetzner VPS Provisioning**: Automated Ubuntu VPS setup with essential software
3. **Secure Settings Storage**: All configuration stored in Cloudflare KV (API keys, server IPs, etc.)
4. **Single Sign-On**: Only Cloudflare API key required for authentication
5. **SSL Automation**: Complete SSL/TLS configuration for domains with certificate storage in KV

## Initial Setup Actions

### 1. Project Structure Setup
```bash
mkdir -p cmd/xanthus
mkdir -p internal/{api,config,handlers,models,services}
mkdir -p web/{static,templates}
mkdir -p scripts
```

### 2. Initialize Go Module
```bash
go mod init xanthus
```

### 3. Install Dependencies
```bash
# Core dependencies
go get github.com/gin-gonic/gin
go get github.com/spf13/viper
go get github.com/cloudflare/cloudflare-go

# Hetzner Cloud API
go get github.com/hetznercloud/hcloud-go/hcloud

# Additional utilities
go get github.com/joho/godotenv
```

### 4. Frontend Dependencies
```bash
# Install Tailwind CSS for production build
npm install

# Frontend CDN dependencies (loaded in templates):
# - HTMX: https://unpkg.com/htmx.org@1.9.10/dist/htmx.min.js
# - Alpine.js: https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js
```

### 5. Configuration Files
- `.env.example` - Environment variables template
- `config.yaml` - Application configuration
- `Dockerfile` - For containerized deployment (optional)
- `Makefile` - Build and development commands

### 6. Core Components to Implement
1. **Main Application** (`cmd/xanthus/main.go`)
2. **Configuration Management** (`internal/config/`)
3. **Cloudflare Integration** (`internal/services/cloudflare.go`) ✅ **IMPLEMENTED**
4. **KV Storage Service** (`internal/services/kv.go`) ✅ **IMPLEMENTED**
5. **Hetzner Cloud Integration** (`internal/services/hetzner.go`)
6. **Web Handlers** (`internal/handlers/`)
7. **API Routes** (`internal/api/`)
8. **Frontend Templates** (`web/templates/`) ✅ **DNS CONFIG UPDATED**
9. **Static Assets** (`web/static/`)

### 7. Development Workflow
```bash
# Run in development mode (builds CSS and starts server)
make dev

# Build for production (builds optimized CSS and Go binary)
make build

# CSS Development Commands
make css           # Build CSS for production (minified)
make css-watch     # Watch CSS changes during development

# Testing and Code Quality
make test          # Run Go tests
make lint          # Format and vet Go code

# Cleanup
make clean         # Remove build artifacts
```

### 8. Tailwind CSS Setup
The project uses Tailwind CSS with a production build process:
- **Source**: `web/static/css/input.css` (Tailwind directives)
- **Built**: `web/static/css/output.css` (optimized CSS served to browser)
- **Config**: `tailwind.config.js` (scans templates for used classes)
- **Benefits**: Only includes used CSS (~10KB vs 3MB CDN), offline support, faster loading

### 9. Security Considerations
- Secure API key storage in Cloudflare KV
- Input validation for all user inputs
- Rate limiting for API endpoints
- HTTPS enforcement
- Proper error handling without exposing sensitive information

## Next Steps
1. Set up the basic project structure ✅ **COMPLETED**
2. Implement core configuration management
3. Create basic web server with Gin ✅ **COMPLETED**
4. Integrate Cloudflare KV for settings storage ✅ **COMPLETED**
5. Implement Hetzner VPS provisioning
6. Add Cloudflare DNS management ✅ **COMPLETED**
7. Create the frontend interface ✅ **DNS CONFIG COMPLETED**
8. Add deployment automation for K3s

## SSL Automation Implementation ✅ **COMPLETED**

### Features Implemented
- **Complete SSL/TLS Configuration**: Automated setup including:
  - SSL mode set to Full (strict)
  - Origin server certificate generation with 15-year validity
  - Cloudflare root CA certificate appending
  - Always Use HTTPS enforcement
  - www to non-www redirect page rules
- **Certificate Storage**: All certificates and private keys stored securely in Cloudflare KV
- **Web Interface**: Interactive domain management with SweetAlert2 modals
- **API Endpoints**: 
  - `POST /dns/configure` - Configure SSL for a domain
  - `POST /dns/remove` - Remove domain configuration
- **Status Tracking**: Visual indication of managed vs unmanaged domains

### Services Architecture
- **CloudflareService** (`internal/services/cloudflare.go`): Handles all Cloudflare API operations
- **KVService** (`internal/services/kv.go`): Manages secure storage and retrieval of configurations
- **Domain SSL Config**: Structured storage of SSL configuration with metadata

### Usage
1. Navigate to `/dns` to view domains
2. Click "Add to Xanthus" to configure SSL automation
3. Click "View Config" to see current configuration status
4. Click "Remove" to remove from Xanthus management

### Certificate Storage Format
```json
{
  "domain": "example.com",
  "zone_id": "cloudflare_zone_id",
  "certificate_id": "origin_cert_id",
  "certificate": "full_certificate_chain_with_root_ca",
  "private_key": "rsa_private_key",
  "configured_at": "2024-06-23T10:30:00Z",
  "ssl_mode": "strict",
  "always_use_https": true,
  "page_rule_created": true
}
```