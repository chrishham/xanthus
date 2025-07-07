# Handlers Architecture

## 📋 Purpose
HTTP request processing layer using domain-driven organization and dependency injection pattern.

## 🏗️ Architecture

### Handler Organization
```
internal/handlers/
├── applications/     # Application deployment domain
├── vps/             # VPS management domain  
└── core handlers    # Auth, DNS, Terminal, Pages
```

### Dependency Injection Pattern
```go
// RouteConfig in cmd/xanthus/main.go:51-61
RouteConfig {
    AuthHandler, DNSHandler, VPSHandlers, AppsHandler
}
```

## 🔧 Key Components

### Applications Domain (`applications/`)
- **`http.go:123`** - `HandleApplicationsCreate()` - Main app deployment endpoint
- **`base.go`** - Handler initialization with service dependencies
- **`codeserver_handlers.go`** - Code-server specific operations
- **`argocd_handlers.go`** - ArgoCD specific operations  
- **`middleware.go`** - Application-specific middleware
- **`config.go`** - Configuration structures for handlers

### VPS Domain (`vps/`)
- **`vps_lifecycle.go`** - `CreateVPS()`, `DeleteVPS()`, `StartVPS()`, `StopVPS()`
- **`vps_info.go`** - `GetVPSInfo()`, `ListVPS()` - Status and information retrieval
- **`vps_config.go`** - `ConfigureVPS()` - VPS configuration management
- **`vps_meta.go`** - `GetVPSMetadata()` - VPS metadata operations
- **`base.go`** - VPS handler initialization

### Core Handlers
- **`auth.go`** - `HandleLogin()`, `HandleLogout()` - Authentication flow
- **`dns.go`** - `HandleDNSConfig()` - DNS configuration
- **`terminal.go`** - `HandleTerminal()` - Web terminal access
- **`pages.go`** - `HandleHomePage()`, `HandleApplicationsPage()` - UI pages
- **`websocket_terminal.go`** - WebSocket terminal connection

## 🔗 Dependencies

### Handler → Service Flow
```
HTTP Request → Handler Validation → Service Call → External API/KV → Response
```

### Service Dependencies (Injected via RouteConfig)
- **Application Handlers**: `catalog`, `validator`, `serviceFactory`
- **VPS Handlers**: `vpsService`, `kvService`, `provider services`
- **Core Handlers**: `authService`, `cloudflareService`, `terminalService`

## 📊 Request/Response Patterns

### JSON API Endpoints
```go
// Standard JSON response structure
{
    "success": bool,
    "message": string,
    "data": interface{}
}
```

### HTMX Endpoints
```go
// Returns HTML partials for dynamic updates
return c.HTML(200, "partials/app-card.html", data)
```

### Form Processing
```go
// applications/http.go:150-175
1. Bind form data to struct
2. Validate application type
3. Look up VPS configuration  
4. Convert to service format
5. Call service method
```

## 🔄 Common Handler Patterns

### Error Handling
```go
if err != nil {
    log.Printf("Error: %v", err)
    return c.JSON(500, gin.H{"error": "Internal server error"})
}
```

### Authentication Middleware
```go
// middleware/auth.go - Token-based authentication
func AuthMiddleware() gin.HandlerFunc {
    // Validate Cloudflare token
    // Set user context
}
```

### Logging Pattern
```go
log.Printf("[%s] %s - %s", method, path, userID)
```

## 🎯 Key Functions Reference

### Application Handlers
- `HandleApplicationsCreate()` - Deploy new application
- `HandleApplicationsList()` - List all applications
- `HandleApplicationsDelete()` - Remove application
- `HandleApplicationsRestart()` - Restart application
- `HandleGetApplicationPassword()` - Retrieve app password

### VPS Handlers  
- `HandleVPSCreate()` - Create new VPS instance
- `HandleVPSList()` - List all VPS instances
- `HandleVPSDelete()` - Delete VPS instance
- `HandleVPSStart()` - Start VPS instance
- `HandleVPSStop()` - Stop VPS instance
- `HandleVPSInfo()` - Get VPS details

### Core Handlers
- `HandleLogin()` - Process login form
- `HandleTerminal()` - Initialize web terminal
- `HandleDNSConfig()` - Configure DNS settings
- `HandleApplicationsPage()` - Render applications UI
- `HandleVPSPage()` - Render VPS management UI

## 🚀 Adding New Handlers

### New Domain Handler
1. Create folder: `internal/handlers/new-domain/`
2. Add `base.go` with initialization
3. Add domain-specific handlers
4. Update `RouteConfig` in `cmd/xanthus/main.go`
5. Register routes in `internal/router/routes.go`

### New Endpoint in Existing Domain
1. Add method to appropriate handler struct
2. Follow validation → service call → response pattern
3. Add route registration
4. Update corresponding service if needed

## 🔒 Security Considerations

- **Authentication**: All handlers require valid Cloudflare token
- **Input Validation**: Form data validated before processing
- **Error Masking**: Internal errors not exposed to users
- **CORS**: Configured for trusted origins only
- **Rate Limiting**: Implemented via middleware

## 📈 Performance Notes

- **Dependency Injection**: Services initialized once at startup
- **Connection Pooling**: External API clients reused
- **Error Caching**: Failed requests cached to prevent retry storms
- **Async Processing**: Long-running operations handled asynchronously