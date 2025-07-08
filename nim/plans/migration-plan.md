# Xanthus Migration Plan: Go → Nim (Jester) + SvelteJS

## Overview

This document outlines the comprehensive migration plan for transitioning the Xanthus infrastructure management platform from Go + HTMX/Alpine.js to Nim (Jester) + SvelteJS with JWT authentication.

## Migration Goals

- **Backend**: Migrate from Go to Nim using Jester web framework
- **Frontend**: Migrate from HTMX/Alpine.js to SvelteJS
- **Authentication**: Implement JWT-based authentication system
- **Testing**: Integrate Jest for comprehensive testing
- **Architecture**: Maintain Handler-Service-Model (HSM) pattern
- **Functionality**: Preserve all existing features and integrations

## Phase 1: Infrastructure Setup & Project Structure

### 1.1 Directory Structure
```
nim/
├── plans/                          # Migration documentation
│   └── migration-plan.md          # This file
├── backend/                        # Nim backend with Jester
│   ├── src/
│   │   ├── handlers/              # HTTP request handlers
│   │   ├── services/              # Business logic services
│   │   ├── models/                # Data models and types
│   │   ├── middleware/            # JWT auth, CORS, etc.
│   │   ├── utils/                 # Utility functions
│   │   └── main.nim               # Application entry point
│   ├── tests/                     # Jest test files
│   ├── config/                    # Configuration files
│   ├── xanthus.nimble            # Nim package configuration
│   └── README.md                  # Backend documentation
├── frontend/                       # SvelteJS frontend
│   ├── src/
│   │   ├── lib/                   # Svelte components
│   │   ├── routes/                # SvelteKit routes
│   │   ├── stores/                # Svelte stores for state
│   │   ├── utils/                 # Utility functions
│   │   └── app.html               # Main app template
│   ├── static/                    # Static assets
│   ├── tests/                     # Frontend tests
│   ├── package.json               # Node.js dependencies
│   └── README.md                  # Frontend documentation
└── shared/                         # Shared types and utilities
    ├── types/                     # TypeScript type definitions
    └── api/                       # API schemas and contracts
```

### 1.2 Development Environment Setup

**Nim Backend Dependencies:**
```nim
# xanthus.nimble
requires "nim >= 1.6.0"
requires "jester >= 0.5.0"
requires "jwt >= 0.2.0"
requires "asynctools >= 0.1.0"
requires "httpx >= 0.3.0"
requires "jsony >= 1.1.0"
requires "redis >= 0.3.0"
requires "sqlite3 >= 0.1.0"
requires "argparse >= 4.0.0"
```

**SvelteJS Frontend Dependencies:**
```json
{
  "devDependencies": {
    "@sveltejs/adapter-auto": "^3.0.0",
    "@sveltejs/kit": "^2.0.0",
    "@sveltejs/vite-plugin-svelte": "^3.0.0",
    "autoprefixer": "^10.4.0",
    "postcss": "^8.4.0",
    "svelte": "^4.0.0",
    "tailwindcss": "^3.4.0",
    "typescript": "^5.0.0",
    "vite": "^5.0.0"
  },
  "dependencies": {
    "jwt-decode": "^4.0.0",
    "socket.io-client": "^4.7.0",
    "xterm": "^5.3.0"
  }
}
```

## Phase 2: Backend Migration (Nim + Jester)

### 2.1 Core Framework Migration & Route Responsibility Definition

**CRITICAL: Clear Route Responsibility Separation**

**Backend (Nim/Jester) - API Routes Only:**
```nim
# main.nim - Backend API Server (Port 8080)
import jester, asyncdispatch, json, jwt
import handlers/[auth, applications, vps, dns, terminal]
import middleware/[auth_middleware, cors_middleware, websocket_middleware]
import services/service_container

proc setupRoutes(app: Jester) =
  # Public API routes
  app.get("/health", healthCheck)
  app.post("/api/auth/login", loginHandler)
  app.post("/api/auth/validate", validateCloudflareToken)
  
  # Protected API routes (JWT + Cloudflare validation)
  app.group("/api/v1", authMiddleware):
    # Applications API (JSON only)
    app.get("/applications", getApplicationsAPI)
    app.post("/applications", createApplicationAPI)
    app.delete("/applications/@id", deleteApplicationAPI)
    app.post("/applications/@id/restart", restartApplicationAPI)
    
    # VPS API (JSON only)
    app.get("/vps", getVPSListAPI)
    app.post("/vps", createVPSAPI)
    app.delete("/vps/@id", deleteVPSAPI)
    app.post("/vps/@id/power", powerVPSAPI)
    
    # DNS API (JSON only)
    app.get("/dns/domains", getDNSDomainsAPI)
    app.post("/dns/configure", configureDNSAPI)
    app.delete("/dns/@id", removeDNSAPI)
    
    # Terminal API (JSON only)
    app.get("/terminal/sessions", getTerminalSessionsAPI)
    app.post("/terminal/sessions", createTerminalSessionAPI)
    app.delete("/terminal/sessions/@id", endTerminalSessionAPI)
  
  # WebSocket endpoints (requires special handling)
  app.get("/ws/terminal/@sessionId", handleTerminalWebSocket)
  app.get("/ws/status", handleStatusWebSocket)

when isMainModule:
  let port = 8080
  let settings = newSettings(port=Port(port))
  var jester = initJester(settings)
  jester.setupRoutes()
  runForever()
```

