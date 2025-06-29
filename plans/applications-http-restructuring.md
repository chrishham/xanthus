# Restructuring Plan for `/internal/handlers/applications/http.go`

## Problem Analysis
The current `http.go` file is **1,486 lines** and violates multiple architectural principles:
- **Mixed concerns**: HTTP handling + business logic + infrastructure operations
- **Code duplication**: Authentication, VPS setup, SSH connections repeated across methods  
- **Long methods**: Some methods exceed 100+ lines (e.g., `deployPredefinedApplication` ~165 lines)
- **Application-specific logic scattered**: code-server and ArgoCD specifics mixed throughout
- **Single responsibility violation**: One file handling 10+ different responsibilities

## Restructuring Strategy

### Phase 1: Extract Middleware & Common Logic ✅ COMPLETED
1. **Create `applications/middleware.go`** ✅:
   - Authentication middleware (eliminates repeated auth code)
   - VPS configuration middleware  
   - Common error handling middleware

2. **Create `applications/common.go`** ✅:
   - VPS connection helper functions
   - SSH setup utilities
   - Common validation logic

**Phase 1 Results:**
- Created `/internal/handlers/applications/middleware.go` (77 lines)
- Created `/internal/handlers/applications/common.go` (236 lines)
- Extracted common authentication patterns into reusable middleware
- Implemented VPS connection helpers with proper error handling
- Added validation helpers for application data and passwords
- All unit tests passing, build successful

### Phase 2: Service Layer Enhancement  
3. **Create new services**:
   - `services/application_service.go`: Core CRUD operations
   - `services/application_deployment_service.go`: Deployment orchestration
   - `services/password_service.go`: Password management across app types
   - `services/deployment_strategy.go`: Interface for app-specific deployments

4. **Create application-specific services**:
   - `services/codeserver_service.go`: Code-server deployment & password logic
   - `services/argocd_service.go`: ArgoCD deployment & password logic

### Phase 3: HTTP Handler Refactoring
5. **Refactor `http.go`** (reduce from 1,486 to ~400 lines):
   - Keep only HTTP request/response handling
   - Delegate all business logic to services
   - Use middleware for common operations
   - Break long methods into focused functions

6. **Create application-specific handlers**:
   - `applications/codeserver_handlers.go`: Code-server HTTP endpoints
   - `applications/argocd_handlers.go`: ArgoCD HTTP endpoints

### Phase 4: Configuration & Constants
7. **Create `applications/config.go`**:
   - Configuration structs
   - Constants to replace hardcoded values
   - Application-specific configuration

## Expected Outcomes
- **Maintainability**: Smaller, focused files (~200-400 lines each)
- **Testability**: Services can be unit tested independently
- **Extensibility**: New application types can be added easily
- **Consistency**: Leverages existing service factory pattern
- **Readability**: Clear separation of concerns

## File Structure After Refactoring
```
internal/handlers/applications/
├── base.go (existing - no changes)
├── http.go (refactored - ~400 lines)
├── middleware.go (new)
├── common.go (new) 
├── codeserver_handlers.go (new)
├── argocd_handlers.go (new)
└── config.go (new)

internal/services/
├── application_service.go (new)
├── application_deployment_service.go (new)
├── password_service.go (new)
├── deployment_strategy.go (new)
├── codeserver_service.go (new)
└── argocd_service.go (new)
```

This approach maintains architectural consistency with the existing service factory pattern while solving the immediate monolithic file problem.