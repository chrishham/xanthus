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

#### VPS Handler (`vps.go`) ✅ COMPLETED
- **Priority**: High (Complex business logic)
- **Implementation**: `/tests/unit/handlers/vps_test.go`
- **Test Cases**:
  - ✅ `TestHandleVPSCreate` - Server creation with validation:
    - ✅ Missing parameters (name, location, server_type)
    - ✅ Invalid token scenarios
    - ✅ SSH key creation flow
    - ✅ Server type pricing calculation
    - ✅ VPS configuration storage
  - ✅ `TestHandleVPSDelete` - Server deletion:
    - ✅ Valid deletion flow
    - ✅ Configuration cleanup
    - ✅ Error handling for non-existent servers
  - ✅ `TestHandleVPSList` - Server listing with cost information
  - ✅ `TestHandleVPSPowerActions` - Power management (on/off/reboot)
  - ✅ `TestHandleVPSServerOptions` - Filtering and sorting logic
  - ✅ `TestHandleVPSValidateName` - Name uniqueness validation
  - ✅ `TestHandleVPSManagePage` - VPS management page rendering
  - ✅ `TestHandleVPSCreatePage` - VPS creation page rendering
  - ✅ `TestHandleVPSConfigure` - SSL certificate configuration for domains
  - ✅ `TestHandleVPSDeploy` - Kubernetes manifest deployment
  - ✅ `TestHandleVPSLocations` - Hetzner location fetching
  - ✅ `TestHandleVPSServerTypes` - Server type filtering and availability
  - ✅ `TestHandleVPSCheckKey/ValidateKey` - Hetzner API key management
  - ✅ `TestHandleVPSSSHKey` - SSH private key retrieval
  - ✅ `TestHandleVPSStatus` - VPS health status via SSH
  - ✅ `TestHandleVPSLogs` - VPS log retrieval
  - ✅ `TestHandleVPSTerminal` - Web terminal session creation
  - ✅ `TestHandleSetupHetzner` - Hetzner API key setup flow
- **Advanced Testing**:
  - ✅ Mock servers for external API calls (Cloudflare, Hetzner)
  - ✅ Edge cases and error handling tests
  - ✅ Concurrent operations testing
  - ✅ Performance benchmarks for high-frequency requests
  - ✅ Integration-style test flows
  - ✅ Server ID parsing validation
  - ✅ Large manifest deployment testing

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

### 1.3 Utility Tests (`internal/utils/`) ✅ COMPLETED

#### Crypto Utils (`crypto.go`) ✅ COMPLETED
- **Priority**: High (Security critical)
- **Implementation**: `/tests/unit/utils/crypto_test.go`
- **Test Cases**:
  - ✅ `TestEncryptData` - AES-256-GCM encryption validation
  - ✅ `TestDecryptData` - Decryption with various tokens
  - ✅ `TestEncryptDecryptRoundTrip` - Data integrity testing with 6 test cases
  - ✅ `TestDecryptDataWithWrongToken` - Security validation
  - ✅ `TestDecryptDataWithInvalidBase64` - Error handling
  - ✅ `TestDecryptDataWithTooShortCiphertext` - Edge case handling
  - ✅ `TestEncryptionConsistency` - Multiple encryption verification
  - ✅ `TestTokenSensitivity` - Cross-token decryption prevention
  - **Mock Strategy**: Direct function testing with various input scenarios
- **Additional**: Benchmark tests for encryption/decryption performance

