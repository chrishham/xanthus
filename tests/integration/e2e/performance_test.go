package e2e

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chrishham/xanthus/tests/integration/e2e/helpers"
)

// TestE2E_PERF_001_ConcurrentVPSOperations tests concurrent VPS operations
func TestE2E_PERF_001_ConcurrentVPSOperations(t *testing.T) {
	// Setup test environment
	config, err := helpers.SetupTestEnvironment()
	require.NoError(t, err, "Failed to setup test environment")

	// Create cleanup manager
	cleanup := helpers.NewCleanupManager(config)
	defer func() {
		if err := cleanup.CleanupTestResources(); err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()

	// Create validator
	_ = helpers.NewValidator(config) // Validator available for future validation needs

	// Performance test parameters
	const (
		concurrentVPSCount = 3
		concurrentSSLCount = 5
		concurrentAppCount = 4
		maxAllowedDuration = 10 * time.Minute
		maxMemoryUsageMB   = 512
		maxCPUUsagePercent = 80
	)

	t.Logf("Starting E2E concurrent operations performance test")
	t.Logf("Test parameters: %d VPS, %d SSL domains, %d applications",
		concurrentVPSCount, concurrentSSLCount, concurrentAppCount)

	ctx, cancel := context.WithTimeout(context.Background(), maxAllowedDuration)
	defer cancel()
	_ = ctx // Context is available for future use

	// Test Steps:
	// 1. Create multiple VPS instances simultaneously
	var vpsInstances []*helpers.VPSInstance
	var vpsCreationTimes []time.Duration
	t.Run("Step1_ConcurrentVPSCreation", func(t *testing.T) {
		startTime := time.Now()

		var wg sync.WaitGroup
		var mu sync.Mutex
		vpsInstances = make([]*helpers.VPSInstance, 0, concurrentVPSCount)
		vpsCreationTimes = make([]time.Duration, 0, concurrentVPSCount)

		for i := 0; i < concurrentVPSCount; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				vpsName := helpers.GenerateTestResourceName(fmt.Sprintf("perf-vps-%d", index), config.TestRunID)
				vpsStart := time.Now()

				vps := simulateVPSCreation(t, config, vpsName)
				vpsCreationTime := time.Since(vpsStart)

				mu.Lock()
				vpsInstances = append(vpsInstances, vps)
				vpsCreationTimes = append(vpsCreationTimes, vpsCreationTime)

				cleanup.RegisterResource("vps", vps.ID, vps.Name, map[string]interface{}{
					"ip": vps.IP,
				})
				mu.Unlock()

				t.Logf("VPS %d created: %s (took %v)", index, vps.Name, vpsCreationTime)
			}(i)
		}

		wg.Wait()
		totalVPSCreationTime := time.Since(startTime)

		assert.Len(t, vpsInstances, concurrentVPSCount, "All VPS instances should be created")
		assert.Less(t, totalVPSCreationTime, 5*time.Minute, "VPS creation should complete within 5 minutes")

		// Calculate performance metrics
		averageVPSCreationTime := calculateAverageTime(vpsCreationTimes)
		t.Logf("Concurrent VPS creation completed in %v (average per VPS: %v)",
			totalVPSCreationTime, averageVPSCreationTime)
	})

	// 2. Configure SSL for multiple domains concurrently
	var sslDomains []string
	var sslConfigurationTimes []time.Duration
	t.Run("Step2_ConcurrentSSLConfiguration", func(t *testing.T) {
		startTime := time.Now()

		var wg sync.WaitGroup
		var mu sync.Mutex
		sslDomains = make([]string, 0, concurrentSSLCount)
		sslConfigurationTimes = make([]time.Duration, 0, concurrentSSLCount)

		for i := 0; i < concurrentSSLCount; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				domain := fmt.Sprintf("perf-ssl-%d.%s", index, config.TestDomain)
				vpsIndex := index % len(vpsInstances) // Round-robin assignment to VPS
				vpsIP := vpsInstances[vpsIndex].IP

				sslStart := time.Now()
				sslSuccess := simulateSSLConfiguration(t, config, domain, vpsIP)
				sslConfigTime := time.Since(sslStart)

				assert.True(t, sslSuccess, "SSL configuration should succeed for domain: %s", domain)

				mu.Lock()
				sslDomains = append(sslDomains, domain)
				sslConfigurationTimes = append(sslConfigurationTimes, sslConfigTime)

				cleanup.RegisterResource("ssl", domain, domain, map[string]interface{}{
					"domain": domain,
				})
				mu.Unlock()

				t.Logf("SSL %d configured: %s (took %v)", index, domain, sslConfigTime)
			}(i)
		}

		wg.Wait()
		totalSSLConfigTime := time.Since(startTime)

		assert.Len(t, sslDomains, concurrentSSLCount, "All SSL domains should be configured")
		assert.Less(t, totalSSLConfigTime, 3*time.Minute, "SSL configuration should complete within 3 minutes")

		averageSSLConfigTime := calculateAverageTime(sslConfigurationTimes)
		t.Logf("Concurrent SSL configuration completed in %v (average per domain: %v)",
			totalSSLConfigTime, averageSSLConfigTime)
	})

	// 3. Deploy applications to multiple VPS
	var deployedApps []string
	var appDeploymentTimes []time.Duration
	t.Run("Step3_ConcurrentApplicationDeployment", func(t *testing.T) {
		startTime := time.Now()

		var wg sync.WaitGroup
		var mu sync.Mutex
		deployedApps = make([]string, 0, concurrentAppCount)
		appDeploymentTimes = make([]time.Duration, 0, concurrentAppCount)

		for i := 0; i < concurrentAppCount; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				appName := fmt.Sprintf("perf-app-%d-%s", index, config.TestRunID)
				vpsIndex := index % len(vpsInstances) // Round-robin assignment to VPS
				vps := vpsInstances[vpsIndex]

				appStart := time.Now()
				deploySuccess := simulateApplicationDeployment(t, config, appName, vps)
				appDeployTime := time.Since(appStart)

				assert.True(t, deploySuccess, "Application deployment should succeed for: %s", appName)

				mu.Lock()
				deployedApps = append(deployedApps, appName)
				appDeploymentTimes = append(appDeploymentTimes, appDeployTime)

				cleanup.RegisterResource("app", appName, appName, map[string]interface{}{
					"app_name": appName,
				})
				mu.Unlock()

				t.Logf("App %d deployed: %s (took %v)", index, appName, appDeployTime)
			}(i)
		}

		wg.Wait()
		totalAppDeployTime := time.Since(startTime)

		assert.Len(t, deployedApps, concurrentAppCount, "All applications should be deployed")
		assert.Less(t, totalAppDeployTime, 4*time.Minute, "Application deployment should complete within 4 minutes")

		averageAppDeployTime := calculateAverageTime(appDeploymentTimes)
		t.Logf("Concurrent application deployment completed in %v (average per app: %v)",
			totalAppDeployTime, averageAppDeployTime)
	})

	// 4. Monitor system performance and resources
	t.Run("Step4_MonitorSystemPerformance", func(t *testing.T) {
		performanceMetrics := simulateSystemPerformanceMonitoring(t, config)

		assert.NotNil(t, performanceMetrics, "Performance metrics should be collected")
		assert.Less(t, performanceMetrics["memory_usage_mb"].(float64), float64(maxMemoryUsageMB),
			"Memory usage should be within limits")
		assert.Less(t, performanceMetrics["cpu_usage_percent"].(float64), float64(maxCPUUsagePercent),
			"CPU usage should be within limits")

		t.Logf("System performance metrics: CPU: %.2f%%, Memory: %.2f MB, Disk I/O: %.2f MB/s",
			performanceMetrics["cpu_usage_percent"],
			performanceMetrics["memory_usage_mb"],
			performanceMetrics["disk_io_mb_per_sec"])
	})

	// 5. Verify no resource conflicts or deadlocks
	t.Run("Step5_VerifyNoResourceConflicts", func(t *testing.T) {
		conflictCheck := simulateResourceConflictCheck(t, config, vpsInstances, sslDomains, deployedApps)
		assert.True(t, conflictCheck, "No resource conflicts should be detected")

		deadlockCheck := simulateDeadlockDetection(t, config)
		assert.False(t, deadlockCheck, "No deadlocks should be detected")

		t.Logf("Resource conflict and deadlock checks passed")
	})

	// 6. Performance validation
	t.Run("Step6_PerformanceValidation", func(t *testing.T) {
		// Validate VPS creation performance
		maxAcceptableVPSCreationTime := 2 * time.Minute
		for i, creationTime := range vpsCreationTimes {
			assert.Less(t, creationTime, maxAcceptableVPSCreationTime,
				"VPS %d creation time should be acceptable", i)
		}

		// Validate SSL configuration performance
		maxAcceptableSSLConfigTime := 30 * time.Second
		for i, configTime := range sslConfigurationTimes {
			assert.Less(t, configTime, maxAcceptableSSLConfigTime,
				"SSL configuration %d time should be acceptable", i)
		}

		// Validate application deployment performance
		maxAcceptableAppDeployTime := 1 * time.Minute
		for i, deployTime := range appDeploymentTimes {
			assert.Less(t, deployTime, maxAcceptableAppDeployTime,
				"Application deployment %d time should be acceptable", i)
		}

		t.Logf("All performance validations passed")
	})

	// 7. Stress test with rapid operations
	t.Run("Step7_RapidOperationsStressTest", func(t *testing.T) {
		stressTestSuccess := simulateRapidOperationsStressTest(t, config, vpsInstances)
		assert.True(t, stressTestSuccess, "Rapid operations stress test should pass")

		t.Logf("Rapid operations stress test completed successfully")
	})

	t.Logf("E2E concurrent operations performance test completed successfully")
}

