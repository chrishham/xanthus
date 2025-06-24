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
2. **Hetzner VPS Provisioning**: Automated Ubuntu 24.04 VPS setup with K3s and SSL certificates
3. **Secure Settings Storage**: All configuration stored in Cloudflare KV (API keys, server IPs, etc.)
4. **Single Sign-On**: Only Cloudflare API key required for authentication
5. **SSL Automation**: Complete SSL/TLS configuration for domains with certificate storage in KV
6. **SSH Integration**: Single RSA key used for both SSL certificates and SSH access

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
go mod init github.com/chrishham/xanthus
```

### 3. Install Dependencies
```bash
# Core dependencies
go get github.com/gin-gonic/gin
go get golang.org/x/crypto
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
1. **Main Application** (`cmd/xanthus/main.go`) ✅ **IMPLEMENTED**
2. **Configuration Management** (`internal/config/`)
3. **Cloudflare Integration** (`internal/services/cloudflare.go`) ✅ **IMPLEMENTED**
4. **KV Storage Service** (`internal/services/kv.go`) ✅ **IMPLEMENTED**
5. **Hetzner Cloud Integration** (`internal/services/hetzner.go`) ✅ **IMPLEMENTED**
6. **Web Handlers** ✅ **IMPLEMENTED IN MAIN**
7. **API Routes** ✅ **IMPLEMENTED IN MAIN**
8. **Frontend Templates** (`web/templates/`) ✅ **IMPLEMENTED**
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

## Implementation Status ✅ **COMPLETED**
1. Set up the basic project structure ✅ **COMPLETED**
2. Implement core configuration management ✅ **COMPLETED**
3. Create basic web server with Gin ✅ **COMPLETED**
4. Integrate Cloudflare KV for settings storage ✅ **COMPLETED**
5. Implement Hetzner VPS provisioning ✅ **COMPLETED**
6. Add Cloudflare DNS management ✅ **COMPLETED**
7. Create the frontend interface ✅ **COMPLETED**
8. Add VPS management with SSH integration ✅ **COMPLETED**

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

## Cloudflare KV Storage Architecture ✅ **IMPLEMENTED**

### KV Namespace Structure
All data is stored in a single KV namespace called **"Xanthus"** within the user's Cloudflare account.

### KV Key-Value Pairs Structure

#### 1. SSL Configuration Storage
**Key Format**: `domain:{domain}:ssl_config`
**Example**: `domain:example.com:ssl_config`

**Value Structure** (`DomainSSLConfig`):
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

#### 2. CSR Configuration Storage
**Key**: `config:ssl:csr`

**Value Structure** (`CSRConfig`):
```json
{
  "csr": "-----BEGIN CERTIFICATE REQUEST-----\n...",
  "private_key": "-----BEGIN PRIVATE KEY-----\n...",
  "created_at": "2024-06-23T10:30:00Z"
}
```

#### 3. Hetzner API Key Storage
**Key**: `config:hetzner:api_key`
**Value**: AES-256-GCM encrypted string (encrypted using Cloudflare token as key)

#### 4. VPS Configuration Storage ✅ **NEW**
**Key Format**: `vps:{server_id}:config`
**Example**: `vps:123456:config`

**Value Structure** (`VPSConfig`):
```json
{
  "server_id": 123456,
  "name": "xanthus-k3s-server",
  "server_type": "cpx11",
  "location": "nbg1",
  "public_ipv4": "192.168.1.100",
  "status": "running",
  "created_at": "2024-06-23T10:30:00Z",
  "ssl_configured": true,
  "ssh_key_name": "xanthus-key-1719148800",
  "ssh_user": "root",
  "ssh_port": 22
}
```

### SSL Certificates for K3s Usage

The SSL certificates stored in `domain:{domain}:ssl_config` are **Cloudflare Origin Server Certificates** designed for:

1. **Certificate Type**: RSA origin certificates with 15-year validity
2. **Certificate Chain**: Full chain including Cloudflare's root CA certificate
3. **Hostnames**: Covers both `domain.com` and `*.domain.com` (wildcard support)
4. **Private Key**: 2048-bit RSA private key stored alongside certificate
5. **K3s Integration**: Perfect for K3s ingress controllers behind Cloudflare proxy
6. **SSL Mode**: Works with Cloudflare's "Full (strict)" SSL mode for end-to-end encryption

