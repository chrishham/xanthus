# Test Suggestions for Xanthus

This document provides comprehensive testing recommendations for the Xanthus K3s deployment tool. The codebase follows a clean architecture with handlers, services, and utilities that can be thoroughly tested.

## Project Overview

Xanthus is a Go-based K3s deployment tool with:
- **Framework**: Gin web framework
- **Architecture**: Handler-based with domain separation
- **External APIs**: Hetzner Cloud, Cloudflare
- **Infrastructure**: K3s, Helm, SSH connections

## Test Structure Recommendation

```
tests/
├── unit/
│   ├── handlers/
│   ├── services/
│   ├── utils/
│   └── middleware/
├── integration/
│   ├── api/
│   ├── external/
│   └── end_to_end/
├── fixtures/
│   ├── responses/
│   └── configs/
└── mocks/
    ├── services/
    └── external/
```

## 1. Unit Tests

### 1.1 Handler Tests (`internal/handlers/`)

#### Authentication Handler (`auth.go`) ✅ COMPLETED
- **Priority**: High
- **Implementation**: `/tests/unit/handlers/auth_test.go`
- **Test Cases**:
  - ✅ `TestHandleRoot` - Should redirect to `/login`
  - ✅ `TestHandleLoginPage` - Should render login template
  - ✅ `TestHandleLogin` - Multiple scenarios:
    - ✅ Empty token should return 400
    - ✅ Invalid token should return error message
    - ⏸️ Valid token should set cookie and redirect (requires external API mocking)
    - ⏸️ KV namespace creation logic (requires external API mocking)
    - ⏸️ CSR generation and storage (requires external API mocking)
  - ✅ `TestHandleLogout` - Should clear cookie and redirect
  - ✅ `TestHandleHealth` - Should return 200 with status
- **Additional**: Benchmark tests included for performance measurement

#### VPS Handler (`vps.go`)
- **Priority**: High (Complex business logic)
- **Test Cases**:
  - `TestHandleVPSCreate` - Server creation with validation:
    - Missing parameters (name, location, server_type)
    - Invalid token scenarios
    - SSH key creation flow
    - Server type pricing calculation
    - VPS configuration storage
  - `TestHandleVPSDelete` - Server deletion:
    - Valid deletion flow
    - Configuration cleanup
    - Error handling for non-existent servers
  - `TestHandleVPSList` - Server listing with cost information
  - `TestPerformVPSAction` - Power management (on/off/reboot)
  - `TestHandleVPSServerOptions` - Filtering and sorting logic
  - `TestHandleVPSValidateName` - Name uniqueness validation

#### Applications Handler (`applications.go`)
- **Priority**: Medium
- **Test Cases**:
  - Helm chart installation validation
  - Application lifecycle management
  - Configuration validation

#### DNS Handler (`dns.go`)
- **Priority**: Medium
- **Test Cases**:
  - SSL certificate configuration
  - Domain validation
  - Cloudflare integration

### 1.2 Service Tests (`internal/services/`) ✅ COMPLETED

#### Hetzner Service (`hetzner.go`) ✅ COMPLETED
- **Priority**: High (External API integration)
- **Implementation**: `/tests/unit/services/hetzner_test.go`
- **Test Cases**:
  - ✅ `TestHetznerService_MakeRequest` - HTTP request building and error handling
  - ✅ `TestHetznerService_ListServers` - Response parsing and filtering
  - ✅ `TestHetznerService_CreateServer` - Server creation with cloud-init
  - ✅ `TestHetznerService_SSHKeyOperations` - SSH key management logic (create, find, list)
  - ✅ `TestHetznerService_DeleteServer` - Server cleanup
  - ✅ `TestHetznerService_PowerOperations` - Power management (on/off/reboot)
  - ✅ `TestHetznerService_ErrorHandling` - API errors and network failures
  - **Mock Strategy**: Mock HTTP client with httptest servers, test against fixtures
- **Additional**: Benchmark tests and helper functions for performance measurement

#### Cloudflare Service (`cloudflare.go`) ✅ COMPLETED
- **Priority**: High (Complex SSL operations)
- **Implementation**: `/tests/unit/services/cloudflare_test.go`
- **Test Cases**:
  - ✅ `TestCloudflareService_GenerateCSR` - CSR and private key generation with validation
  - ✅ `TestCloudflareService_MakeRequest` - HTTP request handling and error responses
  - ✅ `TestCloudflareService_GetZoneID` - Zone retrieval and domain validation
  - ✅ `TestCloudflareService_SSLModeOperations` - SSL mode configuration (strict/flexible)
  - ✅ `TestCloudflareService_AlwaysHTTPSOperations` - HTTPS enforcement settings
  - ✅ `TestCloudflareService_CreateOriginCertificate` - Certificate creation with CSR
  - ✅ `TestCloudflareService_AppendRootCertificate` - Root certificate chain building
  - ✅ `TestCloudflareService_PageRuleOperations` - Page rule creation and management
  - ✅ `TestCloudflareService_ConvertPrivateKeyToSSH` - Key format conversion
  - ✅ `TestCloudflareService_ConfigureDomainSSL` - Complete SSL setup flow
  - ✅ `TestCloudflareService_RemoveDomainFromXanthus` - SSL cleanup and rollback
  - **Mock Strategy**: Mock HTTP responses for API calls with httptest servers