**Frontend (SvelteKit) - Web Routes & Static Files:**
```javascript
// Frontend routes structure (Port 3000)
src/routes/
├── +layout.svelte                    # Root layout with auth check
├── +page.svelte                      # Dashboard/main page
├── auth/
│   ├── login/+page.svelte           # Login page
│   └── logout/+page.svelte          # Logout handler
├── applications/
│   ├── +page.svelte                 # Applications list page
│   ├── create/+page.svelte          # Create application page
│   └── [id]/+page.svelte            # Application details
├── vps/
│   ├── +page.svelte                 # VPS list page
│   ├── create/+page.svelte          # Create VPS page
│   └── [id]/+page.svelte            # VPS details
├── dns/
│   ├── +page.svelte                 # DNS management page
│   └── configure/+page.svelte       # DNS configuration
├── terminal/
│   └── [sessionId]/+page.svelte     # Terminal interface
└── api/                             # SvelteKit API routes (proxy to backend)
    ├── auth/
    │   └── +server.js               # Proxy auth requests
    ├── applications/
    │   └── +server.js               # Proxy application requests
    └── vps/
        └── +server.js               # Proxy VPS requests
```

**Route Responsibility Matrix:**

| Route Type | Backend (Nim/Jester) | Frontend (SvelteKit) |
|------------|---------------------|---------------------|
| **Authentication** | JWT validation, Cloudflare token verification | Login UI, logout handling, token storage |
| **Applications API** | Business logic, Helm deployments, KV storage | User interface, form handling, state management |
| **VPS API** | Provider integrations, server management | User interface, status display, controls |
| **DNS API** | Cloudflare DNS operations | Configuration UI, domain management |
| **Terminal** | WebSocket connections, SSH proxy | Terminal UI, session management |
| **Static Files** | None (API only) | All static assets, CSS, JavaScript |
| **Health/Monitoring** | System health checks | UI health indicators |
| **WebSocket** | Real-time data streaming | Client-side WebSocket handling |

### 2.2 Handler Migration Strategy

**Handler Structure (Handler-Service-Model):**
```nim
# handlers/applications.nim
import jester, json, asyncdispatch
import ../services/application_service
import ../models/application_model
import ../middleware/auth_middleware

proc getApplications*(request: Request): Future[ResponseData] {.async.} =
  let userInfo = request.getUserInfo()
  let appService = getApplicationService()
  
  try:
    let applications = await appService.listApplications(userInfo.accountId)
    return %* {
      "status": "success",
      "data": applications
    }
  except Exception as e:
    return %* {
      "status": "error", 
      "message": e.msg
    }

proc createApplication*(request: Request): Future[ResponseData] {.async.} =
  let userInfo = request.getUserInfo()
  let body = parseJson(request.body)
  let appData = body.to(ApplicationCreateRequest)
  
  let appService = getApplicationService()
  let result = await appService.createApplication(userInfo.accountId, appData)
  
  return %* {
    "status": "success",
    "data": result
  }
```

### 2.3 Service Layer Migration

**Service Architecture:**
```nim
# services/application_service.nim
import asyncdispatch, json, options
import ../models/application_model
import ../utils/[kv_client, helm_client, template_processor]

type
  ApplicationService* = ref object
    kvClient: KVClient
    helmClient: HelmClient
    templateProcessor: TemplateProcessor

proc newApplicationService*(kvClient: KVClient, helmClient: HelmClient): ApplicationService =
  ApplicationService(
    kvClient: kvClient,
    helmClient: helmClient,
    templateProcessor: newTemplateProcessor()
  )

proc listApplications*(service: ApplicationService, accountId: string): Future[seq[Application]] {.async.} =
  let appsJson = await service.kvClient.list(accountId, "app:")
  result = appsJson.mapIt(it.to(Application))

proc createApplication*(service: ApplicationService, accountId: string, appData: ApplicationCreateRequest): Future[Application] {.async.} =
  # Template processing
  let processedConfig = await service.templateProcessor.processTemplate(appData.config, appData.variables)
  
  # Create application record
  let app = Application(
    id: generateId(),
    name: appData.name,
    appType: appData.appType,
    config: processedConfig,
    status: "creating",
    createdAt: now()
  )
  
  # Store in KV
  await service.kvClient.store(accountId, "app:" & app.id, app)
  
  # Deploy via Helm
  asyncCheck service.deployApplication(app)
  
  return app
```

### 2.4 Model Migration

**Data Models:**
```nim
# models/application_model.nim
import json, times, options

type
  Application* = object
    id*: string
    name*: string
    description*: string
    appType*: string
    appVersion*: string
    subdomain*: string
    domain*: string
    vpsId*: string
    vpsName*: string
    namespace*: string
    status*: string
    errorMsg*: Option[string]
    url*: Option[string]
    createdAt*: DateTime
    updatedAt*: DateTime

  ApplicationCreateRequest* = object
    name*: string
    appType*: string
    subdomain*: string
    domain*: string
    vpsId*: string
    config*: JsonNode
    variables*: JsonNode

  ApplicationStatus* = enum
    Creating = "creating"
    Running = "running"
    Failed = "failed"
    Stopped = "stopped"
    Updating = "updating"
```

### 2.5 Authentication & JWT Integration

**CRITICAL: Complex Authentication Flow (Current Go Implementation Analysis)**

The current authentication system is far more complex than initially outlined. Here's the actual flow that must be preserved:

**Current Go Authentication Flow:**
1. **Cloudflare Token Validation** (with 10-minute cache)
2. **KV Namespace Creation/Verification** 
3. **CSR Generation and Storage**
4. **Account Setup and Verification**
5. **Context Storage** (token, account_id, namespace_id)