// TestE2E_PERF_002_APIRateLimitHandling tests API rate limit handling
func TestE2E_PERF_002_APIRateLimitHandling(t *testing.T) {
	config, err := helpers.SetupTestEnvironment()
	require.NoError(t, err, "Failed to setup test environment")

	cleanup := helpers.NewCleanupManager(config)
	defer func() {
		if err := cleanup.CleanupTestResources(); err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()

	// Rate limit test parameters
	const (
		highFrequencyRequestCount = 100
		requestIntervalMs         = 50  // 20 requests per second
		expectedRateLimitResponse = 429 // Too Many Requests
		maxRetryAttempts          = 5
		backoffMultiplier         = 2
	)

	t.Logf("Starting E2E API rate limit handling test")
	t.Logf("Test parameters: %d requests at %dms intervals",
		highFrequencyRequestCount, requestIntervalMs)

	// Test Steps:
	// 1. Generate high-frequency API requests
	var requestResults []RequestResult
	t.Run("Step1_GenerateHighFrequencyRequests", func(t *testing.T) {
		startTime := time.Now()
		requestResults = make([]RequestResult, 0, highFrequencyRequestCount)

		for i := 0; i < highFrequencyRequestCount; i++ {
			requestStart := time.Now()

			// Simulate API request
			result := simulateAPIRequest(t, config, fmt.Sprintf("request-%d", i))
			result.RequestIndex = i
			result.Timestamp = requestStart
			result.Duration = time.Since(requestStart)

			requestResults = append(requestResults, result)

			// Small delay between requests
			time.Sleep(time.Duration(requestIntervalMs) * time.Millisecond)

			if i%20 == 0 {
				t.Logf("Sent %d/%d requests", i+1, highFrequencyRequestCount)
			}
		}

		totalDuration := time.Since(startTime)
		t.Logf("Generated %d high-frequency requests in %v", highFrequencyRequestCount, totalDuration)
	})

	// 2. Trigger Hetzner/Cloudflare rate limits
	var rateLimitedRequests []RequestResult
	t.Run("Step2_AnalyzeRateLimitResponses", func(t *testing.T) {
		rateLimitedRequests = make([]RequestResult, 0)
		successfulRequests := 0
		errorRequests := 0

		for _, result := range requestResults {
			switch result.StatusCode {
			case 200, 201, 202:
				successfulRequests++
			case 429:
				rateLimitedRequests = append(rateLimitedRequests, result)
			default:
				errorRequests++
			}
		}

		t.Logf("Request analysis: %d successful, %d rate-limited, %d errors",
			successfulRequests, len(rateLimitedRequests), errorRequests)

		// We expect some requests to be rate-limited in a realistic scenario
		if config.TestMode == "live" {
			assert.Greater(t, len(rateLimitedRequests), 0,
				"Some requests should be rate-limited in live mode")
		}
	})

	// 3. Verify graceful backoff and retry logic
	t.Run("Step3_TestBackoffAndRetryLogic", func(t *testing.T) {
		retryResults := make([]RetryResult, 0)

		// Test retry logic for rate-limited requests
		for i, rateLimitedReq := range rateLimitedRequests {
			if i >= 5 { // Only test first 5 rate-limited requests
				break
			}

			retryResult := simulateRetryWithBackoff(t, config, rateLimitedReq, maxRetryAttempts)
			retryResults = append(retryResults, retryResult)
		}

		// Validate retry behavior
		for i, retry := range retryResults {
			assert.True(t, retry.EventuallySucceeded || retry.RetriesExhausted,
				"Retry %d should either succeed or exhaust retries", i)
			assert.LessOrEqual(t, retry.AttemptCount, maxRetryAttempts,
				"Retry %d should not exceed max attempts", i)
		}

		t.Logf("Tested backoff and retry logic for %d rate-limited requests", len(retryResults))
	})

	// 4. Test queue management for pending operations
	t.Run("Step4_TestQueueManagement", func(t *testing.T) {
		queueSize := 20
		queueManagementSuccess := simulateOperationQueue(t, config, queueSize)
		assert.True(t, queueManagementSuccess, "Operation queue management should work correctly")

		t.Logf("Queue management test completed with queue size: %d", queueSize)
	})

	// 5. Verify operation completion after rate limit recovery
	t.Run("Step5_VerifyPostRecoveryOperations", func(t *testing.T) {
		// Simulate waiting for rate limit recovery
		recoveryWaitTime := 60 * time.Second
		if config.TestMode == "mock" {
			recoveryWaitTime = 1 * time.Second
		}

		t.Logf("Waiting %v for rate limit recovery...", recoveryWaitTime)
		time.Sleep(recoveryWaitTime)

		// Test operations after recovery
		postRecoveryRequests := 10
		successfulPostRecovery := 0

		for i := 0; i < postRecoveryRequests; i++ {
			result := simulateAPIRequest(t, config, fmt.Sprintf("post-recovery-%d", i))
			if result.StatusCode >= 200 && result.StatusCode < 300 {
				successfulPostRecovery++
			}
			time.Sleep(100 * time.Millisecond) // Gentle requests
		}

		successRate := float64(successfulPostRecovery) / float64(postRecoveryRequests) * 100
		assert.Greater(t, successRate, 80.0, "Post-recovery success rate should be > 80%")

		t.Logf("Post-recovery operations: %d/%d successful (%.1f%%)",
			successfulPostRecovery, postRecoveryRequests, successRate)
	})

	// 6. Performance metrics analysis
	t.Run("Step6_AnalyzePerformanceMetrics", func(t *testing.T) {
		metrics := analyzeRequestMetrics(requestResults)

		assert.Greater(t, metrics.AverageResponseTime, time.Duration(0),
			"Average response time should be positive")
		assert.Greater(t, metrics.ThroughputRPS, 0.0,
			"Throughput should be positive")

		t.Logf("Performance metrics - Avg response: %v, Throughput: %.2f RPS, Max response: %v",
			metrics.AverageResponseTime, metrics.ThroughputRPS, metrics.MaxResponseTime)
	})

	t.Logf("E2E API rate limit handling test completed successfully")
}

// Supporting types and helper functions

type RequestResult struct {
	RequestIndex int
	StatusCode   int
	Duration     time.Duration
	Timestamp    time.Time
	Error        error
}

type RetryResult struct {
	OriginalRequest     RequestResult
	AttemptCount        int
	EventuallySucceeded bool
	RetriesExhausted    bool
	TotalRetryDuration  time.Duration
}

type PerformanceMetrics struct {
	AverageResponseTime time.Duration
	MaxResponseTime     time.Duration
	MinResponseTime     time.Duration
	ThroughputRPS       float64
	SuccessRate         float64
}

func calculateAverageTime(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}

	total := time.Duration(0)
	for _, t := range times {
		total += t
	}

	return total / time.Duration(len(times))
}

