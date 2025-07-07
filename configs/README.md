# Configuration Architecture

## 📋 Purpose
Configuration-driven application deployment system that enables adding new applications without code changes.

## 🏗️ Architecture

### Configuration-Driven Deployment
```
YAML Config → Template Processing → Helm Values → K8s Deployment
```

### Directory Structure
```
configs/
└── applications/
    ├── code-server.yaml    # VS Code in browser
    ├── argocd.yaml        # GitOps continuous deployment
    ├── headlamp.yaml      # Kubernetes dashboard
    ├── open-webui.yaml    # AI interface
    └── template.yaml      # Template for new apps
```

## 🔧 Configuration Schema

### Application Definition Structure
```yaml
# Application metadata
name: "Application Name"
id: "app-id"
category: "development|devops|monitoring"
description: "Short description"
icon: "icon-name"

# Deployment configuration
helm_chart:
  repository: "local|github.com/repo|https://helm.repo"
  chart: "chart-name"
  namespace: "target-namespace"

# Version management
version:
  source: "github|helm|fixed"
  repository: "owner/repo"
  chart_name: "chart-name"

# Application requirements
requirements:
  - "kubernetes"
  - "ingress"
  - "persistent-storage"

# Feature flags
features:
  supports_authentication: true
  supports_ssl: true
  supports_custom_config: false

# Documentation
documentation:
  setup_guide: "URL to setup guide"
  user_guide: "URL to user guide"
```

## 🎯 Key Configuration Files

### Code-Server (`code-server.yaml`)
```yaml
name: "Code-Server"
id: "code-server"
description: "VS Code in the browser"
category: "development"
icon: "code"
helm_chart:
  repository: "local"                    # Uses local chart
  chart: "xanthus-code-server"
  namespace: "code-server"              # Type-based namespace
version:
  source: "github"
  repository: "coder/code-server"
features:
  supports_authentication: true
  supports_ssl: true
  supports_custom_config: true
```

### ArgoCD (`argocd.yaml`)
```yaml
name: "ArgoCD"
id: "argocd"
description: "GitOps continuous deployment"
category: "devops"
icon: "git-branch"
helm_chart:
  repository: "https://argoproj.github.io/argo-helm"
  chart: "argo-cd"
  namespace: "argocd"
version:
  source: "helm"
  chart_name: "argo-cd"
features:
  supports_authentication: true
  supports_ssl: true
  supports_rbac: true
```

### Template (`template.yaml`)
```yaml
name: "{{APP_NAME}}"
id: "{{APP_ID}}"
description: "{{APP_DESCRIPTION}}"
category: "{{CATEGORY}}"
icon: "{{ICON}}"
helm_chart:
  repository: "{{REPOSITORY}}"
  chart: "{{CHART_NAME}}"
  namespace: "{{NAMESPACE}}"
version:
  source: "{{VERSION_SOURCE}}"
  repository: "{{VERSION_REPO}}"
requirements:
  - "kubernetes"
  - "ingress"
features:
  supports_authentication: true
  supports_ssl: true
```

## 🔄 Configuration Loading Process

### 1. Application Catalog Loading
```go
// application_catalog.go
func LoadApplicationCatalog() ([]PredefinedApplication, error) {
    // 1. Scan configs/applications/ directory
    files := scanConfigDir("configs/applications/")
    
    // 2. Parse each YAML file
    for _, file := range files {
        app := parseApplicationConfig(file)
        catalog = append(catalog, app)
    }
    
    // 3. Validate configurations
    validateCatalog(catalog)
    
    return catalog, nil
}
```

### 2. Template Processing
```go
// application_service_templates.go
func (s *SimpleApplicationService) generateValuesFile(app *models.Application, data map[string]interface{}) string {
    // 1. Load application config
    config := s.getApplicationConfig(app.Type)
    
    // 2. Load Helm values template
    template := s.loadTemplate(fmt.Sprintf("internal/templates/applications/%s.yaml", app.Type))
    
    // 3. Placeholder substitution
    values := s.processTemplate(template, data)
    
    return values
}
```

## 📊 Configuration Patterns

### Chart Repository Types
```yaml
# Local chart (fastest deployment)
helm_chart:
  repository: "local"
  chart: "xanthus-code-server"

# GitHub repository
helm_chart:
  repository: "https://github.com/owner/repo"
  chart: "path/to/chart"

# Helm repository
helm_chart:
  repository: "https://helm.repository.com"
  chart: "chart-name"
```