**Enhanced Authentication Service:**
```nim
# services/auth_service.nim
import asyncdispatch, jwt, json, times, httpclient, strformat
import ../models/user_model
import ../utils/[cloudflare_client, kv_client, crypto_utils]

type
  AuthService* = ref object
    cloudflareClient: CloudflareClient
    kvClient: KVClient
    jwtSecret: string
    tokenCache: Table[string, CachedAuthInfo]
  
  CachedAuthInfo* = object
    accountInfo: AccountInfo
    namespaceId: string
    cachedAt: DateTime
    
  CompleteAuthInfo* = object
    accountId: string
    email: string
    name: string
    token: string
    namespaceId: string
    csrGenerated: bool
    jwtToken: string

proc newAuthService*(cloudflareClient: CloudflareClient, kvClient: KVClient, jwtSecret: string): AuthService =
  AuthService(
    cloudflareClient: cloudflareClient,
    kvClient: kvClient,
    jwtSecret: jwtSecret,
    tokenCache: initTable[string, CachedAuthInfo]()
  )

proc authenticateUser*(service: AuthService, cloudflareToken: string): Future[CompleteAuthInfo] {.async.} =
  # Step 1: Check cache (10-minute expiry)
  if service.tokenCache.hasKey(cloudflareToken):
    let cached = service.tokenCache[cloudflareToken]
    if (now() - cached.cachedAt) < 10.minutes:
      return service.buildCompleteAuthInfo(cached, cloudflareToken)
  
  # Step 2: Validate Cloudflare token
  let accountInfo = await service.cloudflareClient.getAccountInfo(cloudflareToken)
  
  # Step 3: Setup/verify KV namespace
  let namespaceId = await service.ensureKVNamespace(accountInfo.id, cloudflareToken)
  
  # Step 4: Generate/verify CSR
  let csrGenerated = await service.ensureCSR(accountInfo.id, namespaceId, cloudflareToken)
  
  # Step 5: Cache the results
  service.tokenCache[cloudflareToken] = CachedAuthInfo(
    accountInfo: accountInfo,
    namespaceId: namespaceId,
    cachedAt: now()
  )
  
  # Step 6: Generate JWT
  let jwtToken = service.generateJWT(accountInfo, namespaceId)
  
  return CompleteAuthInfo(
    accountId: accountInfo.id,
    email: accountInfo.email,
    name: accountInfo.name,
    token: cloudflareToken,
    namespaceId: namespaceId,
    csrGenerated: csrGenerated,
    jwtToken: jwtToken
  )

proc ensureKVNamespace*(service: AuthService, accountId: string, token: string): Future[string] {.async.} =
  # Check if namespace exists
  let namespaces = await service.cloudflareClient.listKVNamespaces(token)
  let namespaceName = fmt"xanthus_{accountId}"
  
  for ns in namespaces:
    if ns.title == namespaceName:
      return ns.id
  
  # Create namespace if it doesn't exist
  let newNamespace = await service.cloudflareClient.createKVNamespace(token, namespaceName)
  return newNamespace.id

proc ensureCSR*(service: AuthService, accountId: string, namespaceId: string, token: string): Future[bool] {.async.} =
  # Check if CSR exists in KV
  let existingCSR = await service.kvClient.get(token, namespaceId, "csr")
  if existingCSR.isSome:
    return true
  
  # Generate new CSR
  let (privateKey, csr) = generateCSR()
  
  # Store in KV
  await service.kvClient.store(token, namespaceId, "csr", csr)
  await service.kvClient.store(token, namespaceId, "private_key", privateKey)
  
  return true

proc generateJWT*(service: AuthService, accountInfo: AccountInfo, namespaceId: string): string =
  let payload = %* {
    "accountId": accountInfo.id,
    "email": accountInfo.email,
    "name": accountInfo.name,
    "namespaceId": namespaceId,
    "exp": (now() + 24.hours).toTime().toUnix()
  }
  
  return jwt.encode(payload, service.jwtSecret)
```

**Enhanced JWT Middleware:**
```nim
# middleware/auth_middleware.nim
import jester, jwt, json, asyncdispatch, times
import ../services/auth_service
import ../models/user_model

type
  UserInfo* = object
    accountId*: string
    email*: string
    name*: string
    namespaceId*: string
    exp*: int64

proc authMiddleware*(request: Request, response: Response): Future[ResponseData] {.async.} =
  let authHeader = request.headers.getOrDefault("Authorization")
  
  if not authHeader.startsWith("Bearer "):
    return %* {
      "status": "error",
      "message": "Missing or invalid authorization header"
    }
  
  let token = authHeader[7..^1]
  
  try:
    let payload = jwt.decode(token, getJWTSecret())
    let userInfo = payload.to(UserInfo)
    
    # Verify token expiration
    if userInfo.exp < now().toTime().toUnix():
      return %* {
        "status": "error",
        "message": "Token expired"
      }
    
    # Store user info in request context
    request.setUserInfo(userInfo)
    
  except Exception as e:
    return %* {
      "status": "error",
      "message": "Invalid token"
    }

proc getUserInfo*(request: Request): UserInfo =
  request.ctx["userInfo"].to(UserInfo)
```

**Authentication Handlers:**
```nim
# handlers/auth.nim
import jester, json, asyncdispatch
import ../services/auth_service
import ../models/user_model

proc loginHandler*(request: Request): Future[ResponseData] {.async.} =
  let body = parseJson(request.body)
  let cloudflareToken = body["token"].getStr()
  
  try:
    let authService = getAuthService()
    let authInfo = await authService.authenticateUser(cloudflareToken)
    
    return %* {
      "status": "success",
      "data": {
        "jwt_token": authInfo.jwtToken,
        "account_id": authInfo.accountId,
        "email": authInfo.email,
        "name": authInfo.name
      }
    }
  except Exception as e:
    return %* {
      "status": "error",
      "message": e.msg
    }

proc validateCloudflareToken*(request: Request): Future[ResponseData] {.async.} =
  let body = parseJson(request.body)
  let cloudflareToken = body["token"].getStr()
  
  try:
    let authService = getAuthService()
    let authInfo = await authService.authenticateUser(cloudflareToken)
    
    return %* {
      "status": "success",
      "data": {
        "valid": true,
        "account_id": authInfo.accountId,
        "namespace_id": authInfo.namespaceId
      }
    }
  except Exception as e:
    return %* {
      "status": "error",
      "message": "Token validation failed"
    }
```

### 2.6 External Integration Migration

