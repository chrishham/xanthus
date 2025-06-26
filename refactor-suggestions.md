# Refactor Suggestions - Xanthus Codebase

## Overview

This document outlines potential improvements and refactoring opportunities for the Xanthus codebase, focusing on extracting reusable components to a `pkg/` directory and architectural improvements.

## 🎯 Reusable Components for pkg/ Migration

### Phase 1: Low-Risk Extractions (Immediate)

#### 1. Cryptographic Utilities → `pkg/crypto/`
**Current Location:** `internal/utils/crypto.go` + CSR generation in `internal/services/cloudflare.go`
**Rationale:** Pure cryptographic functions with zero business logic coupling

**Proposed Structure:**
```
pkg/crypto/
├── aes.go                 # AES encryption/decryption (from utils/crypto.go)
├── csr.go                 # Certificate signing request generation
└── utils.go               # Common crypto utilities
```

**Migration Tasks:**
- Extract `encryptData()` and `decryptData()` functions
- Move CSR generation logic from cloudflare service (lines 77-122)
- Add comprehensive test coverage
- Update import paths in consuming code

#### 2. Network Utilities → `pkg/netutil/`
**Current Location:** `internal/utils/server.go`
**Rationale:** Generic network utilities useful across projects

**Proposed Structure:**
```
pkg/netutil/
├── ports.go               # Port finding utilities
└── network.go             # Additional network helpers
```

**Migration Tasks:**
- Extract `findAvailablePort()` function
- Add port range configuration options
- Add IPv6 support considerations

### Phase 2: Medium-Risk Extractions (Requires Refactoring)

#### 3. SSH Management → `pkg/ssh/`
**Current Location:** `internal/services/ssh.go`
**Rationale:** Generic SSH patterns useful for remote server management

**Proposed Structure:**
```
pkg/ssh/
├── client.go              # Core SSH client with connection management
├── pool.go                # Connection pooling for multiple servers
├── executor.go            # Command execution with output handling
├── types.go               # Common types and interfaces
└── terminal.go            # Terminal session management
```

**Migration Tasks:**
- Extract generic SSH connection logic
- Separate VPS-specific methods (K3s setup, health checks)
- Create interfaces for testability
- Add connection timeout and retry logic
- Implement proper connection cleanup

**Business Logic to Keep Internal:**
- K3s cluster configuration
- VPS health check commands
- Xanthus-specific server setup routines

#### 4. Cloud Provider Clients → `pkg/cloudproviders/`
**Current Location:** `internal/services/{hetzner,cloudflare}.go` + `internal/utils/{hetzner,cloudflare}.go`
**Rationale:** Generic REST API clients for cloud providers

**Proposed Structure:**
```
pkg/cloudproviders/
├── hetzner/
│   ├── client.go          # Core Hetzner API client
│   ├── servers.go         # Server lifecycle management
│   ├── ssh_keys.go        # SSH key operations
│   ├── locations.go       # Datacenter locations
│   ├── types.go           # API response types
│   └── sorting.go         # Server type sorting utilities
├── cloudflare/
│   ├── client.go          # Core Cloudflare API client
│   ├── dns.go             # DNS zone management
│   ├── ssl.go             # SSL certificate operations
│   ├── kv.go              # KV namespace operations
│   └── types.go           # API response types
└── common/
    ├── http.go             # Common HTTP client patterns
    └── auth.go             # Common authentication patterns
```

**Migration Tasks:**
- Extract API client logic from services
- Consolidate utility functions from utils/ files
- Create consistent error handling across providers
- Add retry logic and rate limiting
- Implement proper API response caching

### Phase 3: High-Risk Extractions (Significant Refactoring)

#### 5. HTTP Response Patterns → `pkg/httputil/`
**Current Location:** `internal/utils/responses.go`
**Challenge:** Currently tightly coupled to Gin framework

**Proposed Structure:**
```
pkg/httputil/
├── responses.go           # Framework-agnostic response types
├── errors.go              # Standardized error responses
├── adapters/
│   ├── gin.go             # Gin framework adapter
│   ├── stdlib.go          # Standard library adapter
│   └── interface.go       # Common interface
└── middleware/
    └── cors.go             # Generic CORS middleware
```

**Migration Tasks:**
- Abstract response patterns from Gin-specific code
- Create adapter pattern for different HTTP frameworks
- Standardize error response formats
- Add response validation and sanitization

#### 6. Storage Interface → `pkg/storage/`
**Current Location:** `internal/services/kv.go`
**Rationale:** Generic key-value storage interface

