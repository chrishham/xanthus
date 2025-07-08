# CLAUDE.md - Nim Migration Project

This file provides guidance to Claude Code when working with the Nim + SvelteJS migration of Xanthus.

## 📁 Project Structure

This is the **Nim migration branch** of Xanthus, transitioning from Go + HTMX/Alpine.js to Nim (Jester) + SvelteJS.

```
nim/
├── backend/                    # Nim backend with Jester framework
│   ├── src/
│   │   ├── handlers/          # HTTP request handlers (API only)
│   │   ├── services/          # Business logic services
│   │   ├── models/            # Data models and types
│   │   ├── middleware/        # JWT auth, CORS, WebSocket
│   │   ├── utils/             # Utility functions
│   │   └── main.nim           # Application entry point
│   ├── tests/                 # Backend tests
│   ├── config/                # Configuration files
│   └── xanthus.nimble         # Nim package configuration
├── frontend/                   # SvelteJS frontend
│   ├── src/
│   │   ├── lib/
│   │   │   ├── components/    # Reusable Svelte components
│   │   │   ├── stores/        # Svelte stores for state management
│   │   │   └── utils/         # Frontend utility functions
│   │   ├── routes/            # SvelteKit routes (web pages)
│   │   └── app.html           # Main HTML template
│   ├── static/                # Static assets
│   ├── tests/                 # Frontend tests
│   └── package.json           # Node.js dependencies
├── shared/                     # Shared types and contracts
│   ├── types/                 # TypeScript type definitions
│   └── api/                   # API schemas and contracts
└── plans/
    └── migration-plan.md       # Comprehensive migration documentation
```

## 🚀 Development Commands

### Backend (Nim + Jester)

**Prerequisites:**
- Install Nim: https://nim-lang.org/install.html
- Nim version >= 1.6.0

**Development:**
```bash
cd nim/backend

# Install dependencies
nimble install -y --depsOnly

# Run development server (port 8080)
nimble run

# Build for production
nimble build -d:release

# Run tests
nimble test
```

### Frontend (SvelteJS)

**Prerequisites:**
- Node.js >= 18
- npm or yarn

**Development:**
```bash
cd nim/frontend

# Install dependencies
npm install

# Run development server (port 3000)
npm run dev

# Build for production
npm run build

# Run tests
npm test

# Type checking
npm run check

# Linting and formatting
npm run lint
npm run format
```

### Full Stack Development

**Start both servers:**
```bash
# Terminal 1: Backend
cd nim/backend && nimble run

# Terminal 2: Frontend  
cd nim/frontend && npm run dev
```

**Access:**
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- Health Check: http://localhost:8080/health

## 🏗️ Architecture Overview

### Two-Server Architecture

**Backend (Nim/Jester) - Port 8080:**
- **API routes only** - Returns JSON responses
- **Route pattern**: `/api/v1/*`, `/health`, `/ws/*`
- **Responsibilities**: Business logic, external integrations, WebSocket connections

**Frontend (SvelteKit) - Port 3000:**
- **Web routes + static files** - Serves HTML pages and assets
- **Route pattern**: `/`, `/auth/*`, `/applications/*`, `/vps/*`, `/dns/*`
- **Responsibilities**: User interface, client-side logic, API consumption

**Development Proxy:**
- Vite proxy forwards `/api/*` and `/ws/*` to backend
- Seamless development experience on single port (3000)

### Authentication Flow

**COMPLEX LOGIN (One Time):**
1. User provides Cloudflare API token
2. Backend validates with Cloudflare API
3. Backend creates/verifies KV namespace 
4. Backend generates/verifies CSR
5. Backend creates JWT with user info
6. Frontend stores JWT in localStorage

**SIMPLE USAGE (All Subsequent Requests):**
1. Frontend sends `Authorization: Bearer <jwt_token>` header
2. Backend validates JWT and extracts user info
3. No more Cloudflare API calls needed

## 🛠️ Development Guidelines

### Backend Development (Nim)

**Handler-Service-Model Pattern:**
```nim
# handlers/applications.nim - HTTP request handling (JSON only)
# services/application_service.nim - Business logic
# models/application_model.nim - Data structures
```

**API Response Format:**
```nim
# Success response
%* {
  "status": "success",
  "data": actualData
}

# Error response  
%* {
  "status": "error",
  "message": "Error description"
}
```

**Service Integration:**
- Use dependency injection pattern
- Mock external services for testing
- Implement async/await for I/O operations

### Frontend Development (SvelteJS)

**Component Structure:**
```javascript
// lib/components/ - Reusable UI components
// lib/stores/ - Svelte stores for state management
// lib/utils/ - Utility functions and API client
// routes/ - SvelteKit pages and layouts
```

**State Management:**
- Use Svelte stores for global state
- Auth store manages JWT and user info
- Individual stores for applications, VPS, DNS

**API Integration:**
```javascript
// All API calls use JWT authentication
import { api } from '$lib/utils/api.js';

const result = await api.get('/applications');
const created = await api.post('/applications', data);
```

## 🧪 Testing Strategy

### Backend Testing
```bash
# Unit tests
nimble test

# Integration tests with mocks
# (Testing framework to be implemented)
```

### Frontend Testing
```bash
# Unit tests (Vitest)
npm run test:unit

# Integration tests (Playwright)
npm run test:integration

# Type checking
npm run check
```

## 🔄 Migration Status

**✅ Phase 1 Complete - Infrastructure Setup:**
- Directory structure created
- Basic Nim backend with Jester
- SvelteKit frontend with authentication
- Development environment configured

**🚧 Phase 2 In Progress - Backend Migration:**
- JWT authentication implementation
- Service layer migration
- API endpoint development
- External integrations

**📋 Upcoming Phases:**
- Phase 3: Frontend feature migration
- Phase 4: WebSocket implementation
- Phase 5: Testing and deployment

## 🚨 Critical Migration Notes

### Route Responsibility Matrix

| Feature | Backend (Nim/Jester) | Frontend (SvelteKit) |
|---------|---------------------|---------------------|
| **Authentication** | JWT validation, Cloudflare verification | Login UI, token storage |
| **Applications** | Business logic, Helm deployments | User interface, forms |
| **VPS** | Provider integrations | UI controls, status display |
| **DNS** | Cloudflare API operations | Configuration UI |
| **Terminal** | WebSocket proxy, SSH connections | Terminal UI, session management |
| **Static Files** | None (API only) | All assets, CSS, JavaScript |

### Key Differences from Go Version

1. **Two-server architecture** instead of single Go binary
2. **JWT-based auth** instead of session-based
3. **JSON APIs** instead of HTML templates
4. **WebSocket handling** may need hybrid approach
5. **Asset compilation** handled by Vite instead of Go

### Development Best Practices

- **Backend**: Focus on JSON APIs, avoid HTML generation
- **Frontend**: Use SvelteKit patterns, reactive stores
- **Testing**: Mock external services, test API contracts
- **Security**: Validate JWT tokens, sanitize inputs
- **Performance**: Benchmark against Go implementation

## 🔗 Related Documentation

- **[Migration Plan](plans/migration-plan.md)** - Comprehensive migration strategy
- **[Original CLAUDE.md](../CLAUDE.md)** - Go version development guide
- **[Nim Documentation](https://nim-lang.org/docs/)** - Nim language reference
- **[Jester Documentation](https://github.com/dom96/jester)** - Web framework docs
- **[SvelteKit Documentation](https://kit.svelte.dev/)** - Frontend framework docs

---

**⚠️ Important**: This is a migration branch. Maintain feature parity with the Go version while modernizing the architecture. Test thoroughly and benchmark performance against the original implementation.