**Cloudflare Client:**
```nim
# utils/cloudflare_client.nim
import asyncdispatch, httpclient, json, strformat

type
  CloudflareClient* = ref object
    httpClient: AsyncHttpClient
    baseUrl: string

proc newCloudflareClient*(): CloudflareClient =
  CloudflareClient(
    httpClient: newAsyncHttpClient(),
    baseUrl: "https://api.cloudflare.com/client/v4"
  )

proc getAccountInfo*(client: CloudflareClient, token: string): Future[AccountInfo] {.async.} =
  let headers = newHttpHeaders([
    ("Authorization", "Bearer " & token),
    ("Content-Type", "application/json")
  ])
  
  let response = await client.httpClient.get(
    client.baseUrl & "/accounts",
    headers = headers
  )
  
  let data = parseJson(await response.body)
  return data["result"][0].to(AccountInfo)

proc createDNSRecord*(client: CloudflareClient, token: string, zoneId: string, record: DNSRecord): Future[DNSRecord] {.async.} =
  let headers = newHttpHeaders([
    ("Authorization", "Bearer " & token),
    ("Content-Type", "application/json")
  ])
  
  let response = await client.httpClient.post(
    client.baseUrl & fmt"/zones/{zoneId}/dns_records",
    headers = headers,
    body = $(%* record)
  )
  
  let data = parseJson(await response.body)
  return data["result"].to(DNSRecord)
```

## Phase 3: Frontend Migration (SvelteJS)

### 3.1 SvelteKit Setup

**Project Structure:**
```
frontend/src/
├── lib/                           # Reusable components
│   ├── components/               # UI components
│   │   ├── ApplicationCard.svelte
│   │   ├── VPSCard.svelte
│   │   ├── StatusBadge.svelte
│   │   └── LoadingSpinner.svelte
│   ├── stores/                   # Svelte stores
│   │   ├── auth.js              # Authentication state
│   │   ├── applications.js       # Application state
│   │   ├── vps.js               # VPS state
│   │   └── websocket.js         # WebSocket connections
│   └── utils/                    # Utility functions
│       ├── api.js               # API client
│       ├── jwt.js               # JWT handling
│       └── formatting.js        # Date/text formatting
├── routes/                       # SvelteKit routes
│   ├── (app)/                   # Protected routes
│   │   ├── applications/        # Application management
│   │   ├── vps/                 # VPS management
│   │   ├── dns/                 # DNS management
│   │   └── terminal/            # Terminal interface
│   ├── auth/                    # Authentication routes
│   │   ├── login/
│   │   └── logout/
│   └── +layout.svelte           # Root layout
└── app.html                     # Main HTML template
```

### 3.2 Authentication Integration

**JWT Store:**
```javascript
// lib/stores/auth.js
import { writable } from 'svelte/store';
import { browser } from '$app/environment';
import { jwtDecode } from 'jwt-decode';

function createAuthStore() {
  const { subscribe, set, update } = writable({
    token: null,
    user: null,
    isAuthenticated: false,
    isLoading: false
  });

  return {
    subscribe,
    
    async login(cloudflareToken) {
      update(state => ({ ...state, isLoading: true }));
      
      try {
        const response = await fetch('/api/auth/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ token: cloudflareToken })
        });
        
        if (!response.ok) throw new Error('Login failed');
        
        const data = await response.json();
        const decoded = jwtDecode(data.token);
        
        // Store token in localStorage
        if (browser) {
          localStorage.setItem('jwt_token', data.token);
        }
        
        set({
          token: data.token,
          user: decoded,
          isAuthenticated: true,
          isLoading: false
        });
        
        return true;
      } catch (error) {
        set({
          token: null,
          user: null,
          isAuthenticated: false,
          isLoading: false
        });
        throw error;
      }
    },
    
    logout() {
      if (browser) {
        localStorage.removeItem('jwt_token');
      }
      set({
        token: null,
        user: null,
        isAuthenticated: false,
        isLoading: false
      });
    },
    
    checkAuth() {
      if (!browser) return;
      
      const token = localStorage.getItem('jwt_token');
      if (!token) return;
      
      try {
        const decoded = jwtDecode(token);
        
        // Check if token is expired
        if (decoded.exp * 1000 < Date.now()) {
          this.logout();
          return;
        }
        
        set({
          token,
          user: decoded,
          isAuthenticated: true,
          isLoading: false
        });
      } catch (error) {
        this.logout();
      }
    }
  };
}

export const auth = createAuthStore();
```

### 3.3 API Client Integration

**API Client:**
```javascript
// lib/utils/api.js
import { auth } from '../stores/auth.js';
import { get } from 'svelte/store';

class ApiClient {
  constructor(baseUrl = '/api/v1') {
    this.baseUrl = baseUrl;
  }

  async request(endpoint, options = {}) {
    const authState = get(auth);
    
    const config = {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers
      },
      ...options
    };

    // Add JWT token if authenticated
    if (authState.isAuthenticated && authState.token) {
      config.headers['Authorization'] = `Bearer ${authState.token}`;
    }

    const response = await fetch(`${this.baseUrl}${endpoint}`, config);
    
    // Handle authentication errors
    if (response.status === 401) {
      auth.logout();
      throw new Error('Authentication required');
    }

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.message || 'Request failed');
    }

    return response.json();
  }

  // Convenience methods
  get(endpoint) {
    return this.request(endpoint);
  }

  post(endpoint, data) {
    return this.request(endpoint, {
      method: 'POST',
      body: JSON.stringify(data)
    });
  }

  put(endpoint, data) {
    return this.request(endpoint, {
      method: 'PUT',
      body: JSON.stringify(data)
    });
  }

  delete(endpoint) {
    return this.request(endpoint, {
      method: 'DELETE'
    });
  }
}

export const api = new ApiClient();
```

### 3.4 Component Migration