**Proposed Structure:**
```
pkg/storage/
├── interface.go           # Generic storage interface
├── providers/
│   ├── cloudflare_kv.go   # Cloudflare KV implementation
│   ├── redis.go           # Redis implementation
│   └── memory.go          # In-memory for testing
└── encryption.go          # Encrypted storage wrapper
```

**Migration Tasks:**
- Define generic storage interface
- Implement Cloudflare KV as first provider
- Add encryption wrapper for secure storage
- Create in-memory provider for testing

## 🏗️ Architectural Improvements

### 1. Dependency Injection
**Current Issue:** Services instantiated directly in handlers
```go
// Current pattern
hetznerService := services.NewHetznerService()
```

**Proposed Solution:**
```go
// Dependency injection pattern
type HandlerDependencies struct {
    HetznerClient    cloudproviders.HetznerClient
    CloudflareClient cloudproviders.CloudflareClient
    SSHManager       ssh.Manager
    Storage          storage.Interface
}

func NewVPSHandler(deps HandlerDependencies) *VPSHandler {
    return &VPSHandler{deps: deps}
}
```

### 2. Interface Segregation
**Current Issue:** Concrete implementations throughout codebase

**Proposed Interfaces:**
```go
// Service interfaces
type VPSManager interface {
    CreateServer(ctx context.Context, req CreateServerRequest) (*Server, error)
    DeleteServer(ctx context.Context, serverID string) error
    GetServerStatus(ctx context.Context, serverID string) (*ServerStatus, error)
}

type DNSManager interface {
    ConfigureDNS(ctx context.Context, domain string, target string) error
    RemoveDNS(ctx context.Context, domain string) error
    ListDomains(ctx context.Context) ([]Domain, error)
}
```

### 3. Configuration Management
**Current Issue:** Hardcoded endpoints and timeouts

**Proposed Solution:**
```go
type Config struct {
    Server   ServerConfig
    Hetzner  HetznerConfig
    Cloudflare CloudflareConfig
    SSH      SSHConfig
}

type ServerConfig struct {
    Port         string        `env:"PORT" default:"8080"`
    ReadTimeout  time.Duration `env:"READ_TIMEOUT" default:"30s"`
    WriteTimeout time.Duration `env:"WRITE_TIMEOUT" default:"30s"`
}
```

### 4. Error Handling Standardization
**Current Issue:** Mixed error handling patterns

**Proposed Solution:**
```go
// Domain-specific error types
type VPSError struct {
    Type    VPSErrorType
    Message string
    Cause   error
}

type VPSErrorType string

const (
    VPSErrorTypeNotFound     VPSErrorType = "not_found"
    VPSErrorTypeInvalidState VPSErrorType = "invalid_state"
    VPSErrorTypeAPIError     VPSErrorType = "api_error"
)
```

## 📈 Implementation Roadmap

### Phase 1: Foundation (1-2 weeks)
1. Extract crypto utilities to `pkg/crypto/`
2. Extract network utilities to `pkg/netutil/`
3. Set up comprehensive testing for extracted packages
4. Update import paths in consuming code

### Phase 2: Core Infrastructure (2-3 weeks)
1. Refactor SSH management to `pkg/ssh/`
2. Extract cloud provider clients to `pkg/cloudproviders/`
3. Implement dependency injection pattern
4. Add configuration management system

### Phase 3: Advanced Patterns (2-3 weeks)
1. Abstract HTTP response patterns
2. Implement storage interface
3. Add comprehensive error handling
4. Complete interface segregation

### Phase 4: Testing & Documentation (1 week)
1. Achieve 80%+ test coverage for pkg/ components
2. Add comprehensive documentation
3. Create usage examples
4. Performance benchmarking

## 🎯 Expected Benefits

### Code Reusability
- Cloud provider clients usable in other infrastructure tools
- SSH management patterns applicable to any remote server tool
- Crypto utilities available for secure applications

### Maintainability
- Clear separation between generic and business-specific code
- Easier unit testing with dependency injection
- Consistent error handling across the application

### Performance
- Connection pooling for SSH and HTTP clients
- Caching for API responses
- Reduced memory allocation through interface usage

### Developer Experience
- Better IDE support with proper interfaces
- Clear dependency boundaries
- Easier onboarding for new developers

## ⚠️ Migration Considerations

### Backward Compatibility
- Maintain current API contracts during migration
- Use gradual migration approach with feature flags
- Comprehensive testing before each phase

### Dependencies
- Minimize external dependencies in pkg/ packages
- Use standard library where possible
- Document all third-party dependencies

### Performance Impact
- Benchmark before and after migration
- Monitor memory usage during interface adoption
- Ensure no performance regression in critical paths

### Testing Strategy
- Unit tests for all pkg/ components
- Integration tests for service interactions
- End-to-end tests for critical user workflows