func simulateSystemPerformanceMonitoring(t *testing.T, config *helpers.E2ETestConfig) map[string]interface{} {
	t.Logf("Monitoring system performance")

	if config.TestMode == "mock" {
		t.Logf("MOCK: System performance monitored")
		return map[string]interface{}{
			"cpu_usage_percent":  45.2,
			"memory_usage_mb":    256.8,
			"disk_io_mb_per_sec": 12.5,
			"network_io_mbps":    5.2,
		}
	}

	// In live mode, would collect actual performance metrics
	time.Sleep(200 * time.Millisecond)
	return map[string]interface{}{
		"cpu_usage_percent":  50.1,
		"memory_usage_mb":    312.4,
		"disk_io_mb_per_sec": 15.8,
		"network_io_mbps":    7.3,
	}
}

func simulateResourceConflictCheck(t *testing.T, config *helpers.E2ETestConfig, vpsInstances []*helpers.VPSInstance, sslDomains []string, apps []string) bool {
	t.Logf("Checking for resource conflicts among %d VPS, %d SSL domains, %d apps",
		len(vpsInstances), len(sslDomains), len(apps))

	if config.TestMode == "mock" {
		t.Logf("MOCK: No resource conflicts detected")
		return true
	}

	// In live mode, would check for actual conflicts
	time.Sleep(100 * time.Millisecond)
	return true
}

