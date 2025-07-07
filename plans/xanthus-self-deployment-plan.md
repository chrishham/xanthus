# Xanthus Self-Deployment Plan

## Overview

This plan outlines the implementation of a self-deployment system for xanthus, enabling the platform to deploy and update itself using its own configuration-driven deployment infrastructure. This creates a meta-deployment system where xanthus becomes just another application in its own catalog.

## Architecture Goals

- **Configuration-Driven**: Leverage existing YAML-based application configuration system
- **Zero-Downtime Updates**: Use Kubernetes rolling deployments for seamless updates
- **Self-Service Updates**: Enable updates through the existing xanthus UI
- **Operational Safety**: Implement rollback mechanisms and health checks
- **CI/CD Integration**: Automated building, testing, and deployment pipeline

## Phase 1: Containerization & Build Infrastructure

### 1.1 Docker Infrastructure
- **Create Dockerfile** with multi-stage build:
  - Stage 1: Build environment (Go, Node.js for assets)
  - Stage 2: Production image (minimal Alpine/distroless)
  - Embedded assets compilation during build
  - Security scanning and minimal attack surface

### 1.2 Makefile Enhancements
Add Docker-related targets:
```makefile
docker-build:    # Build Docker image locally
docker-push:     # Push to container registry
docker-tag:      # Tag image with version
docker-multi:    # Multi-architecture build
```

### 1.3 Container Registry Setup
- Use GitHub Container Registry (ghcr.io/chrishham/xanthus)
- Automated image tagging based on Git tags
- Multi-architecture support (AMD64, ARM64)
- Image vulnerability scanning

## Phase 2: CI/CD Pipeline with GitHub Actions

### 2.1 Build & Test Workflow (.github/workflows/ci.yml)
```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    - Run make test (unit + integration)
    - Run make lint
    - Generate coverage reports
  
  build:
    - Build Docker image
    - Run security scans
    - Test image functionality
```

### 2.2 Release Workflow (.github/workflows/release.yml)
```yaml
name: Release
on:
  push:
    tags: ['v*']
jobs:
  release:
    - Build multi-arch Docker images
    - Push to ghcr.io
    - Create GitHub release
    - Update deployment manifests
```

### 2.3 Semantic Versioning
- Use conventional commits for automatic version bumping
- Automated changelog generation
- Git tag creation triggers release workflow

## Phase 3: Xanthus Platform Helm Chart

### 3.1 Chart Structure
```
charts/xanthus-platform/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── deployment.yaml      # StatefulSet for persistence
│   ├── service.yaml         # ClusterIP and LoadBalancer
│   ├── ingress.yaml         # External access
│   ├── configmap.yaml       # Application configuration
│   ├── secret.yaml          # Sensitive data
│   ├── pvc.yaml             # Persistent storage
│   └── _helpers.tpl         # Template helpers
```

### 3.2 Deployment Configuration
- **StatefulSet**: For persistent storage and ordered deployment
- **Resource Limits**: CPU/Memory constraints for stability
- **Health Checks**: Readiness and liveness probes
- **Environment Variables**: Database connections, API keys
- **Persistent Storage**: For application data and logs

### 3.3 Security Configuration
- **Non-root user**: Container security best practices
- **Secret management**: Kubernetes secrets for sensitive data
- **Network policies**: Restrict ingress/egress traffic
- **RBAC**: Minimal required permissions

## Phase 4: Application Catalog Integration

