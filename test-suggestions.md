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

#### Overview
End-to-end tests validate complete user workflows by testing the entire system from frontend to backend, including external service integrations. These tests ensure that all components work together correctly in real-world scenarios.

#### Test Environment Requirements
- **Test Hetzner Account**: Dedicated testing account with limited resources
- **Test Cloudflare Zone**: Sandbox domain for DNS/SSL testing (e.g., `test.example.com`)
- **Test Infrastructure**: Isolated K3s cluster for safe deployment testing
- **Mock External Services**: Fallback mocks for rate-limited APIs

#### 2.3.1 Complete VPS Lifecycle Tests
**Priority**: High  
**Duration**: ~15 minutes per test  
**Resource Requirements**: Test Hetzner account, test domain

##### Test Case: E2E_VPS_001 - Full VPS Deployment Flow
```go
func TestE2E_CompleteVPSLifecycle(t *testing.T) {
    // Test Steps:
    // 1. Login with valid Cloudflare token
    // 2. Configure Hetzner API key
    // 3. Create new VPS with custom configuration
    // 4. Wait for VPS provisioning (up to 5 minutes)
    // 5. Verify K3s cluster health
    // 6. Configure SSL for test domain
    // 7. Deploy test application
    // 8. Verify application accessibility
    // 9. Clean up: Delete VPS and DNS records
}
```

**Validation Points**:
- ✅ VPS creation with cloud-init script
- ✅ SSH connectivity establishment
- ✅ K3s cluster installation and health
- ✅ SSL certificate generation and installation
- ✅ DNS record creation and propagation
- ✅ Application deployment and accessibility
- ✅ Resource cleanup and cost tracking

##### Test Case: E2E_VPS_002 - VPS Configuration Management
```go
func TestE2E_VPSConfigurationUpdates(t *testing.T) {
    // Test Steps:
    // 1. Create base VPS with minimal configuration
    // 2. Update VPS configuration (add SSL domains)
    // 3. Deploy multiple applications
    // 4. Modify application configurations
    // 5. Verify configuration persistence
    // 6. Test VPS power operations (reboot)
    // 7. Verify configurations survive reboot
}
```

##### Test Case: E2E_VPS_003 - VPS Scaling and Management
```go
func TestE2E_VPSScalingOperations(t *testing.T) {
    // Test Steps:
    // 1. Create VPS with small server type
    // 2. Deploy resource-intensive application
    // 3. Monitor resource usage
    // 4. Scale VPS to larger server type (resize)
    // 5. Verify application continues running
    // 6. Test backup/restore operations
    // 7. Validate data persistence
}
```

#### 2.3.2 SSL Certificate Management Tests
**Priority**: High  
**Duration**: ~10 minutes per test  
**Resource Requirements**: Test Cloudflare zone with API access

##### Test Case: E2E_SSL_001 - Complete SSL Configuration Flow
```go
func TestE2E_SSLCertificateLifecycle(t *testing.T) {
    // Test Steps:
    // 1. Create VPS with basic configuration
    // 2. Configure SSL for test subdomain (ssl-test.example.com)
    // 3. Generate CSR and private key
    // 4. Create Cloudflare Origin Certificate
    // 5. Install certificates on VPS
    // 6. Configure Cloudflare SSL settings (Strict mode)
    // 7. Verify HTTPS connectivity
    // 8. Test SSL certificate validation
    // 9. Clean up SSL configuration
}
```

**Validation Points**:
- ✅ CSR generation with correct domain information
- ✅ Cloudflare Origin Certificate creation
- ✅ Certificate installation on VPS
- ✅ SSL mode configuration (Flexible → Strict)
- ✅ HTTPS redirect functionality
- ✅ Certificate chain validation
- ✅ SSL cleanup and rollback

##### Test Case: E2E_SSL_002 - Multi-Domain SSL Configuration
```go
func TestE2E_MultiDomainSSL(t *testing.T) {
    // Test Steps:
    // 1. Configure SSL for primary domain
    // 2. Add secondary domain to same VPS
    // 3. Configure wildcard SSL certificate
    // 4. Verify all domains use HTTPS
    // 5. Test domain-specific routing
    // 6. Remove one domain configuration
    // 7. Verify other domains unaffected
}
```

##### Test Case: E2E_SSL_003 - SSL Certificate Renewal
```go
func TestE2E_SSLCertificateRenewal(t *testing.T) {
    // Test Steps:
    // 1. Create SSL configuration with short-lived cert
    // 2. Wait for certificate expiration warning
    // 3. Trigger certificate renewal process
    // 4. Verify new certificate installation
    // 5. Test zero-downtime renewal
    // 6. Validate certificate chain continuity
}
```

#### 2.3.3 Application Deployment Tests
**Priority**: Medium  
**Duration**: ~8 minutes per test  
**Resource Requirements**: K3s cluster with Helm