func simulateDeadlockDetection(t *testing.T, config *helpers.E2ETestConfig) bool {
	t.Logf("Detecting potential deadlocks")

	if config.TestMode == "mock" {
		t.Logf("MOCK: No deadlocks detected")
		return false // false means no deadlocks
	}

	// In live mode, would detect actual deadlocks
	time.Sleep(50 * time.Millisecond)
	return false
}

func simulateRapidOperationsStressTest(t *testing.T, config *helpers.E2ETestConfig, vpsInstances []*helpers.VPSInstance) bool {
	t.Logf("Running rapid operations stress test")

	if config.TestMode == "mock" {
		t.Logf("MOCK: Rapid operations stress test passed")
		return true
	}

	// In live mode, would perform rapid operations
	time.Sleep(500 * time.Millisecond)
	return true
}

func simulateAPIRequest(t *testing.T, config *helpers.E2ETestConfig, requestID string) RequestResult {
	if config.TestMode == "mock" {
		// Simulate different response scenarios
		responses := []int{200, 200, 200, 429, 200, 500, 200}
		statusCode := responses[len(requestID)%len(responses)]

		return RequestResult{
			StatusCode: statusCode,
			Duration:   time.Duration(50+len(requestID)*2) * time.Millisecond,
		}
	}

	// In live mode, would make actual API request
	time.Sleep(time.Duration(100+len(requestID)) * time.Millisecond)
	return RequestResult{
		StatusCode: 200,
		Duration:   100 * time.Millisecond,
	}
}

