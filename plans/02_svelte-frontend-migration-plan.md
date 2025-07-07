# 🚀 Complete Migration Plan: HTMX to Svelte SPA

Based on comprehensive analysis of the current codebase, here's a detailed plan to complete the migration from HTMX to Svelte SPA with full client-side routing.

## 📋 Migration Overview

**Current Status**: Phase 1 Complete - JWT Authentication System ✅
**Migration Type**: Primarily Alpine.js → Svelte (minimal HTMX usage)
**Phase 1 Achievement**: Complete JWT authentication foundation with token management

---

## ✅ Phase 1: JWT Authentication System (COMPLETED)

### 1.1 JWT Implementation (Mandatory)
```go
// JWT-based authentication for SPA
type JWTService struct {
    secretKey     []byte
    tokenDuration time.Duration
}

func (j *JWTService) GenerateToken(userID string) (string, error) {
    // Generate JWT token with user claims
}

func (j *JWTService) ValidateToken(tokenString string) (*Claims, error) {
    // Validate and parse JWT token
}
```

**Tasks:** ✅ COMPLETED
- ✅ Implement JWT token generation and validation
- ✅ Create JWT middleware for API protection
- ✅ Add token refresh mechanism
- ✅ Implement secure token storage in frontend
- ✅ Add logout token invalidation

**Implementation Details:**
- JWT Service: `internal/services/jwt.go` with 15min access tokens, 7-day refresh tokens
- JWT Middleware: `internal/middleware/jwt_middleware.go` for API, HTML, and WebSocket auth
- Secure 32-byte secret key generation in `main.go`

### 1.2 Authentication Flow Update
```go
// Updated authentication middleware
func JWTAuthMiddleware(c *gin.Context) {
    authHeader := c.GetHeader("Authorization")
    if authHeader == "" {
        c.JSON(401, gin.H{"error": "Authorization header required"})
        c.Abort()
        return
    }
    
    tokenString := strings.TrimPrefix(authHeader, "Bearer ")
    claims, err := jwtService.ValidateToken(tokenString)
    if err != nil {
        c.JSON(401, gin.H{"error": "Invalid token"})
        c.Abort()
        return
    }
    
    c.Set("userID", claims.UserID)
    c.Next()
}
```

**Tasks:** ✅ COMPLETED
- ✅ Update login endpoint to return JWT tokens
- ✅ Implement token-based authentication for all API routes
- ✅ Add proper error handling for expired/invalid tokens
- ✅ Create token refresh endpoint
- ✅ Update Svelte auth store to handle JWT tokens

**Implementation Details:**
- Auth Handler: `internal/handlers/auth.go` with JWT API endpoints
- Routes: `internal/router/routes.go` with public/protected API groups
- Svelte Store: `svelte-app/src/lib/stores/auth.ts` with localStorage token management

### 1.3 Critical API Endpoints (Authentication-dependent)
```bash
# Priority API endpoints for JWT integration
/api/auth/login     → JWT token generation
/api/auth/refresh   → Token refresh
/api/auth/logout    → Token invalidation
/api/user/profile   → User information
```

**Tasks:** ✅ COMPLETED
- ✅ Implement core authentication API endpoints
- ✅ Add proper HTTP status codes and error responses  
- ✅ Test JWT flow with existing Svelte components
- ✅ Ensure backward compatibility during transition

**API Endpoints Implemented:**
- `POST /api/auth/login` - JWT token generation from Cloudflare token
- `POST /api/auth/refresh` - Token refresh with 5-minute auto-refresh
- `POST /api/auth/logout` - Token invalidation
- `GET /api/user/profile` - Authentication status check

---

## 🎯 Phase 2: Incremental API Migration (High Priority)

### 2.1 Applications Module API Migration
```bash
# Applications API endpoints (Week 2)
/applications/           → /api/applications/
/applications/create     → /api/applications/create
/applications/delete     → /api/applications/delete
/applications/passwords  → /api/applications/passwords
```

**Tasks:**
- Migrate applications handlers to return JSON
- Update Svelte applications store to use JWT authentication
- Test applications CRUD operations with new API
- Add proper error handling and loading states