##### Test Case: E2E_APP_001 - Complete Application Lifecycle
```go
func TestE2E_ApplicationDeployment(t *testing.T) {
    // Test Steps:
    // 1. Create VPS with K3s cluster
    // 2. Install Helm chart (nginx-ingress)
    // 3. Verify application deployment status
    // 4. Test application accessibility via ingress
    // 5. Upgrade application to newer version
    // 6. Verify upgrade success and zero downtime
    // 7. Roll back to previous version
    // 8. Uninstall application completely
    // 9. Verify cleanup of all resources
}
```

**Validation Points**:
- ✅ Helm repository addition and update
- ✅ Chart installation with custom values
- ✅ Pod and service deployment verification
- ✅ Ingress configuration and routing
- ✅ Application health checks
- ✅ Upgrade/rollback functionality
- ✅ Complete resource cleanup

##### Test Case: E2E_APP_002 - Multi-Application Deployment
```go
func TestE2E_MultiApplicationDeployment(t *testing.T) {
    // Test Steps:
    // 1. Deploy web application (nginx)
    // 2. Deploy database (postgresql)
    // 3. Deploy monitoring (prometheus)
    // 4. Configure inter-service communication
    // 5. Verify all applications running
    // 6. Test resource sharing and isolation
    // 7. Simulate application failure
    // 8. Verify automatic recovery
}
```

##### Test Case: E2E_APP_003 - Custom Manifest Deployment
```go
func TestE2E_CustomManifestDeployment(t *testing.T) {
    // Test Steps:
    // 1. Create custom Kubernetes manifest
    // 2. Deploy via VPS terminal interface
    // 3. Verify deployment success
    // 4. Test manifest updates and patches
    // 5. Monitor resource consumption
    // 6. Test scaling operations
    // 7. Clean up custom resources
}
```

#### 2.3.4 User Interface End-to-End Tests
**Priority**: Medium  
**Duration**: ~12 minutes per test  
**Resource Requirements**: Headless browser (if UI automation)

##### Test Case: E2E_UI_001 - Complete User Journey
```go
func TestE2E_UserInterfaceFlow(t *testing.T) {
    // Test Steps:
    // 1. Access login page
    // 2. Submit valid Cloudflare token
    // 3. Navigate to VPS creation page
    // 4. Fill VPS creation form
    // 5. Submit VPS creation request
    // 6. Monitor VPS creation progress
    // 7. Access VPS management page
    // 8. Configure SSL through UI
    // 9. Deploy application via UI
    // 10. Verify all UI elements update correctly
}
```

##### Test Case: E2E_UI_002 - Error Handling and Recovery
```go
func TestE2E_UIErrorHandling(t *testing.T) {
    // Test Steps:
    // 1. Submit invalid API credentials
    // 2. Verify error message display
    // 3. Attempt VPS creation with insufficient quota
    // 4. Test network timeout scenarios
    // 5. Verify graceful error handling
    // 6. Test recovery after temporary failures
}
```

#### 2.3.5 Performance and Load Tests
**Priority**: Low  
**Duration**: ~20 minutes per test  
**Resource Requirements**: Load testing tools, multiple test accounts

##### Test Case: E2E_PERF_001 - Concurrent Operations
```go
func TestE2E_ConcurrentVPSOperations(t *testing.T) {
    // Test Steps:
    // 1. Create multiple VPS instances simultaneously
    // 2. Configure SSL for multiple domains concurrently
    // 3. Deploy applications to multiple VPS
    // 4. Monitor system performance and resources
    // 5. Verify no resource conflicts or deadlocks
    // 6. Clean up all created resources
}
```

##### Test Case: E2E_PERF_002 - API Rate Limit Handling
```go
func TestE2E_APIRateLimitHandling(t *testing.T) {
    // Test Steps:
    // 1. Generate high-frequency API requests
    // 2. Trigger Hetzner/Cloudflare rate limits
    // 3. Verify graceful backoff and retry logic
    // 4. Test queue management for pending operations
    // 5. Verify operation completion after rate limit recovery
}
```

#### 2.3.6 Security End-to-End Tests
**Priority**: High  
**Duration**: ~15 minutes per test  
**Resource Requirements**: Security testing tools

##### Test Case: E2E_SEC_001 - Authentication Security
```go
func TestE2E_AuthenticationSecurity(t *testing.T) {
    // Test Steps:
    // 1. Test session management and timeout
    // 2. Verify secure cookie handling
    // 3. Test concurrent session limits
    // 4. Attempt token manipulation attacks
    // 5. Verify proper session cleanup on logout
    // 6. Test cross-site request forgery protection
}
```

