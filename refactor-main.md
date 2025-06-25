# Refactoring Plan for `cmd/xanthus/main.go`

## Current State Analysis

- **File Size**: 3,120 lines (~33,000 tokens)
- **Functions**: 72 total (43 HTTP handlers, 29 utilities)
- **HTTP Routes**: 40+ routes
- **Struct Definitions**: 14 types
- **Main Issues**: Monolithic structure, mixed concerns, repeated patterns

## Refactoring Strategy

### Phase 1: Extract Core Structure

#### 1.1 Create Handler Packages
Create `internal/handlers/` directory with domain-specific handlers:

- **`auth.go`** - Authentication handlers (5 functions)
  - `handleRoot`, `handleLoginPage`, `handleLogin`, `handleLogout`, `handleHealth`

- **`dns.go`** - DNS management (4 functions)  
  - `handleDNSConfigure`, `handleDNSRemove`, `fetchCloudflareDomains`

- **`vps.go`** - VPS operations (20+ functions)
  - All VPS creation, deletion, power management handlers
  - SSH key management handlers
  - Status monitoring and configuration handlers

- **`applications.go`** - Helm/application management (6 functions)
  - Repository management handlers
  - Application lifecycle handlers (create, upgrade, delete)

#### 1.2 Extract Models
- **`internal/models/types.go`** - Move all 14 struct definitions
  - Cloudflare types (CloudflareResponse, KVNamespace, etc.)
  - Hetzner types (HetznerLocation, HetznerServerType, etc.)
  - Application types

#### 1.3 Create Utility Packages
Create `internal/utils/` directory:

- **`responses.go`** - Common JSON response helpers
  - Standardize the 211 `gin.H{}` responses
  - Success/error response templates

- **`cloudflare.go`** - Cloudflare API utilities (7 functions)
  - `verifyCloudflareToken`, `checkKVNamespaceExists`, `createKVNamespace`
  - `putKVValue`, `getXanthusNamespaceID`, `getKVValue`

- **`hetzner.go`** - Hetzner utilities (15+ functions)
  - API validation and data fetching
  - 9 sorting functions for server types
  - `fetchHetznerLocations`, `fetchHetznerServerTypes`, `fetchServerAvailability`

- **`crypto.go`** - Encryption functions
  - `encryptData`, `decryptData`

- **`server.go`** - Server utilities
  - `findAvailablePort`

### Phase 2: Middleware & Route Organization

#### 2.1 Extract Middleware
- **`internal/middleware/auth.go`** - Authentication middleware
  - Token validation logic
  - Remove repeated authentication checks from handlers

#### 2.2 Route Organization
- **`internal/router/routes.go`** - Route registration with grouping
  - Group routes by domain (auth, dns, vps, apps)
  - Clean route registration functions

#### 2.3 Reduce Main Function
- Reduce `main.go` to ~100 lines
  - Server setup and configuration only
  - Route registration delegation
  - Remove all business logic

### Phase 3: Pattern Standardization

#### 3.1 Response Standardization
- Create helper functions to eliminate repetition:
  - 211 instances of `gin.H{}` responses
  - 33 instances of success responses
  - 79 instances of error responses

#### 3.2 Error Handling
- Standardize error handling patterns across all handlers
- Create common error response functions
- Consistent logging patterns

#### 3.3 Service Integration
- Clean up the 42 service calls to `internal/services`
- Ensure consistent service integration patterns

## Implementation Steps

### Step 1: Preparation
1. Create directory structure:
   ```
   internal/
   ├── handlers/
   ├── middleware/
   ├── models/
   ├── router/
   └── utils/
   ```

### Step 2: Extract Types (Lowest Risk)
1. Move all struct definitions to `internal/models/types.go`
2. Update imports in main.go
3. Test compilation

### Step 3: Extract Utilities (Medium Risk)
1. Extract utility functions to appropriate utils files
2. Update imports and function calls
3. Test each utility group

### Step 4: Extract Handlers (High Risk)
1. Move handlers to domain-specific files
2. Create handler structs with dependencies
3. Update route registrations

### Step 5: Create Middleware & Router
1. Extract authentication middleware
2. Create route grouping functions
3. Minimize main.go to essentials

### Step 6: Standardize Patterns
1. Create response helpers
2. Standardize error handling
3. Clean up remaining duplications

## Expected Benefits

### Token Reduction
- **Current**: ~33,000 tokens in single file
- **After**: ~5,000 tokens per focused file
- **Main.go**: Reduced by 95% to ~100 lines

### Maintainability Improvements
- Clear separation of concerns
- Domain-specific code organization
- Easier to locate and modify specific functionality

### Development Speed
- Find/modify code in seconds vs minutes
- Focused context for each feature area
- Better IDE support and navigation

### Testing Benefits
- Isolated units for better test coverage
- Easier to mock dependencies
- Cleaner test organization

### Code Reusability
- Shared utilities across handlers
- Standardized patterns
- Better abstraction layers

## Final File Structure

```
cmd/xanthus/main.go                 (~100 lines)
internal/
├── handlers/
│   ├── auth.go                     (~200 lines)
│   ├── dns.go                      (~150 lines)
│   ├── vps.go                      (~800 lines)
│   └── applications.go             (~250 lines)
├── middleware/
│   └── auth.go                     (~50 lines)
├── models/
│   └── types.go                    (~100 lines)
├── router/
│   └── routes.go                   (~100 lines)
└── utils/
    ├── responses.go                (~100 lines)
    ├── cloudflare.go               (~200 lines)
    ├── hetzner.go                  (~400 lines)
    ├── crypto.go                   (~50 lines)
    └── server.go                   (~30 lines)
```

**Total**: 8-10 focused files vs 1 monolithic file, with significant improvements in maintainability and development velocity.