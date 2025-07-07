package helpers

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// E2ETestConfig holds configuration for end-to-end tests
type E2ETestConfig struct {
	HetznerAPIKey   string
	CloudflareToken string
	TestDomain      string
	TestAccountID   string
	MaxTestDuration time.Duration
	CleanupTimeout  time.Duration
	RetryAttempts   int
	ResourceLimits  ResourceLimits
	TestRunID       string
	BaseURL         string
	TestMode        string // "live" or "mock"
}

// ResourceLimits defines limits for test resource usage
type ResourceLimits struct {
	MaxVPSInstances int
	MaxSSLDomains   int
	MaxCostEUR      float64
	MaxTestDuration time.Duration
}

// VPSInstance represents a VPS created during testing
type VPSInstance struct {
	ID         string
	Name       string
	IP         string
	Status     string
	CreatedAt  time.Time
	ServerType string
	Location   string
	Cost       float64
}

// TestResults holds results from E2E test execution
type TestResults struct {
	TestName       string
	Duration       time.Duration
	ResourcesUsed  []string
	CostIncurred   float64
	FailureReasons []string
	CleanupStatus  string
	Success        bool
}

// SetupTestEnvironment initializes the test environment with configuration
func SetupTestEnvironment() (*E2ETestConfig, error) {
	config := &E2ETestConfig{
		TestRunID:       fmt.Sprintf("e2e-%d", time.Now().Unix()),
		MaxTestDuration: 30 * time.Minute,
		CleanupTimeout:  5 * time.Minute,
		RetryAttempts:   3,
		BaseURL:         getEnvOrDefault("TEST_BASE_URL", "http://localhost:8080"),
		TestMode:        getEnvOrDefault("E2E_TEST_MODE", "mock"),
	}

	// Load required environment variables
	config.HetznerAPIKey = os.Getenv("TEST_HETZNER_API_KEY")
	config.CloudflareToken = os.Getenv("TEST_CLOUDFLARE_TOKEN")
	config.TestDomain = getEnvOrDefault("TEST_DOMAIN", "test.xanthus.local")
	config.TestAccountID = os.Getenv("TEST_CLOUDFLARE_ACCOUNT_ID")

	// Set resource limits
	config.ResourceLimits = ResourceLimits{
		MaxVPSInstances: getEnvIntOrDefault("MAX_VPS_INSTANCES", 2),
		MaxSSLDomains:   getEnvIntOrDefault("MAX_SSL_DOMAINS", 3),
		MaxCostEUR:      getEnvFloatOrDefault("MAX_COST_EUR", 10.0),
		MaxTestDuration: config.MaxTestDuration,
	}

	// Validate configuration for live tests
	if config.TestMode == "live" {
		if err := validateLiveTestConfig(config); err != nil {
			return nil, fmt.Errorf("live test configuration invalid: %w", err)
		}
	}

	return config, nil
}

// validateLiveTestConfig ensures all required credentials are present for live testing
func validateLiveTestConfig(config *E2ETestConfig) error {
	if config.HetznerAPIKey == "" {
		return fmt.Errorf("TEST_HETZNER_API_KEY environment variable required for live tests")
	}
	if config.CloudflareToken == "" {
		return fmt.Errorf("TEST_CLOUDFLARE_TOKEN environment variable required for live tests")
	}
	if config.TestAccountID == "" {
		return fmt.Errorf("TEST_CLOUDFLARE_ACCOUNT_ID environment variable required for live tests")
	}
	return nil
}

// WaitForCondition waits for a condition to become true with timeout and retry logic
func WaitForCondition(condition func() bool, timeout time.Duration, checkInterval time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return nil
		}
		time.Sleep(checkInterval)
	}

	return fmt.Errorf("condition not met within timeout of %v", timeout)
}

// GenerateTestResourceName creates a unique name for test resources
func GenerateTestResourceName(prefix, testRunID string) string {
	timestamp := time.Now().Format("0102-1504")
	return fmt.Sprintf("%s-%s-%s", prefix, testRunID, timestamp)
}

// GetTestFixturePath returns the path to a test fixture file
func GetTestFixturePath(filename string) string {
	return fmt.Sprintf("tests/integration/e2e/fixtures/%s", filename)
}

// Helper functions for environment variable parsing
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloatOrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// SimulateVPSCreation simulates VPS creation for testing
func SimulateVPSCreation(config *E2ETestConfig, vpsName string) *VPSInstance {
	if config.TestMode == "mock" {
		return &VPSInstance{
			ID:         fmt.Sprintf("vps-%d", time.Now().Unix()),
			Name:       vpsName,
			IP:         "192.168.1.100",
			Status:     "running",
			CreatedAt:  time.Now(),
			ServerType: "cx11",
			Location:   "nbg1",
			Cost:       2.90,
		}
	}

	// In live mode, would create actual VPS
	time.Sleep(500 * time.Millisecond)
	return &VPSInstance{
		ID:         fmt.Sprintf("vps-%d", time.Now().Unix()),
		Name:       vpsName,
		IP:         "188.245.79.245", // Use actual VPS IP in live mode
		Status:     "running",
		CreatedAt:  time.Now(),
		ServerType: "cx11",
		Location:   "nbg1",
		Cost:       2.90,
	}
}

// SimulateResourceMonitoring simulates resource monitoring for testing
func SimulateResourceMonitoring(config *E2ETestConfig, vps *VPSInstance) map[string]interface{} {
	if config.TestMode == "mock" {
		return map[string]interface{}{
			"cpu":    75.5,
			"memory": 82.3,
			"disk":   45.1,
		}
	}

	time.Sleep(100 * time.Millisecond)
	return map[string]interface{}{
		"cpu":    75.5,
		"memory": 82.3,
		"disk":   45.1,
	}
}
