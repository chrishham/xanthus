# Services Architecture

## 📋 Purpose
Business logic layer with external service integrations and orchestration patterns.

## 🏗️ Architecture

### Service Organization
```
internal/services/
├── Application Services  # App deployment and management
├── Infrastructure       # Cloud provider integrations  
├── Supporting Services  # Utilities and helpers
└── Factory Pattern     # Service composition
```

### Service Composition Pattern
```go
// Services compose other services for complex operations
SimpleApplicationService {
    KVService, SSHService, HelmService, GitHubService, ...
}
```

## 🔧 Key Components

### Application Services
- **`application_service_core.go:124`** - `CreateApplication()` - Main app creation logic
- **`application_service_deployment.go:14`** - `deployApplication()` - Core deployment orchestration
- **`application_service_templates.go`** - `generateValuesFile()` - Template processing
- **`application_catalog.go`** - `LoadApplicationCatalog()` - App definition management
- **`application_factory.go`** - `CreateApplicationService()` - Service factory
- **`application_registry.go`** - `RegisterApplication()` - App type registry

### Infrastructure Services
- **`hetzner.go`** - `CreateVPS()`, `DeleteVPS()`, `ListVPS()` - Hetzner Cloud API
- **`oci.go`** - `CreateOCIInstance()`, `DeleteOCIInstance()` - Oracle Cloud API
- **`cloudflare_core.go`** - `GetZones()`, `ValidateToken()` - Cloudflare base
- **`cloudflare_dns.go`** - `CreateDNSRecord()`, `DeleteDNSRecord()` - DNS management
- **`cloudflare_ssl.go`** - `GenerateSSLCertificate()` - SSL certificate management

### Supporting Services
- **`ssh_connection.go`** - `EstablishConnection()` - SSH connection management
- **`ssh_operations.go`** - `ExecuteCommand()`, `TransferFile()` - SSH operations
- **`helm.go`** - `InstallChart()`, `UninstallChart()` - Helm deployment
- **`github.go`** - `GetLatestRelease()` - GitHub API integration
- **`kv.go`** - `Get()`, `Set()`, `Delete()` - Key-Value store operations
- **`version_service.go`** - `GetLatestVersion()` - Version resolution

## 🔗 Dependencies & Flow

### Application Deployment Flow
```
CreateApplication() → deployApplication() → SSH Setup → Helm Install → DNS/SSL Config
```

### Service Dependencies
```go
// application_service_core.go
SimpleApplicationService {
    KVService        // Data persistence
    SSHService       // Server communication  
    HelmService      // K8s deployment
    GitHubService    // Version resolution
    CloudflareService // DNS/SSL management
}
```

## 📊 Core Service Patterns

### Error Handling
```go
// Structured error responses
if err != nil {
    log.Printf("Service error: %v", err)
    return nil, fmt.Errorf("operation failed: %w", err)
}
```

### Configuration Loading
```go
// YAML-based configuration
config, err := loadConfig(configPath)
if err != nil {
    return nil, fmt.Errorf("config load failed: %w", err)
}
```

### Resource Management
```go
// Cleanup on failure
defer func() {
    if err != nil {
        cleanup(resources)
    }
}()
```

## 🎯 Key Functions Reference

### Application Services
- `CreateApplication()` - Create new application deployment
- `deployApplication()` - Execute deployment pipeline
- `generateValuesFile()` - Process Helm values templates
- `copyLocalChartToVPS()` - Transfer local charts to VPS
- `configureApplicationDNS()` - Set up DNS records
- `configureVPSSSL()` - Configure SSL certificates
- `retrieveCodeServerPassword()` - Extract app passwords

### Infrastructure Services
- `CreateVPS()` - Provision new VPS instance
- `DeleteVPS()` - Remove VPS instance
- `ListVPS()` - Get VPS inventory
- `CreateDNSRecord()` - Add DNS entry
- `GenerateSSLCertificate()` - Create SSL cert
- `EstablishConnection()` - Create SSH connection
- `ExecuteCommand()` - Run SSH commands

### Supporting Services
- `InstallChart()` - Deploy Helm chart
- `GetLatestRelease()` - Fetch latest version
- `Get()`, `Set()` - KV store operations
- `EncryptPassword()` - Encrypt sensitive data
- `ValidateToken()` - Verify API tokens

## 🚀 Application Deployment Deep Dive