### 2.2 VPS Module API Migration
```bash
# VPS API endpoints (Week 3)
/vps/create    → /api/vps/create
/vps/delete    → /api/vps/delete
/vps/manage    → /api/vps/manage
/vps/terminal  → /api/vps/terminal (WebSocket)
```

**Tasks:**
- Migrate VPS handlers to JSON responses
- Update WebSocket terminal authentication to use JWT
- Test VPS management workflows
- Ensure proper cleanup of WebSocket connections

### 2.3 DNS Module API Migration
```bash
# DNS API endpoints (Week 4)
/dns/configure → /api/dns/configure
/dns/domains   → /api/dns/domains
/dns/records   → /api/dns/records
```

**Tasks:**
- Migrate DNS configuration handlers
- Update DNS management components
- Test domain and record management
- Validate Cloudflare API integration

### 2.4 Setup Module API Migration
```bash
# Setup API endpoints (Week 5)
/setup/hetzner     → /api/setup/hetzner
/setup/cloudflare  → /api/setup/cloudflare
/setup/status      → /api/setup/status
```

**Tasks:**
- Migrate setup handlers to JSON responses
- Update setup workflow in Svelte
- Test initial configuration flow
- Ensure proper validation and error handling

---

## 🎯 Phase 3: Complete Svelte SPA Implementation (Medium Priority)

### 3.1 Build System Integration
```makefile
# Update Makefile to include Svelte build
build: build-assets build-svelte
	go build -o bin/xanthus cmd/xanthus/main.go

build-svelte:
	cd svelte-app && npm run build

build-assets: build-svelte
	# Existing CSS/JS build + Svelte build
```

**Tasks:**
- Integrate Svelte build into `make build` and `make assets`
- Update deployment scripts to include Svelte build
- Ensure proper asset versioning and caching
- Add development build optimization

### 3.2 Complete Missing Svelte Features

#### Applications Module
- [ ] Complete port forwarding modal functionality
- [ ] Implement version upgrade system
- [ ] Add application deletion with confirmation
- [ ] Fix password management edge cases

#### Terminal Integration
- [ ] Complete WebSocket terminal implementation with JWT auth
- [ ] Fix session management and cleanup
- [ ] Add proper error handling and reconnection
- [ ] Implement terminal history and commands

#### Type Safety & Validation
- [ ] Expand TypeScript definitions to match backend models
- [ ] Add form validation throughout the application
- [ ] Implement proper API response validation
- [ ] Add runtime type checking for API responses

---

## 🎯 Phase 4: HTMX Removal & Legacy Cleanup (Low Priority)

### 4.1 Remove HTMX Dependencies
```html
<!-- Remove from templates -->
<script src="https://unpkg.com/htmx.org@1.9.10"></script>
```

**Tasks:**
- Remove HTMX script tags from templates
- Replace HTMX form submissions with Svelte forms
- Convert loading states to Svelte reactive stores
- Update error handling to use Svelte notifications

### 4.2 Complete Client-Side Routing
```javascript
// Update Go routes to redirect to SPA
router.GET("/", func(c *gin.Context) {
    c.Redirect(302, "/app/")
})

// Remove legacy page routes
// router.GET("/applications", ...)  → DELETE
// router.GET("/vps", ...)           → DELETE
// router.GET("/dns", ...)           → DELETE
```

**Tasks:**
- Redirect all legacy routes to Svelte SPA equivalent
- Update navigation components to use Svelte routing
- Implement proper URL structure for SPA
- Add browser history management

### 4.3 Legacy Template Cleanup
```bash
# Remove templates no longer needed
rm web/templates/main.html
rm web/templates/applications.html
rm web/templates/vps-manage.html
rm web/templates/dns-config.html
```

**Tasks:**
- Remove unused Go templates
- Clean up template handler code
- Remove Alpine.js JavaScript modules
- Update static file serving

---

## 🎯 Phase 5: Performance & Polish (Low Priority)

### 5.1 Performance Optimization
- [ ] Implement proper loading states and skeleton loaders
- [ ] Optimize bundle size with code splitting
- [ ] Add service worker for offline functionality
- [ ] Implement proper caching strategies