#### Response Utils (`responses.go`) ✅ COMPLETED
- **Priority**: Medium
- **Implementation**: `/tests/unit/utils/responses_test.go`
- **Test Cases**:
  - ✅ `TestJSONSuccess/JSONSuccessSimple` - Success response formatting
  - ✅ `TestJSONError/JSONBadRequest/JSONUnauthorized/JSONForbidden/JSONNotFound/JSONInternalServerError/JSONServiceUnavailable` - Error responses with status codes
  - ✅ `TestJSONResponse` - Custom response handling
  - ✅ `TestHTMLError/HTMLSuccess` - HTMX HTML responses
  - ✅ `TestHTMXRedirect/HTMXRefresh` - HTMX header management
  - ✅ `TestJSONValidationError` - Field validation error structure
  - ✅ `TestVPSCreationSuccess/VPSDeletionSuccess/VPSConfigurationSuccess` - Domain-specific responses
  - ✅ `TestApplicationSuccess/DNSConfigurationSuccess/SetupSuccess` - Application lifecycle responses
  - **Mock Strategy**: HTTP test server with Gin context mocking
- **Additional**: Benchmark tests for JSON response generation

#### Cloudflare Utils (`cloudflare.go`) ✅ COMPLETED
- **Priority**: High
- **Implementation**: `/tests/unit/utils/cloudflare_test.go`
- **Test Cases**:
  - ✅ `TestVerifyCloudflareToken` - Token verification with real API calls
  - ✅ `TestCheckKVNamespaceExists` - KV namespace discovery and validation
  - ✅ `TestCreateKVNamespace` - Namespace creation logic
  - ✅ `TestGetXanthusNamespaceID` - Namespace ID retrieval
  - ✅ `TestPutKVValue/GetKVValue` - Key-value operations
  - ✅ `TestFetchCloudflareDomains` - Domain zone fetching
  - ✅ `TestCloudflareUtilsIntegration` - Full workflow validation
  - **Mock Strategy**: HTTP test servers for API response mocking (where possible)
- **Additional**: Benchmark tests for token verification and namespace operations

#### Hetzner Utils (`hetzner.go`) ✅ COMPLETED
- **Priority**: Medium
- **Implementation**: `/tests/unit/utils/hetzner_test.go`
- **Test Cases**:
  - ✅ `TestValidateHetznerAPIKey` - API key validation with real API calls
  - ✅ `TestGetHetznerAPIKey` - Encrypted API key retrieval
  - ✅ `TestFetchHetznerLocations/ServerTypes` - Data fetching from API
  - ✅ `TestFetchServerAvailability` - Real-time availability checking
  - ✅ `TestFilterSharedVCPUServers` - Server type filtering logic
  - ✅ `TestGetServerTypeMonthlyPrice` - Price parsing with edge cases (empty, invalid, currency)
  - ✅ `TestSortServerTypesByPrice/CPU/Memory` - Sorting algorithms (ascending/descending)
  - ✅ `TestSortingEdgeCases` - Empty slices, single elements, identical values
  - ✅ `TestHetznerUtilsIntegration` - Full workflow with invalid credentials
  - **Mock Strategy**: HTTP test servers for API mocking
- **Additional**: Benchmark tests for sorting algorithms and price parsing

#### Server Utils (`server.go`) ✅ COMPLETED
- **Priority**: Low
- **Implementation**: `/tests/unit/utils/server_test.go`
- **Test Cases**:
  - ✅ `TestFindAvailablePort` - Port discovery in range 8080-8090
  - ✅ `TestFindAvailablePortEdgeCases` - Boundary testing and format validation
  - ✅ `TestFindAvailablePortPerformance` - Performance testing with 100 iterations
  - ✅ Port occupation scenarios with multiple listeners
  - ✅ Concurrent access validation
  - ✅ Port availability verification
  - **Mock Strategy**: Real port testing with net.Listen()
- **Additional**: Benchmark tests for port scanning performance

### 1.4 Middleware Tests (`internal/middleware/`) ✅ COMPLETED