**Application Card Component:**
```svelte
<!-- lib/components/ApplicationCard.svelte -->
<script>
  import { createEventDispatcher } from 'svelte';
  import StatusBadge from './StatusBadge.svelte';
  import { api } from '../utils/api.js';
  
  export let application;
  
  const dispatch = createEventDispatcher();
  let loading = false;
  
  async function deleteApplication() {
    if (!confirm('Are you sure you want to delete this application?')) return;
    
    loading = true;
    try {
      await api.delete(`/applications/${application.id}`);
      dispatch('deleted', application.id);
    } catch (error) {
      alert('Failed to delete application: ' + error.message);
    } finally {
      loading = false;
    }
  }
  
  async function restartApplication() {
    loading = true;
    try {
      await api.post(`/applications/${application.id}/restart`);
      dispatch('restarted', application.id);
    } catch (error) {
      alert('Failed to restart application: ' + error.message);
    } finally {
      loading = false;
    }
  }
</script>

<div class="card-base p-6">
  <div class="flex items-center justify-between mb-4">
    <h3 class="text-lg font-semibold text-gray-900">{application.name}</h3>
    <StatusBadge status={application.status} />
  </div>
  
  <div class="space-y-2 text-sm text-gray-600 mb-4">
    <p><span class="font-medium">Type:</span> {application.appType}</p>
    <p><span class="font-medium">Version:</span> {application.appVersion}</p>
    <p><span class="font-medium">URL:</span> 
      {#if application.url}
        <a href={application.url} target="_blank" class="text-blue-600 hover:text-blue-800">
          {application.url}
        </a>
      {:else}
        <span class="text-gray-400">Not available</span>
      {/if}
    </p>
  </div>
  
  <div class="flex space-x-2">
    {#if application.status === 'running'}
      <button 
        class="button-secondary" 
        on:click={restartApplication}
        disabled={loading}
      >
        Restart
      </button>
    {/if}
    
    <button 
      class="button-danger" 
      on:click={deleteApplication}
      disabled={loading}
    >
      Delete
    </button>
  </div>
</div>

<style>
  .card-base {
    @apply bg-white rounded-lg shadow-md border hover:shadow-lg transition-shadow;
  }
  
  .button-secondary {
    @apply px-3 py-1.5 text-sm font-medium text-gray-700 bg-gray-200 rounded-md hover:bg-gray-300;
  }
  
  .button-danger {
    @apply px-3 py-1.5 text-sm font-medium text-white bg-red-600 rounded-md hover:bg-red-700;
  }
</style>
```

### 3.5 State Management

**Applications Store:**
```javascript
// lib/stores/applications.js
import { writable } from 'svelte/store';
import { api } from '../utils/api.js';

function createApplicationsStore() {
  const { subscribe, set, update } = writable({
    applications: [],
    loading: false,
    error: null
  });

  return {
    subscribe,
    
    async loadApplications() {
      update(state => ({ ...state, loading: true, error: null }));
      
      try {
        const response = await api.get('/applications');
        set({
          applications: response.data,
          loading: false,
          error: null
        });
      } catch (error) {
        set({
          applications: [],
          loading: false,
          error: error.message
        });
      }
    },
    
    async createApplication(applicationData) {
      try {
        const response = await api.post('/applications', applicationData);
        update(state => ({
          ...state,
          applications: [response.data, ...state.applications]
        }));
        return response.data;
      } catch (error) {
        throw error;
      }
    },
    
    async deleteApplication(id) {
      await api.delete(`/applications/${id}`);
      update(state => ({
        ...state,
        applications: state.applications.filter(app => app.id !== id)
      }));
    },
    
    updateApplication(id, updates) {
      update(state => ({
        ...state,
        applications: state.applications.map(app => 
          app.id === id ? { ...app, ...updates } : app
        )
      }));
    }
  };
}

export const applications = createApplicationsStore();
```

### 3.6 WebSocket Integration

**CRITICAL: WebSocket Implementation Challenges**

Based on Jester documentation review, there are **significant limitations** with WebSocket support that need addressing:

**Current Go WebSocket Implementation:**
- **Sophisticated terminal sessions** with session management
- **Real-time status updates** for applications and VPS
- **SSH proxy functionality** through WebSocket connections
- **Session persistence** and reconnection handling

**Jester WebSocket Limitations:**
- **Limited pattern matching** - "no wildcard patterns" in Jester
- **Basic WebSocket support** - May not handle complex routing
- **Session management** - No built-in session handling

**Enhanced WebSocket Strategy:**

**Option 1: Hybrid Approach (Recommended)**
```nim
# Use separate WebSocket server for complex terminal operations
# Backend handles API, separate Go microservice handles WebSocket
```

**Option 2: Enhanced Jester WebSocket Implementation**
```nim
# utils/websocket_manager.nim
import jester, asyncdispatch, json, tables, times
import ../services/terminal_service

type
  WebSocketManager* = ref object
    connections: Table[string, WebSocket]
    sessions: Table[string, TerminalSession]
    
  TerminalSession* = object
    id: string
    vpsId: string
    accountId: string
    createdAt: DateTime
    lastActivity: DateTime

proc newWebSocketManager*(): WebSocketManager =
  WebSocketManager(
    connections: initTable[string, WebSocket](),
    sessions: initTable[string, TerminalSession]()
  )

proc handleTerminalWebSocket*(manager: WebSocketManager, ws: WebSocket, sessionId: string) {.async.} =
  # Verify session exists
  if not manager.sessions.hasKey(sessionId):
    await ws.send(%* {"error": "Invalid session"})
    return
  
  let session = manager.sessions[sessionId]
  manager.connections[sessionId] = ws
  
  # Handle WebSocket messages
  while ws.readyState == Open:
    try:
      let message = await ws.receiveStrPacket()
      let data = parseJson(message)
      
      case data["type"].getStr():
        of "terminal_input":
          await manager.handleTerminalInput(sessionId, data["data"].getStr())
        of "resize":
          await manager.handleTerminalResize(sessionId, data["cols"].getInt(), data["rows"].getInt())
        of "ping":
          await ws.send(%* {"type": "pong"})
        else:
          await ws.send(%* {"error": "Unknown message type"})
    except Exception as e:
      echo "WebSocket error: ", e.msg
      break
  
  # Clean up connection
  manager.connections.del(sessionId)

proc handleTerminalInput*(manager: WebSocketManager, sessionId: string, input: string) {.async.} =
  let session = manager.sessions[sessionId]
  let terminalService = getTerminalService()
  
  # Send input to SSH session
  await terminalService.sendInput(session.vpsId, input)
  
  # Update last activity
  manager.sessions[sessionId].lastActivity = now()

proc broadcastToSession*(manager: WebSocketManager, sessionId: string, message: JsonNode) {.async.} =
  if manager.connections.hasKey(sessionId):
    let ws = manager.connections[sessionId]
    if ws.readyState == Open:
      await ws.send(message)
```

