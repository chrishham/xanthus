# Testing Architecture

## 📋 Purpose
Three-tier comprehensive testing strategy with mock and live testing capabilities for cost-aware development.

## 🏗️ Architecture

### Three-Tier Testing Strategy
```
tests/
├── unit/           # Component isolation tests
├── integration/    # Cross-component tests
└── integration/e2e/ # End-to-end workflow tests
```

### Testing Modes
```
Mock Mode:  Fast, free, safe for development
Live Mode:  Real APIs, costs money, production-like
```

## 🔧 Test Organization

### Unit Tests (`unit/`)
```
unit/
├── handlers/       # HTTP handler tests
├── services/       # Service layer tests
├── utils/          # Utility function tests
└── middleware/     # Middleware tests
```

### Integration Tests (`integration/`)
```
integration/
├── e2e/           # End-to-end scenarios
├── fixtures/      # Test data and responses
└── helpers/       # Test utilities
```

### E2E Test Categories
```
integration/e2e/
├── application_deployment_test.go  # App deployment workflows
├── vps_lifecycle_test.go          # VPS creation/deletion
├── ssl_management_test.go         # SSL certificate management
├── ui_integration_test.go         # Frontend integration
├── performance_test.go            # Performance benchmarks
├── security_test.go               # Security validations
└── disaster_recovery_test.go      # DR scenarios
```

## 🎯 Test Commands

### Quick Testing (< 5 minutes)
```bash
make test           # Unit + integration (excludes E2E)
make test-unit      # Unit tests only
make test-integration # Integration tests only
```

### Comprehensive Testing
```bash
make test-e2e       # E2E tests in mock mode
make test-e2e-live  # E2E tests with real APIs (costs money)
make test-everything # All tests including E2E
```

### Specialized Test Suites
```bash
make test-e2e-vps       # VPS lifecycle tests
make test-e2e-ssl       # SSL certificate management
make test-e2e-apps      # Application deployment
make test-e2e-ui        # UI integration tests
make test-e2e-perf      # Performance tests
make test-e2e-security  # Security tests
make test-e2e-dr        # Disaster recovery tests
```

## 📊 Test Environment Configuration

### Environment Variables
```bash
# Test mode selection
E2E_TEST_MODE=mock|live     # Default: mock

# Live API credentials (for live mode only)
TEST_HETZNER_API_KEY=token
TEST_CLOUDFLARE_TOKEN=token
TEST_CLOUDFLARE_ACCOUNT_ID=account_id
TEST_DOMAIN=test.xanthus.local
```

### Mock vs Live Mode
```go
// Test configuration
if os.Getenv("E2E_TEST_MODE") == "live" {
    // Use real API endpoints
    config.UseRealAPIs = true
} else {
    // Use mock responses
    config.UseMockAPIs = true
}
```

## 🔧 Key Test Components

### Test Fixtures (`fixtures/`)
```
fixtures/
├── mock_responses/
│   ├── cloudflare_responses.json
│   └── hetzner_responses.json
├── sample_manifests/
│   └── test-nginx.yaml
└── test_configs/
    └── e2e_config.json
```

### Test Helpers (`helpers/`)
```go
// test_setup.go
func SetupTestEnvironment() *TestConfig {
    return &TestConfig{
        BaseURL:    "http://localhost:8081",
        AuthToken:  getTestToken(),
        TestMode:   getTestMode(),
    }
}

// cleanup.go
func CleanupTestResources(config *TestConfig) {
    // Clean up VPS instances
    // Clean up DNS records
    // Clean up applications
}

// validation.go
func ValidateApplicationDeployment(appID string) error {
    // Validate Kubernetes deployment
    // Validate DNS resolution
    // Validate SSL certificate
}
```

## 🎯 Test Scenarios

### Application Deployment Test (`application_deployment_test.go`)
```go
func TestApplicationDeployment(t *testing.T) {
    // 1. Create VPS instance
    vps := createTestVPS(t)
    defer cleanupVPS(t, vps.ID)
    
    // 2. Deploy code-server application
    app := deployApplication(t, "code-server", vps.ID)
    defer cleanupApplication(t, app.ID)
    
    // 3. Validate deployment
    validateDeployment(t, app)
    
    // 4. Test application access
    testApplicationAccess(t, app)
}
```

