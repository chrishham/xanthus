# Code-Server Deployment Flow

This document outlines the complete flow for deploying a new code-server application in Xanthus, from user request to running application.

## Overview

The deployment process involves multiple services and components working together to provision a fully functional code-server instance with persistent storage, SSL, and DNS configuration.

## Flow Diagram

```
User Request → HTTP Handler → SimpleApplicationService → SSH/Helm → VPS → Running App
     ↓              ↓                    ↓                ↓         ↓
   Web UI    ApplicationsCreate    deployApplication   K8s Deploy  Code-Server
```

## Detailed Step-by-Step Flow

### 1. User Initiates Deployment

**Endpoint**: `POST /applications/create`  
**Handler**: `HandleApplicationsCreate` in `internal/handlers/applications/http.go:123`

**User Input**:
- Subdomain (e.g., "maria20")
- Domain (e.g., "myclasses.gr") 
- VPS selection
- Application type ("code-server")
- Description

### 2. Request Validation & Processing

**Location**: `internal/handlers/applications/http.go:150-175`

1. **Validate application type** against predefined catalog
2. **Look up VPS configuration** by VPS ID
3. **Convert request data** to service-compatible format
4. **Prepare application data map**:
   ```go
   appDataMap := map[string]interface{}{
       "subdomain":   "maria20",
       "domain":      "myclasses.gr", 
       "vps_id":      "12345",
       "vps_name":    "my-vps",
       "description": "Development environment",
   }
   ```

### 3. Application Creation Service

**Service**: `SimpleApplicationService.CreateApplication()`  
**Location**: `internal/services/application_service_core.go:124-204`

**Steps**:
1. **Generate unique application ID**: `app-{timestamp}`
2. **Create application model** with status "creating"
3. **Save to KV store** for persistence
4. **Call deployment method**: `s.deployApplication()`
5. **Update status** based on deployment result
6. **Return application object**

### 4. Core Deployment Logic

**Method**: `SimpleApplicationService.deployApplication()`  
**Location**: `internal/services/application_service_deployment.go:14-140`

#### 4.1 Infrastructure Setup
1. **Retrieve VPS configuration** from KV store
2. **Get SSH private key** for VPS access
3. **Establish SSH connection** to target VPS
4. **Generate release name**: `{subdomain}-code-server` (e.g., "maria20-code-server")
5. **Set namespace**: `code-server` (type-based namespace)
6. **Create Kubernetes namespace** if it doesn't exist

#### 4.2 Chart Selection Logic
```go
if predefinedApp.ID == "code-server" && helmConfig.Repository == "local" {
    // Use local chart (NEW IMPLEMENTATION)
    chartPath = "/tmp/xanthus-code-server"
    copyLocalChartToVPS(conn, sshService, chartPath)
    chartName = chartPath
} else if strings.Contains(helmConfig.Repository, "github.com") {
    // Clone GitHub repository
    git clone {repository} /tmp/{app}-chart
    chartName = "/tmp/{app}-chart/{chart}"
} else {
    // Add Helm repository
    helm repo add {app} {repository}
    chartName = "{app}/{chart}"
}
```

#### 4.3 Local Chart Copy Process (NEW)
**Method**: `copyLocalChartToVPS()`

1. **Create remote directory**: `/tmp/xanthus-code-server/`
2. **Copy chart files**:
   - `Chart.yaml` - Chart metadata
   - `values.yaml` - Default values
   - `templates/_helpers.tpl` - Template helpers
   - `templates/deployment.yaml` - Main deployment
   - `templates/service.yaml` - Service configuration
   - `templates/pvc.yaml` - Persistent volume claim
   - `templates/configmap.yaml` - Setup script & VS Code settings

#### 4.4 Values File Generation
**Method**: `generateValuesFile()`

1. **Load template** from `internal/templates/applications/code-server.yaml`
2. **Perform placeholder substitution**:
   - `{{VERSION}}` → Latest version (e.g., "4.101.2")
   - `{{SUBDOMAIN}}` → User subdomain (e.g., "maria20")
   - `{{DOMAIN}}` → User domain (e.g., "myclasses.gr")
   - `{{RELEASE_NAME}}` → Generated release name
3. **Upload values file** to VPS at `/tmp/{release-name}-values.yaml`

#### 4.5 Helm Deployment
```bash
helm install {release-name} {chart-path} \
  --namespace {namespace} \
  --values {values-file} \
  --wait --timeout 10m
```

**Example**:
```bash
helm install maria20-code-server /tmp/xanthus-code-server \
  --namespace code-server \
  --values /tmp/maria20-code-server-values.yaml \
  --wait --timeout 10m
```

### 5. Kubernetes Resources Created

#### 5.1 ConfigMaps (Local Chart)
- **Setup Script ConfigMap**:
  - Name: `maria20-code-server-setup-script`
  - Contains: `setup-dev-environment.sh`
  - Purpose: On-demand development tools installation

- **VS Code Settings ConfigMap**:
  - Name: `maria20-code-server-vscode-settings`
  - Contains: `settings.json` with default IDE configuration
  - Purpose: Pre-configured VS Code settings