##### Test Case: E2E_SEC_002 - Data Encryption Security
```go
func TestE2E_DataEncryptionSecurity(t *testing.T) {
    // Test Steps:
    // 1. Store sensitive data (API keys, SSH keys)
    // 2. Verify encryption at rest in Cloudflare KV
    // 3. Test encrypted data transmission
    // 4. Attempt data decryption with wrong tokens
    // 5. Verify secure key rotation procedures
}
```

#### 2.3.7 Disaster Recovery Tests
**Priority**: Medium  
**Duration**: ~25 minutes per test  
**Resource Requirements**: Backup/restore infrastructure

##### Test Case: E2E_DR_001 - VPS Recovery Scenarios
```go
func TestE2E_VPSDisasterRecovery(t *testing.T) {
    // Test Steps:
    // 1. Create VPS with applications and data
    // 2. Simulate VPS failure (force shutdown)
    // 3. Attempt VPS recovery procedures
    // 4. Verify data persistence and application recovery
    // 5. Test backup restoration processes
    // 6. Validate recovery time objectives (RTO)
}
```

##### Test Case: E2E_DR_002 - Service Dependency Failures
```go
func TestE2E_ServiceDependencyFailures(t *testing.T) {
    // Test Steps:
    // 1. Simulate Hetzner API outage
    // 2. Test graceful degradation of VPS operations
    // 3. Simulate Cloudflare API outage
    // 4. Verify SSL operations fallback behavior
    // 5. Test service recovery after outage resolution
}
```

#### 2.3.8 Test Implementation Framework

##### Test Structure
```
tests/integration/e2e/
├── vps_lifecycle_test.go
├── ssl_management_test.go
├── application_deployment_test.go
├── ui_integration_test.go
├── performance_test.go
├── security_test.go
├── disaster_recovery_test.go
├── helpers/
│   ├── test_setup.go
│   ├── cleanup.go
│   └── validation.go
└── fixtures/
    ├── test_configs/
    ├── sample_manifests/
    └── mock_responses/
```

##### Test Configuration
```go
type E2ETestConfig struct {
    HetznerAPIKey    string
    CloudflareToken  string
    TestDomain       string
    TestAccountID    string
    MaxTestDuration  time.Duration
    CleanupTimeout   time.Duration
    RetryAttempts    int
    ResourceLimits   ResourceLimits
}
```

##### Common Test Utilities
```go
// Helper functions for E2E tests
func SetupTestEnvironment() (*E2ETestConfig, error)
func CreateTestVPS(config *E2ETestConfig) (*VPSInstance, error)
func ValidateVPSHealth(vps *VPSInstance) error
func CleanupTestResources(config *E2ETestConfig) error
func WaitForCondition(condition func() bool, timeout time.Duration) error
```

#### 2.3.9 Test Execution Strategy

##### Parallel Execution
- **Resource Isolation**: Each test uses unique resource names
- **Account Separation**: Different tests use different test accounts
- **Cleanup Coordination**: Shared cleanup routines with resource locking

##### Test Data Management
- **Dynamic Resource Names**: Include timestamp and test ID
- **Cleanup Verification**: Verify all resources cleaned up after test
- **Cost Monitoring**: Track test costs and resource usage

##### Failure Handling
- **Automatic Cleanup**: Clean up resources even on test failure
- **Retry Logic**: Retry transient failures (network, API limits)
- **Detailed Logging**: Comprehensive logs for debugging failures

##### Test Reporting
```go
type E2ETestReport struct {
    TestName        string
    Duration        time.Duration
    ResourcesUsed   []string
    CostIncurred    float64
    FailureReasons  []string
    CleanupStatus   string
}
```

#### 2.3.10 Continuous Integration Integration

##### GitHub Actions Workflow
```yaml
name: E2E Tests
on:
  schedule:
    - cron: '0 2 * * *'  # Run nightly
  workflow_dispatch:    # Manual trigger
jobs:
  e2e-tests:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        test-suite: [vps, ssl, apps, security]
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
      - name: Run E2E Tests
        env:
          HETZNER_API_KEY: ${{ secrets.TEST_HETZNER_API_KEY }}
          CLOUDFLARE_TOKEN: ${{ secrets.TEST_CLOUDFLARE_TOKEN }}
        run: |
          make test-e2e-${{ matrix.test-suite }}
      - name: Upload Test Results
        uses: actions/upload-artifact@v3
        with:
          name: e2e-results-${{ matrix.test-suite }}
          path: test-results/
```

##### Test Environment Management
- **Dedicated Test Accounts**: Separate accounts for CI/CD testing
- **Resource Quotas**: Limited quotas to prevent runaway costs
- **Automated Cleanup**: Scheduled cleanup of orphaned test resources
- **Cost Alerts**: Monitoring and alerts for unexpected test costs

This comprehensive end-to-end testing strategy ensures that Xanthus works correctly in real-world scenarios while maintaining cost control and resource management.

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