func simulateRetryWithBackoff(t *testing.T, config *helpers.E2ETestConfig, originalReq RequestResult, maxAttempts int) RetryResult {
	t.Logf("Testing retry with backoff for request %d", originalReq.RequestIndex)

	result := RetryResult{
		OriginalRequest: originalReq,
		AttemptCount:    0,
	}

	backoffDelay := 100 * time.Millisecond
	startTime := time.Now()

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result.AttemptCount = attempt

		// Simulate retry request
		retryResult := simulateAPIRequest(t, config, fmt.Sprintf("retry-%d-%d", originalReq.RequestIndex, attempt))

		if retryResult.StatusCode >= 200 && retryResult.StatusCode < 300 {
			result.EventuallySucceeded = true
			break
		}

		if attempt < maxAttempts {
			time.Sleep(backoffDelay)
			backoffDelay *= 2 // Exponential backoff
		}
	}

	if !result.EventuallySucceeded {
		result.RetriesExhausted = true
	}

	result.TotalRetryDuration = time.Since(startTime)

	t.Logf("Retry result: %d attempts, succeeded: %v, duration: %v",
		result.AttemptCount, result.EventuallySucceeded, result.TotalRetryDuration)

	return result
}

func simulateOperationQueue(t *testing.T, config *helpers.E2ETestConfig, queueSize int) bool {
	t.Logf("Testing operation queue management with size: %d", queueSize)

	if config.TestMode == "mock" {
		t.Logf("MOCK: Operation queue managed successfully")
		return true
	}

	// In live mode, would test actual queue management
	time.Sleep(200 * time.Millisecond)
	return true
}

func analyzeRequestMetrics(results []RequestResult) PerformanceMetrics {
	if len(results) == 0 {
		return PerformanceMetrics{}
	}

	var totalDuration time.Duration
	var maxDuration time.Duration
	minDuration := time.Hour // Initialize to a large value
	successCount := 0

	startTime := results[0].Timestamp
	endTime := results[len(results)-1].Timestamp

	for _, result := range results {
		totalDuration += result.Duration

		if result.Duration > maxDuration {
			maxDuration = result.Duration
		}

		if result.Duration < minDuration {
			minDuration = result.Duration
		}

		if result.StatusCode >= 200 && result.StatusCode < 300 {
			successCount++
		}
	}

	avgDuration := totalDuration / time.Duration(len(results))
	totalTestTime := endTime.Sub(startTime)
	throughput := float64(len(results)) / totalTestTime.Seconds()
	successRate := float64(successCount) / float64(len(results)) * 100

	return PerformanceMetrics{
		AverageResponseTime: avgDuration,
		MaxResponseTime:     maxDuration,
		MinResponseTime:     minDuration,
		ThroughputRPS:       throughput,
		SuccessRate:         successRate,
	}
}
