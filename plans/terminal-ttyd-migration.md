# Terminal Production-Ready Migration Plan

## Overview
Replace GoTTY with integrated WebSocket terminal solution for production deployments.

## Current Issues
1. **GoTTY dependency**: Remote server deployment fails with "gotty: executable file not found in $PATH"
2. **Port exposure problem**: Each terminal uses different ports (9000-9100) that aren't exposed in production
3. **Multiple terminal limitation**: 4 terminals = 4 different unexposed ports
4. **Hardcoded localhost**: Terminal opens at `localhost:9000` instead of remote server hostname

## Plan: WebSocket-Based Terminal Integration

### Phase 1: Research WebSocket Terminal Solutions
1. Evaluate Go WebSocket terminal libraries (gorilla/websocket, etc.)
2. Research existing implementations (xterm.js + WebSocket backend)
3. Design session multiplexing over single WebSocket endpoint
4. Plan authentication and session management

### Phase 2: Implement WebSocket Terminal Service
1. **Replace external dependency**: Remove GoTTY/ttyd dependency entirely
2. **Create WebSocket handler**: Implement `/ws/terminal/{session_id}` endpoint
3. **SSH bridge**: Create Go-based SSH client that bridges to WebSocket
4. **Session management**: Manage multiple SSH sessions over single WebSocket connection
5. **Terminal multiplexing**: Support multiple terminals through single exposed port

### Phase 2.5: Security Implementation
1. **WebSocket Authentication**: 
   - Validate JWT tokens during WebSocket handshake (header/query param)
   - Reject unauthenticated WebSocket upgrade attempts
   - Use existing AuthMiddleware patterns for WebSocket endpoints
2. **Session Authorization**:
   - Validate session ownership before allowing WebSocket connection
   - Ensure users can only access their own terminal sessions
   - Cross-reference session ID with user account
3. **Session Security**:
   - Implement session expiry (auto-cleanup after inactivity)
   - Generate cryptographically secure session IDs
   - Session cleanup on user logout/token expiry
4. **Rate Limiting & Protection**:
   - Limit terminal creation per user/IP address
   - Prevent session flooding attacks
   - Monitor and log suspicious terminal access attempts

### Phase 3: Frontend Integration
1. **Replace GoTTY iframe**: Implement xterm.js with WebSocket connection
2. **Dynamic connection**: Connect to `wss://api.myclasses.gr/ws/terminal/{session_id}`
3. **Session handling**: Manage terminal sessions through main application
4. **UI updates**: Update both modal and new-tab terminals

### Phase 4: Production Architecture
1. **Single port deployment**: All terminals work through port 443/80 (HTTPS/HTTP)
2. **Session isolation**: Each terminal gets unique session ID, same WebSocket endpoint
3. **Scalability**: Support unlimited terminals without port conflicts
4. **Security**: Proper authentication and session validation

### Phase 5: Testing and Validation
1. Test multiple concurrent terminal sessions
2. Verify production deployment compatibility
3. Test session cleanup and resource management
4. Validate WebSocket security and authentication

## Expected Architecture

```
Browser (xterm.js) <-- WebSocket --> Xanthus Server <-- SSH --> VPS
    Multiple terminals              Single endpoint         Multiple SSH
    wss://api.myclasses.gr/ws/terminal/session-1
    wss://api.myclasses.gr/ws/terminal/session-2
    wss://api.myclasses.gr/ws/terminal/session-3
    wss://api.myclasses.gr/ws/terminal/session-4
```

## Files to Create/Modify
- `internal/handlers/websocket_terminal.go` - New WebSocket terminal handler with authentication
- `internal/services/websocket_terminal_service.go` - WebSocket terminal service with session management
- `internal/middleware/websocket_auth.go` - WebSocket authentication middleware
- `internal/router/routes.go` - Add WebSocket terminal routes with security
- `web/static/js/modules/terminal.js` - New xterm.js terminal implementation
- `web/static/js/modules/vps-management.js` - Update to use WebSocket terminals with auth tokens
- `web/templates/partials/terminal/` - New terminal UI components

## Security Architecture

```
Client Request → JWT Validation → Session Authorization → WebSocket Upgrade → SSH Bridge
     ↓               ↓                    ↓                    ↓             ↓
- Auth token    - Valid user      - User owns session  - Secure WebSocket - VPS SSH
- Rate limits   - Active session  - Session not expired - Auth headers    - Isolated shell
```

## Attack Prevention
- **Unauthenticated access**: JWT validation before WebSocket upgrade
- **Session hijacking**: Cryptographically secure session IDs + ownership validation  
- **Resource exhaustion**: Rate limiting + session cleanup
- **Direct WebSocket access**: Auth token required in WebSocket headers
- **Cross-user access**: Session ownership validation per user account

## Expected Benefits
- **Production ready**: Works with any number of terminals through single port
- **No external dependencies**: Pure Go implementation
- **Better security**: WebSocket authentication through existing auth system
- **Scalable**: No port limitations or external process management
- **Real-time**: Direct WebSocket connection for better performance

## Risk Assessment
- **Medium risk**: Larger architectural change requiring WebSocket implementation
- **Implementation complexity**: Requires SSH client integration with WebSocket
- **Testing requirements**: Need thorough testing of WebSocket connection handling
- **Rollback plan**: Keep existing terminal service as fallback during transition