### 1. Application Creation (`application_service_core.go:124`)
```go
func (s *SimpleApplicationService) CreateApplication(data map[string]interface{}) (*models.Application, error) {
    // 1. Generate unique ID
    appID := fmt.Sprintf("app-%d", time.Now().Unix())
    
    // 2. Create application model
    app := &models.Application{
        ID: appID,
        Status: "creating",
        // ... other fields
    }
    
    // 3. Save to KV store
    s.kvService.Set(fmt.Sprintf("app:%s", appID), app)
    
    // 4. Deploy application
    err := s.deployApplication(app, data)
    
    // 5. Update status
    if err != nil {
        app.Status = "failed"
    } else {
        app.Status = "deployed"
    }
    
    return app, err
}
```

### 2. Deployment Pipeline (`application_service_deployment.go:14`)
```go
func (s *SimpleApplicationService) deployApplication(app *models.Application, data map[string]interface{}) error {
    // 1. VPS Setup
    vpsConfig := s.getVPSConfig(vpsID)
    conn := s.sshService.EstablishConnection(vpsConfig)
    
    // 2. Chart Setup
    if helmConfig.Repository == "local" {
        s.copyLocalChartToVPS(conn, chartPath)
    } else {
        s.cloneGitHubRepo(conn, repoURL)
    }
    
    // 3. Values Generation
    valuesFile := s.generateValuesFile(app, data)
    
    // 4. Helm Deployment
    err := s.helmService.InstallChart(releaseName, chartPath, valuesFile)
    
    // 5. DNS/SSL Configuration
    s.configureApplicationDNS(app)
    s.configureVPSSSL(app)
    
    // 6. Password Retrieval
    password := s.retrieveCodeServerPassword(app)
    
    return err
}
```

### 3. Template Processing (`application_service_templates.go`)
```go
func (s *SimpleApplicationService) generateValuesFile(app *models.Application, data map[string]interface{}) string {
    // 1. Load template
    template := s.loadTemplate(app.Type)
    
    // 2. Placeholder substitution
    values := strings.ReplaceAll(template, "{{VERSION}}", version)
    values = strings.ReplaceAll(values, "{{SUBDOMAIN}}", subdomain)
    values = strings.ReplaceAll(values, "{{DOMAIN}}", domain)
    
    // 3. Upload to VPS
    valuesPath := fmt.Sprintf("/tmp/%s-values.yaml", app.ID)
    s.sshService.TransferFile(conn, values, valuesPath)
    
    return valuesPath
}
```

## 🔄 Service Interaction Patterns

### Chain of Responsibility
```go
// Services call each other in sequence
CreateApplication → DeployApplication → ConfigureDNS → ConfigureSSL
```

### Factory Pattern
```go
// application_factory.go
func CreateApplicationService(config *Config) ApplicationService {
    return &SimpleApplicationService{
        kvService: NewKVService(config.KV),
        sshService: NewSSHService(config.SSH),
        helmService: NewHelmService(config.Helm),
    }
}
```

### Observer Pattern
```go
// Status updates propagated through KV store
app.Status = "deploying"
s.kvService.Set(fmt.Sprintf("app:%s", app.ID), app)
```

## 📈 Performance Optimizations

### Connection Pooling
```go
// SSH connections reused within deployment
conn := s.sshService.GetConnection(vpsID)
defer s.sshService.ReleaseConnection(conn)
```

### Version Caching
```go
// version_cache.go - Cache GitHub API responses
cache := NewVersionCache(time.Hour * 24)
version := cache.GetLatestVersion(repo)
```

### Async Operations
```go
// Long-running operations handled asynchronously
go func() {
    err := s.deployApplication(app, data)
    // Update status in background
}()
```

## 🔒 Security Considerations

### Credential Management
```go
// Passwords encrypted with user token
encryptedPassword := s.encryptPassword(password, userToken)
s.kvService.Set(fmt.Sprintf("app:%s:password", appID), encryptedPassword)
```

### SSH Key Security
```go
// Private keys stored in KV, never logged
privateKey := s.kvService.Get("ssh:private_key")
// Key used for connection, never exposed
```

### API Token Validation
```go
// Cloudflare tokens validated before operations
if !s.cloudflareService.ValidateToken(token) {
    return fmt.Errorf("invalid token")
}
```

## 🛠️ Adding New Services

### New Infrastructure Service
1. Create service file: `new_provider.go`
2. Implement provider interface
3. Add to `provider_resolver.go`
4. Register in service factory
5. Add configuration support

### New Application Type
1. Add YAML config in `configs/applications/`
2. Create Helm values template
3. Service automatically supports new type
4. No code changes required

## 🔧 Troubleshooting

### Common Service Issues
- **SSH Connection Failures**: Check VPS status and network
- **Helm Deployment Failures**: Verify chart syntax and values
- **DNS Propagation**: Allow time for DNS changes
- **Version Resolution**: Check GitHub API rate limits
- **KV Store Errors**: Verify Cloudflare token permissions