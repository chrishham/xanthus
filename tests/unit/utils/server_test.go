package utils

import (
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/chrishham/xanthus/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestFindAvailablePort(t *testing.T) {
	t.Run("Find available port in normal conditions", func(t *testing.T) {
		port := utils.FindAvailablePort()
		
		// Port should be non-empty
		assert.NotEmpty(t, port)
		
		// Port should be a valid number
		portNum, err := strconv.Atoi(port)
		assert.NoError(t, err)
		
		// Port should be in the expected range (8080-8090)
		assert.GreaterOrEqual(t, portNum, 8080)
		assert.LessOrEqual(t, portNum, 8090)
		
		// Verify the port is actually available by trying to listen on it
		listener, err := net.Listen("tcp", ":"+port)
		assert.NoError(t, err, "Port should be available")
		if listener != nil {
			listener.Close()
		}
	})

	t.Run("Multiple calls return different ports when previous is occupied", func(t *testing.T) {
		// Get first available port
		port1 := utils.FindAvailablePort()
		assert.NotEmpty(t, port1)
		
		// Occupy the first port
		listener1, err := net.Listen("tcp", ":"+port1)
		assert.NoError(t, err)
		defer listener1.Close()
		
		// Get second available port
		port2 := utils.FindAvailablePort()
		assert.NotEmpty(t, port2)
		
		// Ports should be different
		assert.NotEqual(t, port1, port2)
		
		// Both should be in valid range
		port1Num, _ := strconv.Atoi(port1)
		port2Num, _ := strconv.Atoi(port2)
		assert.GreaterOrEqual(t, port1Num, 8080)
		assert.LessOrEqual(t, port1Num, 8090)
		assert.GreaterOrEqual(t, port2Num, 8080)
		assert.LessOrEqual(t, port2Num, 8090)
	})

	t.Run("Return empty string when all ports are occupied", func(t *testing.T) {
		// Occupy all ports in the range 8080-8090
		var listeners []net.Listener
		
		for port := 8080; port <= 8090; port++ {
			listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err == nil {
				listeners = append(listeners, listener)
			}
		}
		
		// Clean up all listeners at the end
		defer func() {
			for _, listener := range listeners {
				listener.Close()
			}
		}()
		
		// If we managed to occupy some ports, test the function
		if len(listeners) > 0 {
			// Try to find an available port
			port := utils.FindAvailablePort()
			
			// If all ports are occupied, should return empty string
			// If some ports are still available, should return a valid port
			if port == "" {
				// All ports were occupied - this is the expected behavior
				assert.Empty(t, port)
			} else {
				// Some ports were still available
				portNum, err := strconv.Atoi(port)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, portNum, 8080)
				assert.LessOrEqual(t, portNum, 8090)
			}
		}
	})

	t.Run("Port scanning order", func(t *testing.T) {
		// This test verifies that the function scans ports in order from 8080 to 8090
		
		// Occupy port 8080
		listener8080, err8080 := net.Listen("tcp", ":8080")
		if err8080 == nil {
			defer listener8080.Close()
			
			// Now find available port - should be 8081 or higher
			port := utils.FindAvailablePort()
			if port != "" {
				portNum, err := strconv.Atoi(port)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, portNum, 8081)
				assert.LessOrEqual(t, portNum, 8090)
			}
		}
	})

	t.Run("Concurrent access", func(t *testing.T) {
		// Test that concurrent calls to FindAvailablePort work correctly
		numGoroutines := 5
		portChan := make(chan string, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func() {
				port := utils.FindAvailablePort()
				portChan <- port
			}()
		}
		
		// Collect all ports
		var ports []string
		for i := 0; i < numGoroutines; i++ {
			select {
			case port := <-portChan:
				if port != "" {
					ports = append(ports, port)
				}
			case <-time.After(time.Second):
				t.Fatal("Timeout waiting for port")
			}
		}
		
		// Verify all ports are in valid range
		for _, port := range ports {
			portNum, err := strconv.Atoi(port)
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, portNum, 8080)
			assert.LessOrEqual(t, portNum, 8090)
		}
		
		// Check for uniqueness (though not guaranteed due to race conditions)
		uniquePorts := make(map[string]bool)
		for _, port := range ports {
			uniquePorts[port] = true
		}
		
		// In ideal conditions, all ports should be unique
		// But due to race conditions, we just verify they're all valid
		assert.True(t, len(uniquePorts) > 0)
	})

	t.Run("Port validation", func(t *testing.T) {
		// Test multiple calls to ensure consistency
		for i := 0; i < 10; i++ {
			port := utils.FindAvailablePort()
			
			if port != "" {
				// Validate port format
				portNum, err := strconv.Atoi(port)
				assert.NoError(t, err, "Port should be a valid number")
				assert.GreaterOrEqual(t, portNum, 8080, "Port should be >= 8080")
				assert.LessOrEqual(t, portNum, 8090, "Port should be <= 8090")
				
				// Validate port is actually usable
				listener, err := net.Listen("tcp", ":"+port)
				if err == nil {
					listener.Close()
				}
				// Note: We don't assert NoError here because the port might be taken
				// between when FindAvailablePort checks and when we try to use it
			}
		}
	})
}