### 5.2 User Experience Enhancements
- [ ] Add keyboard shortcuts for power users
- [ ] Improve mobile responsiveness
- [ ] Add dark mode support
- [ ] Implement advanced filtering and search

---

## 📅 Implementation Timeline (6-8 Weeks)

### ✅ Week 1: JWT Authentication Foundation (COMPLETED)
- ✅ **Days 1-2**: Implement JWT service and token generation
- ✅ **Days 3-4**: Create JWT middleware and authentication flow
- ✅ **Days 5-6**: Update login/logout endpoints for JWT
- ✅ **Day 7**: Test JWT integration with existing Svelte auth store

**Delivered:**
- Complete JWT authentication system with secure token management
- API endpoints: `/api/auth/login`, `/api/auth/refresh`, `/api/auth/logout`, `/api/user/profile`
- Svelte auth store with automatic token refresh and localStorage persistence
- JWT middleware for API protection and WebSocket authentication

### Week 2: Applications Module API Migration
- **Days 1-2**: Migrate applications handlers to JSON responses
- **Days 3-4**: Update Svelte applications components for JWT
- **Days 5-6**: Test applications CRUD operations
- **Day 7**: Fix any issues and add proper error handling

### Week 3: VPS Module API Migration
- **Days 1-2**: Migrate VPS handlers to JSON responses
- **Days 3-4**: Update WebSocket terminal authentication
- **Days 5-6**: Test VPS management workflows
- **Day 7**: Ensure proper cleanup and error handling

### Week 4: DNS Module API Migration
- **Days 1-2**: Migrate DNS configuration handlers
- **Days 3-4**: Update DNS management components
- **Days 5-6**: Test domain and record management
- **Day 7**: Validate Cloudflare API integration

### Week 5: Setup Module API Migration
- **Days 1-2**: Migrate setup handlers to JSON responses
- **Days 3-4**: Update setup workflow in Svelte
- **Days 5-6**: Test initial configuration flow
- **Day 7**: Comprehensive testing of setup process

### Week 6: Complete Svelte Implementation
- **Days 1-2**: Complete missing application features
- **Days 3-4**: Fix terminal integration with JWT
- **Days 5-6**: Add TypeScript definitions and validation
- **Day 7**: Build system integration and optimization

### Week 7: HTMX Removal & Cleanup
- **Days 1-2**: Remove HTMX dependencies
- **Days 3-4**: Complete client-side routing
- **Days 5-6**: Remove legacy templates and code
- **Day 7**: Final cleanup and documentation

### Week 8: Testing & Polish
- **Days 1-3**: Comprehensive testing of all features
- **Days 4-5**: Performance optimization and security audit
- **Days 6-7**: Final polish and deployment preparation

## 🔄 Rollback Strategy

### Phase-by-Phase Rollback
- **Phase 1**: Revert to cookie-based authentication
- **Phase 2**: Restore original API endpoints by module
- **Phase 3**: Re-enable HTMX/Alpine.js components
- **Phase 4**: Restore legacy templates if needed

### Git Strategy
- Create feature branches for each phase
- Merge only after thorough testing
- Tag stable versions for easy rollback
- Keep legacy code until migration is complete

---

## 🧪 Testing Strategy

### Frontend Testing
```bash
# Add to svelte-app/
npm run test        # Unit tests for components
npm run test:e2e    # End-to-end tests with Playwright
npm run test:int    # Integration tests for API calls
```

### Backend Testing
```bash
# Update existing tests
make test           # Ensure API endpoints work correctly
make test-e2e       # Test SPA integration
```

### Migration Testing
- [ ] Test all user workflows in Svelte SPA
- [ ] Verify API endpoint compatibility
- [ ] Test authentication flow
- [ ] Validate data consistency

---

## 🔧 Technical Considerations

### State Management
- Leverage existing Svelte stores (auth, VPS, applications, DNS)
- Add loading states for all async operations
- Implement proper error handling throughout

### API Design
- RESTful endpoints with consistent HTTP methods
- Proper status codes and error responses
- Pagination for large datasets
- Rate limiting and security headers