### 4.1 Application Configuration (configs/applications/xanthus.yaml)
```yaml
id: xanthus
name: Xanthus Platform
description: Self-hosted infrastructure management platform
icon: 🚀
category: DevOps

version_source:
  type: github
  source: chrishham/xanthus
  pattern: "v*"

helm_chart:
  repository: local
  chart: xanthus-platform
  version: 1.0.0
  namespace: xanthus-platform
  values_template: xanthus.yaml
  placeholders:
    APPLICATION_VERSION: "{{.Version}}"
    DOMAIN: "{{.Domain}}"
    SUBDOMAIN: "{{.Subdomain}}"

default_port: 8081

# Update strategy configuration
update_policy:
  strategy: manual              # User chooses version (like code-server)
  auto_patch: false            # Don't auto-update patch versions
  auto_minor: false            # Don't auto-update minor versions
  auto_major: false            # Never auto-update major versions
  rollback_enabled: true       # Enable rollback functionality
  
ui_features:
  show_release_notes: true     # Display changelog for selected version
  allow_downgrade: true        # Let users downgrade if needed
  require_confirmation: true   # Confirm before updates
  show_current_version: true   # Display currently running version

requirements:
  min_cpu: 0.5
  min_memory_gb: 1
  min_disk_gb: 5

features:
  - Infrastructure management
  - VPS provisioning
  - DNS/SSL automation
  - Application deployment
  - User-controlled version selection
  - Self-updating capabilities with rollback
```

### 4.2 Helm Values Template (internal/templates/applications/xanthus.yaml)
```yaml
image:
  repository: ghcr.io/chrishham/xanthus
  tag: "{{APPLICATION_VERSION}}"
  pullPolicy: IfNotPresent

persistence:
  enabled: true
  size: 10Gi
  storageClass: ""

ingress:
  enabled: true
  host: "{{SUBDOMAIN}}.{{DOMAIN}}"
  tls: true

resources:
  requests:
    cpu: 200m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 1Gi

env:
  - name: GIN_MODE
    value: "release"
  - name: PORT
    value: "8081"
```

## Phase 5: Self-Update Features

### 5.1 Version Management API
- **GET /api/version/current**: Current running version
- **GET /api/version/available**: Available versions from GitHub releases
- **POST /api/version/update**: Trigger update to specific version
- **GET /api/version/status**: Update progress monitoring
- **POST /api/version/rollback**: Rollback to previous version

### 5.2 UI Integration (Following Code-Server Pattern)
- **Version selector dropdown**: List of available versions from GitHub
- **Current version display**: Show currently running version with highlight
- **"Update to Latest" button**: Quick update to newest version
- **"Update to Selected" button**: Update to user-chosen version
- **Release notes display**: Show changelog for selected version
- **Version comparison**: Highlight if selected version is newer/older
- **Update confirmation modal**: Confirm version change with details
- **Progress tracking**: Real-time update status with progress bar

### 5.3 Update Strategy Options
```yaml
# Update strategies available in UI
update_strategies:
  manual:           # User selects specific version (default)
    description: "Full control over version selection"
    auto_update: false
    
  auto_patch:       # Auto-update patch versions (1.0.1 → 1.0.2)
    description: "Automatically apply security patches"
    auto_update: true
    scope: "patch"
    
  auto_minor:       # Auto-update minor versions (1.0.x → 1.1.x)
    description: "Automatically apply feature updates"
    auto_update: true
    scope: "minor"
    
  auto_major:       # Manual only for major versions (1.x.x → 2.x.x)
    description: "Major versions require manual approval"
    auto_update: false
    scope: "major"
```

### 5.4 Safety Mechanisms
- **Pre-update validation**: Check system health before update
- **Compatibility checks**: Warn about breaking changes between versions
- **Backup creation**: Automatic backup before major updates
- **Rolling deployment**: Zero-downtime updates via Kubernetes
- **Health monitoring**: Continuous health checks during update
- **Automatic rollback**: Revert on failed health checks
- **Manual rollback**: UI option to revert to any previous version
- **Update staging**: Test updates in staging environment first (if available)