**Frontend WebSocket Store (Enhanced):**
```javascript
// lib/stores/websocket.js
import { writable } from 'svelte/store';
import { browser } from '$app/environment';
import { auth } from './auth.js';

function createWebSocketStore() {
  const { subscribe, set, update } = writable({
    connected: false,
    socket: null,
    messages: [],
    sessions: new Map()
  });

  let reconnectAttempts = 0;
  const maxReconnectAttempts = 5;
  const heartbeatInterval = 30000; // 30 seconds

  return {
    subscribe,
    
    connectTerminal(sessionId, vpsId) {
      if (!browser) return;
      
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${protocol}//${window.location.host}/ws/terminal/${sessionId}`;
      
      const socket = new WebSocket(wsUrl);
      
      socket.onopen = () => {
        console.log('Terminal WebSocket connected');
        reconnectAttempts = 0;
        
        // Start heartbeat
        const heartbeat = setInterval(() => {
          if (socket.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify({ type: 'ping' }));
          } else {
            clearInterval(heartbeat);
          }
        }, heartbeatInterval);
        
        update(state => ({ 
          ...state, 
          connected: true, 
          socket,
          sessions: state.sessions.set(sessionId, { vpsId, heartbeat })
        }));
      };
      
      socket.onmessage = (event) => {
        const message = JSON.parse(event.data);
        
        switch (message.type) {
          case 'terminal_output':
            this.handleTerminalOutput(sessionId, message.data);
            break;
          case 'pong':
            // Heartbeat response
            break;
          case 'error':
            this.handleError(sessionId, message.error);
            break;
          default:
            console.warn('Unknown WebSocket message:', message);
        }
      };
      
      socket.onclose = () => {
        console.log('Terminal WebSocket disconnected');
        
        // Clean up heartbeat
        update(state => {
          const session = state.sessions.get(sessionId);
          if (session && session.heartbeat) {
            clearInterval(session.heartbeat);
          }
          state.sessions.delete(sessionId);
          return { ...state, connected: false, socket: null };
        });
        
        // Attempt to reconnect
        if (reconnectAttempts < maxReconnectAttempts) {
          reconnectAttempts++;
          setTimeout(() => this.connectTerminal(sessionId, vpsId), 1000 * reconnectAttempts);
        }
      };
      
      socket.onerror = (error) => {
        console.error('Terminal WebSocket error:', error);
      };
    },
    
    sendTerminalInput(sessionId, input) {
      update(state => {
        if (state.socket && state.connected) {
          state.socket.send(JSON.stringify({
            type: 'terminal_input',
            data: input
          }));
        }
        return state;
      });
    },
    
    resizeTerminal(sessionId, cols, rows) {
      update(state => {
        if (state.socket && state.connected) {
          state.socket.send(JSON.stringify({
            type: 'resize',
            cols: cols,
            rows: rows
          }));
        }
        return state;
      });
    },
    
    disconnect(sessionId) {
      update(state => {
        if (state.socket) {
          state.socket.close();
        }
        const session = state.sessions.get(sessionId);
        if (session && session.heartbeat) {
          clearInterval(session.heartbeat);
        }
        state.sessions.delete(sessionId);
        return { connected: false, socket: null, messages: [] };
      });
    }
  };
}

export const websocket = createWebSocketStore();
```

**WebSocket Risk Mitigation:**
1. **Fallback Strategy**: Keep Go WebSocket service as fallback during migration
2. **Performance Testing**: Benchmark Nim WebSocket vs Go performance
3. **Session Management**: Implement robust session tracking and cleanup
4. **Error Handling**: Comprehensive error handling and reconnection logic
5. **Security**: Ensure proper authentication on WebSocket connections

## Phase 4: Testing Strategy

### 4.1 Backend Testing (Jest for Nim)

**Test Structure:**
```
backend/tests/
├── unit/                          # Unit tests
│   ├── services/                  # Service tests
│   ├── handlers/                  # Handler tests
│   ├── models/                    # Model tests
│   └── utils/                     # Utility tests
├── integration/                   # Integration tests
│   ├── api/                       # API endpoint tests
│   ├── database/                  # Database tests
│   └── external/                  # External service tests
├── e2e/                          # End-to-end tests
│   ├── auth/                      # Authentication flow
│   ├── applications/              # Application management
│   └── vps/                       # VPS management
├── mocks/                        # Mock implementations
└── fixtures/                     # Test data
```

**Example Test:**
```nim
# tests/unit/services/test_application_service.nim
import unittest, asyncdispatch, json
import ../../../src/services/application_service
import ../../mocks/mock_kv_client

suite "Application Service Tests":
  setup:
    let mockKVClient = newMockKVClient()
    let appService = newApplicationService(mockKVClient, mockHelmClient)
  
  test "should create application successfully":
    let createRequest = ApplicationCreateRequest(
      name: "test-app",
      appType: "code-server",
      subdomain: "test",
      domain: "example.com",
      vpsId: "vps-123"
    )
    
    let result = waitFor appService.createApplication("account-123", createRequest)
    
    check result.id != ""
    check result.name == "test-app"
    check result.status == "creating"
  
  test "should list applications for account":
    mockKVClient.setMockData("account-123", "app:", @[
      %* {"id": "app-1", "name": "App 1"},
      %* {"id": "app-2", "name": "App 2"}
    ])
    
    let results = waitFor appService.listApplications("account-123")
    
    check results.len == 2
    check results[0].name == "App 1"
