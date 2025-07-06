# Models Architecture

## 📋 Purpose
Domain models and data structures for type-safe data handling across the application.

## 🏗️ Architecture

### Model Organization
```
internal/models/
├── application.go   # Application and deployment models
├── catalog.go      # Application catalog structures
├── config.go       # Configuration models
├── types.go        # Common types and enums
└── requirements.go # Application requirements
```

### Data Flow Pattern
```
JSON/Form → Model Struct → Service Layer → External API → KV Store
```

## 🔧 Key Models

### Application Models (`application.go`)
```go
// Main application entity
type Application struct {
    ID          string    `json:"id" kv:"id"`
    Name        string    `json:"name" kv:"name"`
    Type        string    `json:"type" kv:"type"`
    Status      string    `json:"status" kv:"status"`
    URL         string    `json:"url,omitempty" kv:"url"`
    Subdomain   string    `json:"subdomain" kv:"subdomain"`
    Domain      string    `json:"domain" kv:"domain"`
    VPS         string    `json:"vps" kv:"vps"`
    VPSName     string    `json:"vps_name" kv:"vps_name"`
    Description string    `json:"description" kv:"description"`
    CreatedAt   time.Time `json:"created_at" kv:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" kv:"updated_at"`
}

// Predefined application definition
type PredefinedApplication struct {
    ID           string              `yaml:"id" json:"id"`
    Name         string              `yaml:"name" json:"name"`
    Category     string              `yaml:"category" json:"category"`
    Description  string              `yaml:"description" json:"description"`
    Icon         string              `yaml:"icon" json:"icon"`
    HelmChart    HelmChartConfig     `yaml:"helm_chart" json:"helm_chart"`
    Version      VersionConfig       `yaml:"version" json:"version"`
    Requirements []string            `yaml:"requirements" json:"requirements"`
    Features     ApplicationFeatures `yaml:"features" json:"features"`
    Documentation DocumentationLinks `yaml:"documentation" json:"documentation"`
}
```

### Catalog Models (`catalog.go`)
```go
// Application catalog structure
type ApplicationCatalog struct {
    Applications []PredefinedApplication `json:"applications"`
    Categories   []string                `json:"categories"`
    LastUpdated  time.Time              `json:"last_updated"`
}

// Helm chart configuration
type HelmChartConfig struct {
    Repository string `yaml:"repository" json:"repository"`
    Chart      string `yaml:"chart" json:"chart"`
    Namespace  string `yaml:"namespace" json:"namespace"`
    Version    string `yaml:"version,omitempty" json:"version,omitempty"`
}

// Version configuration
type VersionConfig struct {
    Source     string `yaml:"source" json:"source"`         // github|helm|fixed
    Repository string `yaml:"repository" json:"repository"` // owner/repo
    ChartName  string `yaml:"chart_name" json:"chart_name"` // helm chart name
    Value      string `yaml:"value,omitempty" json:"value,omitempty"` // fixed version
}
```

### Configuration Models (`config.go`)
```go
// Main configuration structure
type Config struct {
    Server      ServerConfig      `yaml:"server" json:"server"`
    Cloudflare  CloudflareConfig  `yaml:"cloudflare" json:"cloudflare"`
    Hetzner     HetznerConfig     `yaml:"hetzner" json:"hetzner"`
    Oracle      OracleConfig      `yaml:"oracle" json:"oracle"`
    KV          KVConfig          `yaml:"kv" json:"kv"`
    SSH         SSHConfig         `yaml:"ssh" json:"ssh"`
    Logging     LoggingConfig     `yaml:"logging" json:"logging"`
}

// VPS configuration
type VPSConfig struct {
    ID          string            `json:"id" kv:"id"`
    Name        string            `json:"name" kv:"name"`
    Provider    string            `json:"provider" kv:"provider"`
    IPAddress   string            `json:"ip_address" kv:"ip_address"`
    Status      string            `json:"status" kv:"status"`
    Location    string            `json:"location" kv:"location"`
    Size        string            `json:"size" kv:"size"`
    CreatedAt   time.Time         `json:"created_at" kv:"created_at"`
    Metadata    map[string]string `json:"metadata" kv:"metadata"`
}
```

### Type Definitions (`types.go`)
```go
// Application status enum
type ApplicationStatus string