### Version Source Types
```yaml
# GitHub releases
version:
  source: "github"
  repository: "owner/repo"

# Helm chart versions
version:
  source: "helm"
  chart_name: "chart-name"

# Fixed version
version:
  source: "fixed"
  value: "1.0.0"
```

### Namespace Strategy
```yaml
# Type-based namespaces (recommended)
helm_chart:
  namespace: "code-server"    # All code-server apps
  namespace: "argocd"         # All ArgoCD apps
  namespace: "monitoring"     # All monitoring apps
```

## 🔧 Configuration Validation

### Required Fields
```yaml
# These fields are mandatory
name: "Required - Display name"
id: "Required - Unique identifier"
helm_chart:
  repository: "Required - Chart source"
  chart: "Required - Chart name"
  namespace: "Required - Target namespace"
```

### Validation Rules
```go
// config_catalog.go
func validateApplicationConfig(config *ApplicationConfig) error {
    // 1. Check required fields
    if config.Name == "" || config.ID == "" {
        return fmt.Errorf("missing required fields")
    }
    
    // 2. Validate repository URL
    if !isValidRepository(config.HelmChart.Repository) {
        return fmt.Errorf("invalid repository URL")
    }
    
    // 3. Check namespace format
    if !isValidNamespace(config.HelmChart.Namespace) {
        return fmt.Errorf("invalid namespace format")
    }
    
    return nil
}
```

## 🚀 Adding New Applications

### Step 1: Create Configuration
```bash
# Create new config file
cp configs/applications/template.yaml configs/applications/new-app.yaml
```

### Step 2: Configure Application
```yaml
# Edit configs/applications/new-app.yaml
name: "New Application"
id: "new-app"
description: "Application description"
category: "development"
helm_chart:
  repository: "https://github.com/owner/chart-repo"
  chart: "new-app"
  namespace: "new-app"
version:
  source: "github"
  repository: "owner/new-app"
```

### Step 3: Create Helm Values Template
```bash
# Create template file
touch internal/templates/applications/new-app.yaml
```

### Step 4: Test Configuration
```bash
# Restart application to load new config
make dev
```

## 🔗 Integration with Services

### Service Layer Integration
```go
// Services automatically discover new applications
func (s *SimpleApplicationService) LoadCatalog() {
    // Automatically loads all configs from configs/applications/
    catalog := LoadApplicationCatalog()
    s.catalog = catalog
}
```

### Template System Integration
```go
// Templates automatically processed for new apps
func (s *SimpleApplicationService) DeployApplication(appType string) {
    // 1. Load config for appType
    config := s.getApplicationConfig(appType)
    
    // 2. Process corresponding template
    template := s.loadTemplate(fmt.Sprintf("internal/templates/applications/%s.yaml", appType))
    
    // 3. Deploy using config + template
    s.deployWithConfig(config, template)
}
```

## 📈 Configuration Management Best Practices

### Naming Conventions
```yaml
# File naming: lowercase with hyphens
code-server.yaml
open-webui.yaml
vault-secrets.yaml

# ID naming: match filename
id: "code-server"
id: "open-webui"
id: "vault-secrets"
```

### Repository Management
```yaml
# Prefer local charts for custom applications
helm_chart:
  repository: "local"
  chart: "xanthus-custom-app"

# Use official repositories for standard apps
helm_chart:
  repository: "https://official.helm.repo"
  chart: "official-chart"
```

### Version Management
```yaml
# Use GitHub for active development
version:
  source: "github"
  repository: "owner/repo"

# Use Helm for stable releases
version:
  source: "helm"
  chart_name: "stable-chart"
```

## 🔒 Security Considerations

### Configuration Validation
- **Required field validation** prevents incomplete deployments
- **URL validation** prevents malicious repository injection
- **Namespace validation** ensures proper resource isolation

### Secret Management
- **No secrets in configuration files**
- **Secrets managed through KV store**
- **Template-based secret injection**

### Access Control
- **Configuration files version controlled**
- **Deployment requires valid authentication**
- **Namespace-based resource isolation**

## 🛠️ Troubleshooting

### Common Configuration Issues
- **Missing required fields**: Check validation errors
- **Invalid repository URLs**: Verify chart source accessibility
- **Namespace conflicts**: Ensure unique namespace naming
- **Template syntax errors**: Validate YAML formatting
- **Version resolution failures**: Check GitHub API limits

### Debugging Configuration Loading
```go
// Enable debug logging
log.Printf("Loading config: %s", configPath)
log.Printf("Parsed config: %+v", config)
log.Printf("Validation result: %v", err)
```