### VPS Lifecycle Test (`vps_lifecycle_test.go`)
```go
func TestVPSLifecycle(t *testing.T) {
    // 1. Create VPS
    vps := createVPS(t, "test-vps")
    
    // 2. Validate creation
    validateVPSCreation(t, vps)
    
    // 3. Start/Stop operations
    testVPSOperations(t, vps)
    
    // 4. Delete VPS
    deleteVPS(t, vps.ID)
    
    // 5. Validate cleanup
    validateVPSCleanup(t, vps.ID)
}
```

### SSL Management Test (`ssl_management_test.go`)
```go
func TestSSLManagement(t *testing.T) {
    // 1. Create test domain
    domain := createTestDomain(t)
    defer cleanupDomain(t, domain)
    
    // 2. Generate SSL certificate
    cert := generateSSLCertificate(t, domain)
    
    // 3. Validate certificate
    validateSSLCertificate(t, cert)
    
    // 4. Test certificate renewal
    testCertificateRenewal(t, cert)
}
```

## 🔄 Mock Testing Strategy

### Mock Response System
```go
// Mock HTTP responses for external APIs
type MockTransport struct {
    responses map[string]*http.Response
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    key := fmt.Sprintf("%s:%s", req.Method, req.URL.String())
    
    if response, exists := m.responses[key]; exists {
        return response, nil
    }
    
    return &http.Response{
        StatusCode: 404,
        Body:       ioutil.NopCloser(strings.NewReader("Not Found")),
    }, nil
}
```

### Mock Data Loading
```go
// Load mock responses from fixtures
func loadMockResponses() map[string]*http.Response {
    responses := make(map[string]*http.Response)
    
    // Load Cloudflare responses
    cfData := loadFixture("mock_responses/cloudflare_responses.json")
    responses["GET:https://api.cloudflare.com/client/v4/zones"] = createMockResponse(cfData)
    
    // Load Hetzner responses
    hetznerData := loadFixture("mock_responses/hetzner_responses.json")
    responses["GET:https://api.hetzner.cloud/v1/servers"] = createMockResponse(hetznerData)
    
    return responses
}
```

## 📈 Performance Testing

### Load Testing (`performance_test.go`)
```go
func TestApplicationDeploymentPerformance(t *testing.T) {
    // Test concurrent application deployments
    concurrency := 5
    var wg sync.WaitGroup
    
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func(index int) {
            defer wg.Done()
            
            // Deploy application
            start := time.Now()
            app := deployApplication(t, fmt.Sprintf("test-app-%d", index))
            duration := time.Since(start)
            
            // Validate performance
            if duration > 5*time.Minute {
                t.Errorf("Deployment too slow: %v", duration)
            }
            
            // Cleanup
            cleanupApplication(t, app.ID)
        }(i)
    }
    
    wg.Wait()
}
```

### Memory Usage Testing
```go
func TestMemoryUsage(t *testing.T) {
    // Monitor memory usage during operations
    var memStats runtime.MemStats
    runtime.ReadMemStats(&memStats)
    
    initialMemory := memStats.Alloc
    
    // Perform operations
    for i := 0; i < 100; i++ {
        deployApplication(t, fmt.Sprintf("test-%d", i))
    }
    
    runtime.ReadMemStats(&memStats)
    finalMemory := memStats.Alloc
    
    memoryIncrease := finalMemory - initialMemory
    if memoryIncrease > 100*1024*1024 { // 100MB
        t.Errorf("Memory usage too high: %d bytes", memoryIncrease)
    }
}
```

## 🔒 Security Testing

