# VPS Handler Refactoring Plan

## Overview
The `internal/handlers/vps.go` file is currently 1,283 lines and contains 25+ handler methods with significant code duplication and mixed responsibilities. This plan outlines a systematic refactoring to improve maintainability, readability, and follow established patterns in the codebase.

## Current Issues
- **File size**: 1,283 lines in a single file
- **Code duplication**: Repeated patterns for authentication, error handling, and service initialization
- **Mixed responsibilities**: Single file handles lifecycle, configuration, information retrieval, and metadata operations
- **Maintenance burden**: Changes require navigating through a very large file

## Proposed Structure

### 1. Split into Functional Areas
Break the monolithic file into 4 focused files under `internal/handlers/`:

#### `vps_lifecycle.go` (~300 lines)
**Responsibility**: VPS creation, deletion, and power management
- `HandleVPSCreate` - Creates new VPS with K3s setup
- `HandleVPSDelete` - Deletes VPS and cleans up configuration
- `HandleVPSPowerOff` - Powers off VPS instance
- `HandleVPSPowerOn` - Powers on VPS instance  
- `HandleVPSReboot` - Reboots VPS instance
- `performVPSAction` - Common power management logic

#### `vps_info.go` (~350 lines)
**Responsibility**: Information retrieval and monitoring
- `HandleVPSList` - Returns JSON list of VPS instances
- `HandleVPSStatus` - Gets VPS health status via SSH
- `HandleVPSLogs` - Fetches VPS logs via SSH
- `HandleVPSInfo` - Retrieves VPS info including ArgoCD credentials
- `HandleVPSSSHKey` - Returns SSH private key for VPS access
- `HandleVPSTerminal` - Creates web terminal session

#### `vps_config.go` (~350 lines)
**Responsibility**: Configuration and deployment operations
- `HandleVPSConfigure` - Configures VPS with SSL certificates
- `HandleVPSDeploy` - Deploys Kubernetes manifests
- `HandleVPSCheckKey` - Checks if Hetzner API key exists
- `HandleVPSValidateKey` - Validates and stores Hetzner API key
- `HandleSetupHetzner` - Configures Hetzner API key in setup

#### `vps_meta.go` (~300 lines)
**Responsibility**: Metadata and validation operations
- `HandleVPSManagePage` - Renders VPS management page
- `HandleVPSCreatePage` - Renders VPS creation page
- `HandleVPSServerOptions` - Fetches server types and locations with filtering
- `HandleVPSLocations` - Fetches available VPS locations
- `HandleVPSServerTypes` - Fetches server types for specific location
- `HandleVPSValidateName` - Validates VPS names against existing servers

### 2. Extract Common Patterns

#### Authentication Helper
Create `internal/utils/auth_helpers.go`:
```go
func ValidateTokenAndGetAccount(c *gin.Context) (token, accountID string, err error)
```

#### Server ID Helper
Add to `internal/utils/server.go`:
```go
func ParseServerID(serverIDStr string) (int, error)
```

#### Enhanced Error Responses
Extend `internal/utils/responses.go` with VPS-specific helpers:
```go
func JSONVPSNotFound(c *gin.Context)
func JSONHetznerKeyMissing(c *gin.Context)
```

### 3. Service Layer Improvements

#### VPS Service Enhancements
Add to `internal/services/vps.go` (new file):
```go
type VPSService struct {
    hetzner *HetznerService
    kv      *KVService
    ssh     *SSHService
}

func (s *VPSService) GetVPSWithCosts(token, accountID string, serverID int) (*EnhancedVPS, error)
func (s *VPSService) ValidateVPSAccess(token, accountID string, serverID int) (*VPSConfig, error)
```

#### Cost Calculation Service
Extract cost calculation logic into dedicated methods:
```go
func (s *KVService) EnhanceServersWithCosts(token, accountID string, servers []HetznerServer) error
```

### 4. Shared Handler Base

Create `internal/handlers/base.go`:
```go
type BaseHandler struct {
    vpsService    *services.VPSService
    hetznerService *services.HetznerService
    kvService     *services.KVService
    sshService    *services.SSHService
}

func (h *BaseHandler) validateTokenAndAccount(c *gin.Context) (string, string, error)
func (h *BaseHandler) getVPSConfig(c *gin.Context, serverID int) (*services.VPSConfig, error)
```

## Implementation Steps

### Phase 1: Extract Common Patterns
1. Create authentication helper functions
2. Create server ID parsing helper
3. Enhance error response utilities
4. Create base handler with shared dependencies

### Phase 2: Create Service Layer
1. Create VPS service for business logic
2. Extract cost calculation methods
3. Create VPS access validation methods

### Phase 3: Split Handler File
1. Create the 4 new handler files
2. Move methods to appropriate files
3. Update imports and dependencies
4. Ensure all methods use new helpers

### Phase 4: Update Router
1. Update `internal/router/routes.go` to use new handlers
2. Ensure all routes are properly mapped
3. Test all endpoints

### Phase 5: Cleanup
1. Remove original `vps.go` file
2. Update any remaining imports
3. Run tests to ensure functionality

## Benefits

1. **Maintainability**: Smaller, focused files are easier to navigate and modify
2. **Reusability**: Common patterns extracted into reusable utilities
3. **Testability**: Smaller units are easier to test in isolation
4. **Consistency**: Follows established patterns in the codebase
5. **Performance**: Reduced cognitive load when working on specific VPS functionality

## Risk Mitigation

1. **Gradual migration**: Implement in phases to ensure stability
2. **Comprehensive testing**: Test each endpoint after refactoring
3. **Backup strategy**: Maintain original file until refactoring is complete
4. **Import management**: Carefully track and update all import statements

## Success Criteria

- [ ] Original 1,283-line file split into 4 files of 200-400 lines each
- [ ] All VPS endpoints continue to function correctly
- [ ] Code duplication reduced by >70%
- [ ] Authentication logic centralized
- [ ] Service layer properly abstracts business logic
- [ ] Router properly configured for new structure