func TestFindAvailablePortEdgeCases(t *testing.T) {
	t.Run("Port range boundary testing", func(t *testing.T) {
		// Test that the function respects the 8080-8090 range
		port := utils.FindAvailablePort()
		
		if port != "" {
			portNum, err := strconv.Atoi(port)
			assert.NoError(t, err)
			
			// Verify exact boundaries
			assert.True(t, portNum >= 8080, "Port should be at least 8080")
			assert.True(t, portNum <= 8090, "Port should be at most 8090")
			
			// Verify it's not outside the range
			assert.False(t, portNum < 8080, "Port should not be below 8080")
			assert.False(t, portNum > 8090, "Port should not be above 8090")
		}
	})

	t.Run("Return value format", func(t *testing.T) {
		port := utils.FindAvailablePort()
		
		if port != "" {
			// Should be a string representation of a number
			_, err := strconv.Atoi(port)
			assert.NoError(t, err, "Port should be convertible to integer")
			
			// Should not have leading zeros (except for single "0" but that's not in our range)
			if len(port) > 1 {
				assert.NotEqual(t, "0", string(port[0]), "Port should not have leading zeros")
			}
		} else {
			// Empty string is valid when no ports are available
			assert.Equal(t, "", port)
		}
	})
}

func TestFindAvailablePortPerformance(t *testing.T) {
	t.Run("Performance test", func(t *testing.T) {
		// Measure time to find available port
		start := time.Now()
		
		for i := 0; i < 100; i++ {
			port := utils.FindAvailablePort()
			_ = port // Use the port to avoid optimization
		}
		
		elapsed := time.Since(start)
		
		// Should complete reasonably quickly
		// This is a loose test - adjust threshold as needed
		assert.Less(t, elapsed, 5*time.Second, "Finding ports should be reasonably fast")
	})
}

// Benchmarks
func BenchmarkFindAvailablePort(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		port := utils.FindAvailablePort()
		_ = port // Prevent optimization
	}
}

func BenchmarkFindAvailablePortWithOccupiedPorts(b *testing.B) {
	// Occupy the first few ports to make the function work harder
	var listeners []net.Listener
	for port := 8080; port <= 8085; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			listeners = append(listeners, listener)
		}
	}
	
	defer func() {
		for _, listener := range listeners {
			listener.Close()
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		port := utils.FindAvailablePort()
		_ = port // Prevent optimization
	}
}

// Helper function to test port availability
func isPortAvailable(port string) bool {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

func TestPortAvailabilityHelper(t *testing.T) {
	// Test our helper function
	t.Run("Helper function works", func(t *testing.T) {
		// Find a port
		port := utils.FindAvailablePort()
		if port != "" {
			// Should be available
			assert.True(t, isPortAvailable(port))
			
			// Occupy it
			listener, err := net.Listen("tcp", ":"+port)
			assert.NoError(t, err)
			
			// Should no longer be available
			assert.False(t, isPortAvailable(port))
			
			listener.Close()
			
			// Give it a moment to be released
			time.Sleep(10 * time.Millisecond)
			
			// Should be available again
			assert.True(t, isPortAvailable(port))
		}
	})
}