const (
    StatusCreating  ApplicationStatus = "creating"
    StatusDeploying ApplicationStatus = "deploying"
    StatusDeployed  ApplicationStatus = "deployed"
    StatusFailed    ApplicationStatus = "failed"
    StatusStopped   ApplicationStatus = "stopped"
)

// VPS provider enum
type VPSProvider string

const (
    ProviderHetzner VPSProvider = "hetzner"
    ProviderOracle  VPSProvider = "oracle"
    ProviderAWS     VPSProvider = "aws"
)

// Application category enum
type ApplicationCategory string

const (
    CategoryDevelopment ApplicationCategory = "development"
    CategoryDevOps      ApplicationCategory = "devops"
    CategoryMonitoring  ApplicationCategory = "monitoring"
    CategorySecurity    ApplicationCategory = "security"
)
```

## 🔗 Model Relationships

### Application → VPS Relationship
```go
// Application references VPS
type Application struct {
    VPS     string `json:"vps" kv:"vps"`         // VPS ID
    VPSName string `json:"vps_name" kv:"vps_name"` // VPS display name
}

// VPS contains application metadata
type VPSConfig struct {
    Metadata map[string]string `json:"metadata" kv:"metadata"`
    // metadata["applications"] = "app1,app2,app3"
}
```

### Application → Catalog Relationship
```go
// Application.Type links to PredefinedApplication.ID
application := &Application{
    Type: "code-server", // References PredefinedApplication.ID
}

predefinedApp := catalog.GetApplication("code-server")
```

## 📊 Data Serialization Patterns

### JSON Serialization
```go
// JSON tags for API responses
type Application struct {
    ID     string `json:"id"`
    Name   string `json:"name"`
    Status string `json:"status"`
}

// Omit empty fields
type Application struct {
    URL string `json:"url,omitempty"`
}
```

### KV Store Serialization
```go
// KV tags for storage
type Application struct {
    ID     string `json:"id" kv:"id"`
    Name   string `json:"name" kv:"name"`
    Status string `json:"status" kv:"status"`
}

// KV key pattern: "app:{id}"
kvKey := fmt.Sprintf("app:%s", application.ID)
```

### YAML Configuration
```go
// YAML tags for configuration files
type PredefinedApplication struct {
    ID          string          `yaml:"id" json:"id"`
    Name        string          `yaml:"name" json:"name"`
    HelmChart   HelmChartConfig `yaml:"helm_chart" json:"helm_chart"`
}
```

## 🔄 Model Validation

### Required Field Validation
```go
// Validation tags
type Application struct {
    ID        string `json:"id" validate:"required"`
    Name      string `json:"name" validate:"required"`
    Type      string `json:"type" validate:"required"`
    Subdomain string `json:"subdomain" validate:"required,alphanum"`
    Domain    string `json:"domain" validate:"required,fqdn"`
}
```

### Custom Validation
```go
// Model validation methods
func (a *Application) Validate() error {
    if a.ID == "" {
        return fmt.Errorf("application ID is required")
    }
    
    if !IsValidSubdomain(a.Subdomain) {
        return fmt.Errorf("invalid subdomain format")
    }
    
    return nil
}
```

## 🎯 Key Model Functions

### Application Models
- `NewApplication()` - Create new application instance
- `(a *Application) GetURL()` - Generate application URL
- `(a *Application) IsDeployed()` - Check deployment status
- `(a *Application) Update()` - Update application fields
- `(a *Application) ToMap()` - Convert to map for templates

### Catalog Models
- `LoadApplicationCatalog()` - Load predefined applications
- `(c *ApplicationCatalog) GetApplication(id)` - Find application by ID
- `(c *ApplicationCatalog) GetByCategory(category)` - Filter by category
- `(c *ApplicationCatalog) Validate()` - Validate catalog structure

### Configuration Models
- `LoadConfig()` - Load application configuration
- `(c *Config) Validate()` - Validate configuration
- `(c *Config) GetVPSConfig(id)` - Get VPS configuration
- `(c *Config) GetProviderConfig(provider)` - Get provider settings

## 🔧 Model Utilities

### Type Conversion
```go
// Convert between model formats
func (a *Application) ToJSON() ([]byte, error) {
    return json.Marshal(a)
}