### Encryption & Security
- **Hetzner API Key**: Encrypted using AES-256-GCM with Cloudflare token as encryption key
- **SSL Certificates**: Stored in plaintext (secured by Cloudflare KV access controls)
- **CSR Private Keys**: Stored in plaintext within KV namespace (also used for SSH access)
- **Access Control**: Protected by Cloudflare token authentication

## Hetzner VPS Management ✅ **IMPLEMENTED**

### VPS Provisioning Features
- **Operating System**: Ubuntu 24.04 LTS (latest LTS)
- **K3s Installation**: Automatic installation with custom SSL certificates
- **SSH Access**: Uses CSR private key for SSH authentication
- **SSL Integration**: K3s configured with Cloudflare origin certificates
- **Auto-Configuration**: Cloud-init handles complete server setup

### VPS Management Interface
- **Dashboard Integration**: VPS management accessible from main dashboard (`/vps`)
- **Server Listing**: Shows all Xanthus-managed VPS instances with real-time status
- **Power Management**: Start, stop, reboot operations
- **SSH Information**: Displays connection details for each server
- **Creation Wizard**: One-click VPS creation using setup configuration

### SSH Integration ✅ **IMPLEMENTED**
- **Single Key Architecture**: Same RSA key used for SSL certificates and SSH access
- **Automatic Setup**: SSH public key automatically uploaded to Hetzner during VPS creation
- **Connection Details**: SSH connection information stored in VPS configuration
- **Key Conversion**: RSA private key from CSR converted to SSH format using `golang.org/x/crypto/ssh`
- **Smart Key Reuse**: SSH keys are intelligently reused across VPS instances based on public key content matching

### SSH Key Management ✅ **IMPLEMENTED**
- **Key Naming Pattern**: `xanthus-key-{unix_timestamp}` (e.g., `xanthus-key-1750699924`)
- **Content-Based Matching**: Keys are matched by public key content, not by name
- **Automatic Reuse**: Existing SSH keys with matching public key content are reused for new VPS instances
- **Single Key Source**: All SSH keys derive from the same CSR private key stored in Cloudflare KV
- **Key Identification**: All keys are labeled with `managed_by: xanthus` for identification
- **Efficient Management**: Prevents duplicate SSH keys in Hetzner Cloud while maintaining security

#### SSH Key Reuse Logic
1. **VPS Creation Process**: Each VPS creation generates a new timestamp-based key name
2. **Existing Key Check**: System searches for existing keys with matching public key content
3. **Smart Reuse**: If found, the existing key is reused (retaining its original name)
4. **New Key Creation**: If no match found, a new SSH key is created with the timestamp name
5. **Consistent Access**: All VPS instances use the same underlying RSA private key for SSH access

### API Endpoints
- `GET /vps` - VPS management page
- `GET /vps/list` - List all VPS instances (JSON)
- `POST /vps/create` - Create new VPS with SSL and SSH
- `POST /vps/delete` - Delete VPS and clean up KV storage
- `POST /vps/{poweroff|poweron|reboot}` - Power management operations

### Cloud-Init Configuration
VPS instances are created with comprehensive cloud-init setup:
```yaml
packages:
  - curl, wget, git, apt-transport-https, ca-certificates, gnupg, lsb-release

write_files:
  - path: /opt/xanthus/ssl/server.crt    # Cloudflare origin certificate
  - path: /opt/xanthus/ssl/server.key    # RSA private key
  - path: /opt/xanthus/info.txt          # Server metadata

runcmd:
  - Install K3s with custom SSL certificates
  - Configure K3s to use /opt/xanthus/ssl/ certificates
  - Enable and start K3s service
```

### SSH Access Pattern
```bash
# Users can access VPS using the CSR private key:
ssh -i /path/to/csr_private_key root@{server_ip}

# The app can programmatically access servers using stored configuration:
# 1. Retrieve config:ssl:csr from KV
# 2. Get vps:{server_id}:config for connection details
# 3. Use CSR private key for SSH authentication
```