#### 5.2 Persistent Volume Claim
- **Name**: `maria20-code-server-home`
- **Size**: 10GB
- **Access**: ReadWriteOnce
- **Purpose**: Persistent home directory storage

#### 5.3 Deployment
- **Name**: `maria20-code-server`
- **Image**: `codercom/code-server:4.101.2`
- **Init Containers**:
  - `setup-home`: Sets up home directory permissions and VS Code settings
- **Volumes**:
  - Persistent volume mounted at `/home/coder`
  - ConfigMaps mounted for setup script and settings

#### 5.4 Service
- **Name**: `maria20-code-server`
- **Type**: ClusterIP
- **Port**: 8080
- **Purpose**: Internal cluster communication

### 6. Post-Deployment Configuration

#### 6.1 SSL Certificate Setup
**Method**: `configureVPSSSL()`

1. **Check existing SSL configuration** for domain
2. **Generate SSL certificates** if needed (Cloudflare Origin CA)
3. **Configure K3s ingress** with SSL termination
4. **Create TLS secret** in Kubernetes namespace
5. **Mark SSL as configured** in KV store

#### 6.2 DNS Configuration
**Method**: `configureApplicationDNS()`

1. **Get Cloudflare zone ID** for domain
2. **Create DNS A record**: `maria20.myclasses.gr` → VPS IP
3. **Enable proxying** through Cloudflare

#### 6.3 Password Retrieval
**Method**: `retrieveCodeServerPassword()`

1. **Connect to VPS** via SSH
2. **Extract password** from Kubernetes secret:
   ```bash
   kubectl get secret maria20-code-server -n code-server \
     -o jsonpath='{.data.password}' | base64 --decode
   ```
3. **Encrypt password** using Cloudflare token
4. **Store in KV store** with key `app:{app-id}:password`

### 7. Final Status Update

1. **Update application status** to "deployed" or "failed"
2. **Return response** to HTTP handler
3. **Include initial password** in response (for immediate access)

## Configuration Files Used

### Application Definition
**File**: `configs/applications/code-server.yaml`
```yaml
helm_chart:
  repository: local          # Triggers local chart deployment
  chart: xanthus-code-server # Local chart name
  namespace: code-server     # Type-based namespace
```

### Values Template
**File**: `internal/templates/applications/code-server.yaml`
- Contains Helm values with placeholders
- Processed during deployment for customization

### Local Chart Files
**Directory**: `charts/xanthus-code-server/`
- Complete Helm chart with all Kubernetes manifests
- Supports persistent storage and ConfigMaps
- No external dependencies

## Key Features of New Implementation

### ✅ Local Chart Benefits
- **No external dependencies** - No GitHub cloning required
- **Faster deployments** - Local files copy quickly
- **Version control** - Chart templates managed in codebase
- **Customizable** - Full control over Kubernetes manifests

### ✅ Persistent Storage
- **10GB home directory** - All user data persists across restarts
- **Development tools persistence** - Installed packages survive upgrades
- **Project storage** - User code and configurations preserved

### ✅ Setup Script Integration
- **On-demand installation** - Run `./setup-dev-environment.sh` when needed
- **Comprehensive tooling** - Node.js, Python, Go, Docker, kubectl, Helm
- **User-specific packages** - npm and pip packages in persistent home

### ✅ Pre-configured Environment
- **VS Code settings** - Dark theme, auto-save, optimized defaults
- **Terminal ready** - Proper shell configuration
- **Directory structure** - `~/projects`, `~/workspace`, `~/scripts`

## Error Handling

### Deployment Failures
- Application status set to "failed"
- Detailed error messages logged
- Partial deployments cleaned up
- User receives error notification

### Network Issues
- SSH connection retries
- Helm timeout handling (10 minutes)
- DNS propagation delays handled
- SSL certificate generation retries

### Resource Constraints
- PVC creation failure handling
- Namespace permission issues
- Image pull failures
- Pod scheduling problems

## Monitoring & Observability

### Logs
- Deployment progress logged at each step
- SSH command outputs captured
- Helm deployment status tracked
- Error details preserved for debugging

### Status Tracking
- Application status in KV store
- Kubernetes pod status monitoring
- SSL certificate status
- DNS record verification

## Security Considerations

### Secrets Management
- Passwords encrypted with user token
- SSH keys stored securely in KV
- TLS certificates managed per namespace
- No secrets in application logs

### Network Security
- Cloudflare proxy protection
- SSL/TLS encryption enforced
- Internal cluster communication only
- VPS firewall configuration

### Access Control
- User-specific application isolation
- Namespace-based separation
- SSH key-based VPS access
- Token-based API authentication

## Performance Optimizations

### Resource Management
- Appropriate CPU/memory limits
- Efficient storage allocation
- Init container optimization
- Image caching on VPS

### Network Efficiency
- Local chart copying reduces network usage
- Persistent connections for SSH
- Cloudflare CDN benefits
- Optimized DNS TTL settings

### Deployment Speed
- Parallel operations where possible
- Cached dependencies
- Pre-warmed VPS environments
- Optimized container startup