func (a *Application) FromJSON(data []byte) error {
    return json.Unmarshal(data, a)
}

func (a *Application) ToMap() map[string]interface{} {
    return map[string]interface{}{
        "id":          a.ID,
        "name":        a.Name,
        "type":        a.Type,
        "subdomain":   a.Subdomain,
        "domain":      a.Domain,
    }
}
```

### Model Helpers
```go
// Helper functions for common operations
func (a *Application) GetFullURL() string {
    return fmt.Sprintf("https://%s.%s", a.Subdomain, a.Domain)
}

func (a *Application) IsRunning() bool {
    return a.Status == string(StatusDeployed)
}

func (a *Application) IsFailed() bool {
    return a.Status == string(StatusFailed)
}
```

## 📈 Performance Considerations

### Memory Optimization
```go
// Use pointers for large structs
type Application struct {
    Metadata *map[string]string `json:"metadata,omitempty"`
}

// Lazy loading for complex relationships
func (a *Application) GetVPSConfig() *VPSConfig {
    if a.vpsConfig == nil {
        a.vpsConfig = LoadVPSConfig(a.VPS)
    }
    return a.vpsConfig
}
```

### Caching Strategy
```go
// Cache frequently accessed models
type CachedApplicationCatalog struct {
    catalog   *ApplicationCatalog
    lastLoad  time.Time
    cacheTTL  time.Duration
}

func (c *CachedApplicationCatalog) GetApplication(id string) *PredefinedApplication {
    if time.Since(c.lastLoad) > c.cacheTTL {
        c.reload()
    }
    return c.catalog.GetApplication(id)
}
```

## 🔒 Security Considerations

### Sensitive Data Handling
```go
// Exclude sensitive fields from JSON
type Application struct {
    ID       string `json:"id"`
    Password string `json:"-"`        // Never serialize
    Token    string `json:"-"`        // Never serialize
}
```

### Input Sanitization
```go
// Validate and sanitize model inputs
func (a *Application) SetSubdomain(subdomain string) error {
    // Sanitize input
    clean := sanitize(subdomain)
    
    // Validate format
    if !isValidSubdomain(clean) {
        return fmt.Errorf("invalid subdomain")
    }
    
    a.Subdomain = clean
    return nil
}
```

## 🛠️ Adding New Models

### New Entity Model
1. Create struct with appropriate tags
2. Add validation methods
3. Implement serialization interfaces
4. Add to service layer integration
5. Update database schema if needed

### New Configuration Model
1. Add to `config.go`
2. Update YAML configuration files
3. Add validation logic
4. Update loading functions
5. Document configuration options

## 🔧 Common Model Patterns

### Builder Pattern
```go
// Application builder for complex construction
type ApplicationBuilder struct {
    app *Application
}

func NewApplicationBuilder() *ApplicationBuilder {
    return &ApplicationBuilder{app: &Application{}}
}

func (b *ApplicationBuilder) WithName(name string) *ApplicationBuilder {
    b.app.Name = name
    return b
}

func (b *ApplicationBuilder) Build() *Application {
    b.app.ID = generateID()
    b.app.CreatedAt = time.Now()
    return b.app
}
```

### Factory Pattern
```go
// Model factory for different application types
func CreateApplication(appType string, data map[string]interface{}) *Application {
    switch appType {
    case "code-server":
        return createCodeServerApp(data)
    case "argocd":
        return createArgoCDApp(data)
    default:
        return createGenericApp(data)
    }
}
```