#### Auth Middleware (`auth.go`) ✅ COMPLETED
- **Priority**: High
- **Implementation**: `/tests/unit/middleware/auth_test.go`
- **Test Cases**:
  - ✅ `TestAuthMiddleware_NoCookie` - Missing cookie redirects to login
  - ✅ `TestAuthMiddleware_EmptyCookie` - Empty cookie redirects to login
  - ✅ `TestAuthMiddleware_InvalidToken` - Invalid token redirects to login
  - ⏸️ `TestAuthMiddleware_ValidToken` - Valid token allows access (requires token mocking)
  - ⏸️ `TestAuthMiddleware_TokenStoredInContext` - Token storage in context (requires token mocking)
  - ✅ `TestAPIAuthMiddleware_NoCookie` - Missing cookie returns 401 JSON
  - ✅ `TestAPIAuthMiddleware_EmptyCookie` - Empty cookie returns 401 JSON
  - ✅ `TestAPIAuthMiddleware_InvalidToken` - Invalid token returns 401 JSON
  - ⏸️ `TestAPIAuthMiddleware_ValidToken` - Valid token allows API access (requires token mocking)
  - ⏸️ `TestAPIAuthMiddleware_TokenStoredInContext` - Token storage in API context (requires token mocking)
- **Mock Strategy**: Direct middleware testing with Gin test mode, HTTP test servers
- **Additional**: Benchmark tests for performance measurement of middleware operations

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

### Test Commands ✅ IMPLEMENTED

The following test commands are now available in the Makefile:

```makefile
# Run all structured tests
test:
	go test -v ./tests/...

# Run unit tests only
test-unit:
	go test -v ./tests/unit/...

# Run integration tests (when they exist)
test-integration:
	go test -v ./tests/integration/...

# Run tests with coverage report
test-coverage:
	go test -v ./tests/... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run all tests including any legacy tests
test-all:
	go test -v ./...

# Clean build artifacts and coverage files
clean:
	rm -rf bin/
	rm -f web/static/css/output.css
	rm -rf web/static/js/vendor/
	rm -f coverage.out coverage.html
```

**Usage Examples:**
- `make test` - Run structured tests (recommended for development)
- `make test-unit` - Run only unit tests for quick feedback
- `make test-coverage` - Generate coverage report in `coverage.html`
- `make test-all` - Run everything including legacy tests (CI/CD)

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

### Phase 1 (Critical) ✅ COMPLETED
1. ✅ Crypto utils tests - **COMPLETED** (`/tests/unit/utils/crypto_test.go`)
2. ✅ Core service tests (Hetzner, Cloudflare, SSH, Helm, KV) - **COMPLETED** (`/tests/unit/services/`)
3. ✅ Utility layer tests (responses, server, cloudflare, hetzner utils) - **COMPLETED** (`/tests/unit/utils/`)
4. ✅ Authentication handler tests - **COMPLETED** (`/tests/unit/handlers/auth_test.go`)
5. ✅ Improved Makefile with structured test commands - **COMPLETED**

### Phase 2 (Important) 🔄 IN PROGRESS
1. ✅ Authentication middleware tests - **COMPLETED** (`/tests/unit/middleware/auth_test.go`)
2. ✅ VPS handler tests - **COMPLETED** (`/tests/unit/handlers/vps_test.go`)
3. 🔄 Remaining handler tests (applications, dns, pages)
4. 🔄 Integration tests
5. 🔄 End-to-end workflows

### Phase 3 (Enhancement) ⏳ PENDING
1. ⏳ Performance tests
2. ⏳ Security tests
3. ⏳ Load testing
4. ⏳ Documentation tests

### Current Status Summary
- **Unit Tests Completed**: 13 test files covering 4 major layers
  - **Handlers**: 2/5 files (auth_test.go, vps_test.go)
  - **Services**: 5/5 files (cloudflare, helm, hetzner, kv, ssh)
  - **Utils**: 5/5 files (crypto, responses, cloudflare, hetzner, server)
  - **Middleware**: 1/1 files (auth_test.go)
- **Test Structure**: Fully organized under `/tests/unit/`
- **Makefile**: Enhanced with 5 new test commands
- **Coverage**: Ready for coverage reporting via `make test-coverage`

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