### 5.5 Version Selection UI Components
```html
<!-- Version selector similar to code-server -->
<div class="version-selector">
  <label>Select Version:</label>
  <select id="version-dropdown">
    <option value="v2.1.0" selected>v2.1.0 (Current)</option>
    <option value="v2.1.1">v2.1.1 (Latest)</option>
    <option value="v2.0.5">v2.0.5</option>
    <option value="v2.0.4">v2.0.4</option>
  </select>
  
  <div class="version-info">
    <p>Current: v2.1.0</p>
    <p>Selected: v2.1.1 (newer)</p>
  </div>
  
  <div class="update-actions">
    <button class="update-latest">Update to Latest</button>
    <button class="update-selected">Update to Selected</button>
    <button class="rollback">Rollback</button>
  </div>
  
  <div class="release-notes">
    <h4>Release Notes for v2.1.1:</h4>
    <ul>
      <li>Bug fixes in VPS management</li>
      <li>Security improvements</li>
      <li>Performance optimizations</li>
    </ul>
  </div>
</div>
```

## Phase 6: Operational Considerations

### 6.1 Monitoring & Observability
- **Health endpoints**: `/health`, `/ready`, `/metrics`
- **Structured logging**: JSON format for log aggregation
- **Metrics collection**: Prometheus-compatible metrics
- **Alerting**: Critical failure notifications

### 6.2 Backup & Recovery
- **Configuration backup**: Automated backup of critical data
- **State preservation**: Persistent volume management
- **Disaster recovery**: Documentation and procedures
- **Database migration**: Schema updates during upgrades

### 6.3 Security Considerations
- **Image scanning**: Vulnerability assessment in CI/CD
- **Secrets rotation**: Automated credential management
- **Network isolation**: Kubernetes network policies
- **Access control**: RBAC for deployment operations

## Implementation Timeline

### Week 1-2: Foundation
- [ ] Create Dockerfile and optimize build process
- [ ] Set up GitHub Actions CI/CD pipeline
- [ ] Implement container registry workflow

### Week 3-4: Helm Chart Development
- [ ] Create xanthus-platform Helm chart
- [ ] Implement StatefulSet with persistence
- [ ] Add security and monitoring configurations

### Week 5-6: Application Integration
- [ ] Add xanthus to application catalog
- [ ] Create values template and configuration
- [ ] Test self-deployment functionality

### Week 7-8: Self-Update Features
- [ ] Implement version management APIs
- [ ] Create version selector UI (following code-server pattern)
- [ ] Add update strategy configuration options
- [ ] Implement safety and rollback mechanisms
- [ ] Create release notes display functionality

### Week 9-10: Testing & Documentation
- [ ] Comprehensive testing of self-deployment
- [ ] Performance and security testing
- [ ] Documentation and operational procedures

## Benefits

1. **Automated Updates**: Streamlined deployment process
2. **Consistency**: Same deployment patterns as other applications
3. **Zero Downtime**: Kubernetes rolling deployments
4. **Self-Service**: Updates through existing UI
5. **Rollback Safety**: Built-in rollback capabilities
6. **CI/CD Integration**: Automated testing and deployment
7. **Operational Excellence**: Monitoring and alerting

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Self-update failures | Automatic rollback on health check failures |
| Resource exhaustion | Resource limits and monitoring |
| Security vulnerabilities | Automated scanning and minimal attack surface |
| Data loss | Persistent volume management and backups |
| Deployment loops | Circuit breakers and manual intervention |

## Success Criteria

- [ ] Xanthus can deploy itself as an application
- [ ] Zero-downtime updates through Kubernetes
- [ ] UI shows current version and available versions (like code-server)
- [ ] Users can select specific versions or update to latest
- [ ] Release notes are displayed for version selection
- [ ] Update confirmation and progress tracking work reliably
- [ ] Rollback functionality works for any previous version
- [ ] Automatic rollback on health check failures
- [ ] Comprehensive monitoring and alerting
- [ ] Full CI/CD pipeline with automated testing

This plan transforms xanthus from a deployment tool into a self-managing platform, embodying the principles of infrastructure as code and GitOps while maintaining operational safety and reliability.