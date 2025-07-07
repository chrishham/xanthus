package utils

import (
	"fmt"
	"net"
	"strconv"
)

// FindAvailablePort finds an available port in the range 8080-8110
func FindAvailablePort() string {
	for port := 8080; port <= 8110; port++ {
		address := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", address)
		if err == nil {
			listener.Close()
			return fmt.Sprintf("%d", port)
		}
	}
	return ""
}

// ParseServerID parses a server ID string to integer
// This extracts the common pattern used across VPS handlers
func ParseServerID(serverIDStr string) (int, error) {
	if serverIDStr == "" {
		return 0, fmt.Errorf("server ID is required")
	}

	serverID, err := strconv.Atoi(serverIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid server ID format: %w", err)
	}

	return serverID, nil
}
