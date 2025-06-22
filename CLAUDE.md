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
# Install via CDN or download locally
# - HTMX: https://unpkg.com/htmx.org@1.9.10/dist/htmx.min.js
# - Tailwind CSS: https://cdn.tailwindcss.com
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
3. **Cloudflare Integration** (`internal/services/cloudflare.go`)
4. **Hetzner Cloud Integration** (`internal/services/hetzner.go`)
5. **Web Handlers** (`internal/handlers/`)
6. **API Routes** (`internal/api/`)
7. **Frontend Templates** (`web/templates/`)
8. **Static Assets** (`web/static/`)

### 7. Development Workflow
```bash
# Run in development mode
make dev

# Build for production
make build

# Run tests
make test

# Lint code
make lint
```

### 8. Security Considerations
- Secure API key storage in Cloudflare KV
- Input validation for all user inputs
- Rate limiting for API endpoints
- HTTPS enforcement
- Proper error handling without exposing sensitive information

## Next Steps
1. Set up the basic project structure
2. Implement core configuration management
3. Create basic web server with Gin
4. Integrate Cloudflare KV for settings storage
5. Implement Hetzner VPS provisioning
6. Add Cloudflare DNS management
7. Create the frontend interface
8. Add deployment automation for K3s