- **Additional**: Benchmark tests for CSR generation and key conversion performance

#### Helm Service (`helm.go`) ✅ COMPLETED
- **Priority**: Medium
- **Implementation**: `/tests/unit/services/helm_test.go`
- **Test Cases**:
  - ✅ `TestHelmService_InstallChart` - Chart installation with custom values and validation
  - ✅ `TestHelmService_UpgradeChart` - Release upgrade logic and version management
  - ✅ `TestHelmService_UninstallChart` - Chart removal and cleanup
  - ✅ `TestHelmService_GetReleaseStatus` - Status parsing (deployed/failed/pending/unknown)
  - ✅ `TestHelmService_CommandConstruction` - Helm command building with parameters
  - ✅ `TestHelmService_ParameterValidation` - Input validation and error handling
  - ✅ `TestHelmService_ErrorScenarios` - Network, auth, and cluster access failures
  - **Mock Strategy**: Mock SSH service for command execution testing
- **Additional**: Benchmark tests for command construction and chart operations

#### SSH Service (`ssh.go`) ✅ COMPLETED
- **Priority**: High (Security critical)
- **Implementation**: `/tests/unit/services/ssh_test.go`
- **Test Cases**:
  - ✅ `TestSSHService_ConnectionCaching` - Connection establishment and reuse
  - ✅ `TestSSHService_ExecuteCommand` - Command execution and result handling
  - ✅ `TestSSHService_PrivateKeyParsing` - PEM key validation and error handling
  - ✅ `TestSSHService_CheckVPSHealth` - Comprehensive health checks with status parsing
  - ✅ `TestSSHService_ConfigureK3s` - SSL certificate configuration and K3s management
  - ✅ `TestSSHService_DeployManifest` - Kubernetes manifest deployment
  - ✅ `TestSSHService_GetK3sLogs` - Log retrieval and parsing
  - ✅ `TestSSHService_HelmOperations` - Helm repository and chart management
  - ✅ `TestSSHService_ConnectionLifecycle` - Connection cleanup and management
  - ✅ `TestSSHService_ErrorHandling` - Timeout, authentication, and network failures
  - **Mock Strategy**: Mock SSH client and connection for testing
- **Additional**: Benchmark tests for command execution and connection caching

#### KV Service (`kv.go`) ✅ COMPLETED
- **Priority**: Medium (Data persistence)
- **Implementation**: `/tests/unit/services/kv_test.go`
- **Test Cases**:
  - ✅ `TestKVService_GetXanthusNamespaceID` - Namespace discovery and validation
  - ✅ `TestKVService_PutValue` - Key-value storage with JSON marshaling
  - ✅ `TestKVService_GetValue` - Data retrieval and unmarshaling
  - ✅ `TestKVService_DeleteValue` - Key deletion and cleanup
  - ✅ `TestKVService_DomainSSLOperations` - SSL configuration storage and management
  - ✅ `TestKVService_VPSConfigOperations` - VPS configuration CRUD operations
  - ✅ `TestKVService_CalculateVPSCosts` - Cost calculation with time-based billing
  - ✅ `TestKVService_KeyParsing` - Key format validation and domain extraction
  - ✅ `TestKVService_ErrorHandling` - Network, auth, and data format errors
  - **Mock Strategy**: Mock HTTP responses for Cloudflare KV API calls
- **Additional**: Benchmark tests for cost calculations and key operations

### 1.3 Utility Tests (`internal/utils/`)

#### Crypto Utils (`crypto.go`)
- **Priority**: High (Security critical)
- **Test Cases**:
  - `TestEncryptData` - AES-256-GCM encryption
  - `TestDecryptData` - Decryption with various tokens
  - `TestEncryptDecryptRoundTrip` - Data integrity
  - Edge cases: empty data, invalid tokens

#### Response Utils (`responses.go`)
- **Priority**: Medium
- **Test Cases**:
  - `TestJSONSuccess` - Success response formatting
  - `TestJSONError` - Error response formatting
  - `TestValidationError` - Validation error structure
  - Status code correctness