### Performance
- Code splitting for large pages
- Lazy loading for non-critical components
- Efficient state updates and reactivity
- Optimized build output

---

## 🎉 Success Criteria

### Technical Goals
- [ ] Zero HTMX dependencies remaining
- [ ] All routing handled by Svelte
- [ ] Complete API standardization
- [ ] Comprehensive TypeScript coverage
- [ ] Passing test suite

### User Experience Goals
- [ ] Faster page transitions (SPA benefits)
- [ ] Better error handling and user feedback
- [ ] Improved mobile responsiveness
- [ ] Consistent UI components

### Development Goals
- [ ] Simpler codebase maintenance
- [ ] Better developer experience
- [ ] Improved build process
- [ ] Enhanced debugging capabilities

This migration plan will transform Xanthus into a modern, fully client-side Svelte SPA while maintaining all existing functionality and improving the overall user experience.

## 📊 Current Implementation Analysis

### HTMX Usage (Minimal - Easy Migration)
- **Setup pages**: Form submissions (`hx-post="/setup/hetzner"`)
- **Login form**: Authentication with loading states
- **Error handling**: Inline error display (`hx-target="#error-message"`)

### Alpine.js Components (Primary Migration Target)
- **Applications management**: Complex state management with auto-refresh
- **VPS management**: CRUD operations with modal management
- **DNS configuration**: Domain management with SweetAlert2 modals

### Svelte SPA Status (70-75% Complete)
- ✅ Core architecture and routing
- ✅ State management stores
- ✅ Component library and UI
- ✅ API client integration
- ✅ Auto-refresh system
- ⚠️ Authentication integration needs work
- ⚠️ Terminal functionality incomplete
- ❌ Some API endpoints need standardization

### Migration Recommendation
The current implementation is well-positioned for completion. The heavy lifting has been done with the Svelte foundation, and the remaining work focuses on API standardization and finishing incomplete features rather than fundamental architectural changes.

---

## 🎉 Phase 1 Completion Status (January 2025)

### ✅ JWT Authentication System - COMPLETE

**What was delivered:**
1. **Backend JWT Infrastructure**
   - JWT Service (`internal/services/jwt.go`) with 15-minute access tokens and 7-day refresh tokens
   - JWT Middleware (`internal/middleware/jwt_middleware.go`) for API, HTML, and WebSocket authentication
   - Updated Auth Handler (`internal/handlers/auth.go`) with JWT API endpoints
   - Route integration (`internal/router/routes.go` and `main.go`) with secure key generation

2. **Frontend JWT Integration**
   - Updated Svelte Auth Store (`svelte-app/src/lib/stores/auth.ts`) with complete JWT token management
   - Automatic token refresh 5 minutes before expiry
   - localStorage persistence for tokens
   - `authenticatedFetch()` helper for seamless API requests
   - `initializeAuth()` for automatic authentication restoration

3. **API Endpoints Ready**
   - `POST /api/auth/login` - Login with Cloudflare token, returns JWT tokens
   - `POST /api/auth/refresh` - Refresh access token using refresh token
   - `POST /api/auth/logout` - Logout and invalidate tokens
   - `GET /api/user/profile` - Get authenticated user information

4. **Security Features**
   - 32-byte cryptographically secure secret key generation
   - Bearer token authentication with proper error handling
   - Automatic token expiration and refresh handling
   - Secure token storage with expiration tracking

**Files Created/Modified:**
- `internal/services/jwt.go` (new)
- `internal/services/jwt_test.go` (new)
- `internal/middleware/jwt_middleware.go` (new)
- `internal/handlers/auth.go` (modified - added JWT endpoints)
- `internal/router/routes.go` (modified - added JWT routes)
- `main.go` (modified - JWT service integration)
- `svelte-app/src/lib/stores/auth.ts` (completely rewritten for JWT)
- `go.mod` (updated - added JWT dependency)

**Next Priority: Phase 2 - API Migration**
Ready to begin migrating existing handlers (Applications, VPS, DNS, Setup) to return JSON responses for the Svelte SPA, using the now-complete JWT authentication foundation.