```

### 4.2 Frontend Testing

**Test Structure:**
```
frontend/tests/
├── unit/                          # Component unit tests
│   ├── components/               # Component tests
│   ├── stores/                   # Store tests
│   └── utils/                    # Utility tests
├── integration/                   # Integration tests
│   ├── api/                      # API integration tests
│   └── auth/                     # Authentication tests
├── e2e/                          # End-to-end tests
│   ├── playwright/               # Playwright tests
│   └── cypress/                  # Cypress tests (alternative)
└── mocks/                        # Mock implementations
```

**Example Component Test:**
```javascript
// tests/unit/components/ApplicationCard.test.js
import { render, fireEvent, screen } from '@testing-library/svelte';
import ApplicationCard from '../../../src/lib/components/ApplicationCard.svelte';

describe('ApplicationCard', () => {
  const mockApplication = {
    id: 'app-1',
    name: 'Test App',
    appType: 'code-server',
    appVersion: '1.0.0',
    status: 'running',
    url: 'https://test.example.com'
  };

  test('renders application information correctly', () => {
    render(ApplicationCard, { application: mockApplication });
    
    expect(screen.getByText('Test App')).toBeInTheDocument();
    expect(screen.getByText('code-server')).toBeInTheDocument();
    expect(screen.getByText('1.0.0')).toBeInTheDocument();
  });

  test('handles delete confirmation', async () => {
    const { component } = render(ApplicationCard, { application: mockApplication });
    
    const deleteButton = screen.getByText('Delete');
    
    // Mock window.confirm
    window.confirm = jest.fn(() => true);
    
    await fireEvent.click(deleteButton);
    
    expect(window.confirm).toHaveBeenCalledWith('Are you sure you want to delete this application?');
  });
});
```

## Phase 5: Migration Execution Plan

### 5.1 Migration Timeline

**Week 1-2: Foundation Setup**
- Set up nim/ directory structure
- Configure Nim backend with Jester
- Set up SvelteKit frontend
- Implement basic JWT authentication
- Create development environment

**Week 3-4: Core Backend Migration**
- Migrate authentication handlers
- Implement JWT middleware
- Migrate application service layer
- Set up Cloudflare client integration
- Create basic API endpoints

**Week 5-6: Frontend Foundation**
- Set up SvelteKit routing
- Implement authentication UI
- Create basic component library
- Set up state management stores
- Implement API client

**Week 7-8: Feature Migration**
- Application management features
- VPS management features  
- DNS management features
- Terminal WebSocket integration
- Testing implementation

**Week 9-10: Testing & Refinement**
- Comprehensive testing suite
- Performance optimization
- Security audit
- Documentation
- Deployment preparation

### 5.2 Migration Strategy

**Parallel Development:**
- Backend and frontend can be developed simultaneously
- Use API contracts to define interfaces
- Mock external services during development
- Implement feature parity incrementally

**Data Migration:**
- Cloudflare KV storage remains unchanged
- API contracts maintain backward compatibility
- Gradual transition with feature flags
- Rollback capabilities

**Testing Strategy:**
- Test-driven development approach
- Mock external services for unit tests
- Integration tests with real APIs
- End-to-end testing for critical workflows

### 5.3 Risk Mitigation

**CRITICAL Technical Risks:**
- **Nim ecosystem maturity** → Use established libraries, fallback to Go for complex integrations
- **SvelteKit learning curve** → Incremental adoption, extensive documentation
- **JWT security** → Use established patterns, security audit
- **Performance concerns** → Benchmark against Go implementation
- **WebSocket limitations** → Hybrid approach, keep Go service as fallback
- **Complex auth flow** → Preserve all existing authentication steps
- **Static file serving** → Clear strategy for asset compilation and serving
- **Domain separation** → Maintain current handler organization patterns

**CRITICAL Operational Risks:**
- **Feature parity** → Comprehensive testing, gradual rollout
- **Data integrity** → Backup strategies, rollback procedures
- **Deployment complexity** → Containerization, automated deployment
- **User experience** → Maintain UI consistency, user feedback
- **Route responsibility confusion** → Clear API/web separation
- **VPS SSH access** → Maintain current SSH proxy functionality
- **Configuration system** → Preserve YAML-driven app definitions

**CRITICAL Architecture Risks:**
- **Two-server architecture** → Need reverse proxy/load balancer
- **Port management** → Backend:8080, Frontend:3000, production deployment
- **Session management** → WebSocket sessions across server boundaries
- **CORS configuration** → Proper cross-origin setup between services
- **SSL/TLS termination** → Clear SSL handling strategy

## Phase 6: Deployment & Operations

### 6.1 Containerization

**Backend Dockerfile:**
```dockerfile
FROM nimlang/nim:alpine
WORKDIR /app
COPY backend/xanthus.nimble .
RUN nimble install -y --depsOnly
COPY backend/src ./src
RUN nimble build -d:release
EXPOSE 8080
CMD ["./xanthus"]
```

**Frontend Dockerfile:**
```dockerfile
FROM node:18-alpine
WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build
FROM nginx:alpine
COPY --from=0 /app/build /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

### 6.2 Development Environment

**Docker Compose:**
```yaml
# nim/docker-compose.yml
version: '3.8'

services:
  backend:
    build: ./backend
    ports:
      - "8080:8080"
    environment:
      - JWT_SECRET=${JWT_SECRET}
      - CLOUDFLARE_API_TOKEN=${CLOUDFLARE_API_TOKEN}
    volumes:
      - ./backend:/app
    depends_on:
      - redis

  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    environment:
      - VITE_API_URL=http://localhost:8080
    volumes:
      - ./frontend:/app
    depends_on:
      - backend

  redis:
    image: redis:alpine
    ports:
      - "6379:6379"
```