#### Cloudflare Utils (`cloudflare.go`)
- **Priority**: High
- **Test Cases**:
  - Token verification logic
  - KV operations (get, put, delete)
  - Zone and namespace management

#### Hetzner Utils (`hetzner.go`)
- **Priority**: Medium
- **Test Cases**:
  - API key validation
  - Server type filtering and sorting
  - Pricing calculations

### 1.4 Middleware Tests (`internal/middleware/`)

#### Auth Middleware (`auth.go`)
- **Priority**: High
- **Test Cases**:
  - `TestAuthMiddleware` - Token validation flow
  - `TestAPIAuthMiddleware` - API endpoint protection
  - Cookie handling and context setting

## 2. Integration Tests

### 2.1 API Integration Tests
- **Full HTTP request/response cycles**
- **Database/KV storage integration**
- **Authentication flows**

### 2.2 External Service Integration
- **Hetzner Cloud API** (with real test account)
- **Cloudflare API** (with test zone)
- **SSH connections** (with test VPS)

### 2.3 End-to-End Tests
- **Complete VPS lifecycle**: Create → Configure → Deploy → Delete
- **SSL certificate management**: Configure → Validate → Remove
- **Application deployment**: Install → Upgrade → Uninstall

## 3. Test Implementation Strategy

### 3.1 Mocking Strategy

#### HTTP Client Mocking
```go
// Use httptest for mocking external API calls
func TestHetznerService_ListServers(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Mock Hetzner API response
    }))
    defer server.Close()
    
    // Test with mocked server
}
```

#### SSH Client Mocking
```go
// Mock SSH service for Helm operations
type MockSSHService struct{}
func (m *MockSSHService) ConnectToVPS(ip, user, key string) (*ssh.Connection, error) {
    // Return mock connection
}
```

### 3.2 Test Data Management

#### Fixtures (`tests/fixtures/`)
- **API responses**: JSON files for Hetzner/Cloudflare responses
- **Configurations**: Sample VPS and SSL configurations
- **SSH keys**: Test key pairs for validation

#### Test Database
- **In-memory KV storage** for unit tests
- **Test Cloudflare account** for integration tests

### 3.3 Test Coverage Goals

| Component | Target Coverage |
|-----------|----------------|
| Handlers | 85%+ |
| Services | 90%+ |
| Utils | 95%+ |
| Middleware | 90%+ |

## 4. Testing Tools & Libraries

### Recommended Testing Stack
- **Framework**: Go standard `testing` package
- **Assertions**: `testify/assert` and `testify/require`
- **Mocking**: `testify/mock` for interfaces
- **HTTP Testing**: `httptest` for API endpoints
- **Database**: In-memory implementations

### Test Commands
```makefile
# Add to Makefile
test-unit:
	go test -v ./internal/... -short

test-integration:
	go test -v ./tests/integration/...

test-coverage:
	go test -v ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

test-all:
	go test -v ./...
```

## 5. Security Testing Considerations

### Crypto Operations
- **Key generation randomness**
- **Encryption/decryption integrity**
- **Token validation robustness**

### API Security
- **Authentication bypass attempts**
- **Input validation and sanitization**
- **Rate limiting (if implemented)**

### SSH Security
- **Key handling and storage**
- **Connection security**
- **Command injection prevention**

## 6. Performance Testing

### Load Testing
- **Concurrent VPS operations**
- **API endpoint performance**
- **Memory usage under load**

### Stress Testing
- **Multiple Hetzner API calls**
- **Cloudflare rate limits**
- **SSH connection pooling**

## 7. Error Handling Tests

### Network Failures
- **API timeouts**
- **Connection drops**
- **Partial failures**

### Invalid Data
- **Malformed responses**
- **Missing required fields**
- **Type conversion errors**

## 8. Implementation Priority

### Phase 1 (Critical)
1. Crypto utils tests
2. Authentication middleware tests
3. Core service tests (Hetzner, Cloudflare)
4. VPS handler tests

### Phase 2 (Important)
1. Remaining handler tests
2. Integration tests
3. End-to-end workflows
4. Error handling scenarios

### Phase 3 (Enhancement)
1. Performance tests
2. Security tests
3. Load testing
4. Documentation tests

## 9. Continuous Integration

### GitHub Actions Workflow
```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
      - run: make test-all
      - run: make test-coverage
```

## 10. Test Maintenance

### Regular Tasks
- **Update fixtures** when APIs change
- **Refresh test data** periodically
- **Review test coverage** with each release
- **Update mocks** for new features

This comprehensive testing strategy will ensure Xanthus is robust, secure, and maintainable as it grows.