### Security Validation (`security_test.go`)
```go
func TestSecurityValidation(t *testing.T) {
    // Test authentication
    testAuthenticationSecurity(t)
    
    // Test authorization
    testAuthorizationSecurity(t)
    
    // Test input validation
    testInputValidation(t)
    
    // Test secret management
    testSecretManagement(t)
}

func testAuthenticationSecurity(t *testing.T) {
    // Test invalid token rejection
    response := makeRequestWithToken(t, "invalid-token")
    assert.Equal(t, 401, response.StatusCode)
    
    // Test token expiration
    expiredToken := generateExpiredToken()
    response = makeRequestWithToken(t, expiredToken)
    assert.Equal(t, 401, response.StatusCode)
}
```

## 🛠️ Test Utilities

### Test Configuration
```go
type TestConfig struct {
    BaseURL     string
    AuthToken   string
    TestMode    string
    Timeout     time.Duration
    RetryCount  int
    CleanupFunc func()
}

func NewTestConfig() *TestConfig {
    return &TestConfig{
        BaseURL:    getEnv("TEST_BASE_URL", "http://localhost:8081"),
        AuthToken:  getEnv("TEST_AUTH_TOKEN", "test-token"),
        TestMode:   getEnv("E2E_TEST_MODE", "mock"),
        Timeout:    30 * time.Second,
        RetryCount: 3,
    }
}
```

### Test Assertions
```go
// Custom assertions for application testing
func AssertApplicationDeployed(t *testing.T, appID string) {
    app := getApplication(t, appID)
    assert.Equal(t, "deployed", app.Status)
    assert.NotEmpty(t, app.URL)
}

func AssertApplicationAccessible(t *testing.T, appURL string) {
    resp, err := http.Get(appURL)
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
}

func AssertVPSRunning(t *testing.T, vpsID string) {
    vps := getVPS(t, vpsID)
    assert.Equal(t, "running", vps.Status)
    assert.NotEmpty(t, vps.IPAddress)
}
```

## 📊 Test Coverage

### Coverage Commands
```bash
make test-coverage    # Generate coverage reports
# Creates coverage.html for viewing
```

### Coverage Targets
- **Unit tests**: > 80% coverage
- **Integration tests**: > 70% coverage
- **E2E tests**: > 60% coverage
- **Critical paths**: > 95% coverage

## 🔧 Test Maintenance

### Test Data Management
```go
// Clean test data between runs
func cleanupTestData(t *testing.T) {
    // Remove test applications
    cleanupTestApplications(t)
    
    // Remove test VPS instances
    cleanupTestVPS(t)
    
    // Remove test DNS records
    cleanupTestDNS(t)
}
```

### Test Environment Reset
```go
// Reset test environment to known state
func resetTestEnvironment(t *testing.T) {
    // Clear KV store test data
    clearTestKVData(t)
    
    // Reset test configurations
    resetTestConfigs(t)
    
    // Initialize test fixtures
    initTestFixtures(t)
}
```

## 🚀 Running Tests

### Local Development
```bash
# Quick feedback loop
make test-unit

# Integration testing
make test-integration

# Full testing (mock mode)
make test-e2e
```

### CI/CD Pipeline
```bash
# Automated testing
make test-everything E2E_TEST_MODE=mock

# Production validation (careful - costs money)
make test-e2e-live
```

### Test Debugging
```bash
# Verbose output
go test -v ./tests/unit/...

# Run specific test
go test -v -run TestApplicationDeployment ./tests/integration/e2e/

# Debug with race detection
go test -race -v ./tests/...
```

## 📈 Test Performance

### Optimization Strategies
- **Parallel execution** - Run independent tests concurrently
- **Mock responses** - Avoid external API calls in unit tests
- **Resource pooling** - Reuse expensive resources like VPS instances
- **Selective testing** - Run only relevant tests for specific changes

### Test Timing
- **Unit tests**: < 30 seconds
- **Integration tests**: < 2 minutes
- **E2E tests (mock)**: < 5 minutes
- **E2E tests (live)**: < 15 minutes

## 🔒 Test Security

### Credential Management
- **No hardcoded secrets** in test files
- **Environment variables** for test credentials
- **Separate test accounts** for live testing
- **Automatic cleanup** of test resources

### Test Isolation
- **Unique test prefixes** for resource naming
- **Cleanup functions** for proper resource disposal
- **Mock mode by default** to prevent accidental live API usage
- **Test data segregation** from production data