### 6.3 CI/CD Pipeline

**GitHub Actions:**
```yaml
# .github/workflows/nim-ci.yml
name: Nim Backend CI/CD

on:
  push:
    branches: [ nim-backend ]
  pull_request:
    branches: [ nim-backend ]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Setup Nim
      uses: jiro4989/setup-nim-action@v1
      with:
        nim-version: 1.6.0
    
    - name: Install dependencies
      run: nimble install -y --depsOnly
      working-directory: ./nim/backend
    
    - name: Run tests
      run: nimble test
      working-directory: ./nim/backend
    
    - name: Build
      run: nimble build -d:release
      working-directory: ./nim/backend

  frontend-test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Setup Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '18'
    
    - name: Install dependencies
      run: npm ci
      working-directory: ./nim/frontend
    
    - name: Run tests
      run: npm test
      working-directory: ./nim/frontend
    
    - name: Build
      run: npm run build
      working-directory: ./nim/frontend
```

## 🚨 CRITICAL HOLES IDENTIFIED & FIXES APPLIED

### **Original Plan Holes:**
1. **❌ Route Responsibility Confusion** - Plan didn't separate API from web routes
2. **❌ Authentication Oversimplification** - Missed complex Cloudflare token flow
3. **❌ WebSocket Implementation Gaps** - Ignored Jester WebSocket limitations
4. **❌ Static File Serving Missing** - No strategy for asset compilation/serving
5. **❌ Domain Organization Loss** - Risk of losing current domain boundaries
6. **❌ Mixed Response Types** - Didn't address HTML/JSON dual responses
7. **❌ Deployment Architecture** - No reverse proxy/load balancer strategy
8. **❌ Session Management** - No cross-server session handling
9. **❌ CORS Configuration** - Missing cross-origin setup
10. **❌ Performance Benchmarking** - No comparison strategy

### **✅ Fixes Applied:**

**1. Route Responsibility Matrix** (Section 2.1)
- **Backend (Nim/Jester)**: API routes only (JSON responses)
- **Frontend (SvelteKit)**: Web routes + static files
- **Clear separation**: `/api/v1/*` vs `/app/*`

**2. Enhanced Authentication Flow** (Section 2.5)
- **Preserved complex flow**: Cloudflare token → KV namespace → CSR → JWT
- **10-minute caching**: Token validation caching
- **Complete auth info**: Account setup, namespace verification

**3. WebSocket Strategy** (Section 3.6)
- **Hybrid approach**: Option to keep Go WebSocket service
- **Enhanced Nim implementation**: Session management, heartbeat
- **Risk mitigation**: Fallback strategies, performance testing

**4. Deployment Architecture** (Section 6.1-6.2)
- **Two-server setup**: Backend:8080, Frontend:3000
- **Reverse proxy**: Nginx configuration for production
- **Docker compose**: Development environment setup

**5. Critical Risk Assessment** (Section 5.3)
- **Technical risks**: Nim ecosystem, WebSocket limitations
- **Operational risks**: Feature parity, deployment complexity
- **Architecture risks**: Session management, CORS, SSL termination

## **RECOMMENDATION: PROCEED WITH CAUTION**

### **HIGH-RISK ITEMS REQUIRING IMMEDIATE ATTENTION:**

1. **WebSocket Implementation** - Consider keeping Go service for terminals
2. **Complex Authentication** - Ensure all 5 steps are implemented
3. **Performance Benchmarking** - Must compare against Go implementation
4. **Session Management** - Cross-server session handling is critical
5. **Static File Strategy** - Asset compilation pipeline must be defined

### **BEFORE STARTING MIGRATION:**

1. **Prototype WebSocket** - Test Jester WebSocket with terminal sessions
2. **Benchmark Performance** - Compare Nim vs Go for critical operations
3. **Test Authentication** - Verify all 5 auth steps work in Nim
4. **Define Asset Pipeline** - CSS/JS compilation strategy
5. **Setup Reverse Proxy** - Production deployment architecture

### **SUCCESS FACTORS (UPDATED):**

- **✅ Maintain Handler-Service-Model architecture**
- **✅ Preserve ALL existing functionality** (especially complex auth)
- **✅ Implement comprehensive testing** (unit, integration, E2E)
- **✅ Use established libraries and patterns** (avoid experimental)
- **✅ Plan for rollback capabilities** (especially for WebSocket)
- **✅ Focus on performance and security** (benchmark everything)
- **✅ Clear route responsibility separation** (API vs web)
- **✅ Hybrid approach for high-risk components** (WebSocket fallback)

### **FINAL ASSESSMENT:**

This migration plan now addresses the critical architectural holes identified in the original plan. However, the **complexity and risk level is HIGH** due to:

- **WebSocket limitations** in Jester
- **Complex authentication flow** requiring precise implementation
- **Two-server architecture** requiring careful coordination
- **Performance requirements** for production workloads

**RECOMMENDATION**: Consider a **phased approach** starting with non-WebSocket functionality first, then gradually migrating terminal/WebSocket features with a proven fallback strategy.

## Conclusion

This **significantly enhanced** migration plan provides a comprehensive roadmap for transitioning Xanthus from Go + HTMX/Alpine.js to Nim (Jester) + SvelteJS with JWT authentication. The plan now addresses all critical architectural holes while maintaining the excellent patterns already established.

The **enhanced phased approach** allows for incremental migration while maintaining system stability and feature parity. The focus on **risk mitigation, performance benchmarking, and comprehensive testing** ensures a successful transition with minimal disruption to users and operations.

**The new Nim + SvelteJS stack will provide:**
- Better performance and memory efficiency (if benchmarks confirm)
- Type safety and compile-time guarantees
- Modern, reactive user interface
- Improved developer experience
- Better testing and maintenance capabilities
- **Clear separation of concerns** between API and web layers