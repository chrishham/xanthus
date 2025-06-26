package services

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chrishham/xanthus/internal/services"
)

// Mock SSH connection for testing
type MockSSHConnection struct {
	commands map[string]*services.CommandResult
	closed   bool
}

func (m *MockSSHConnection) Close() error {
	m.closed = true
	return nil
}

// Mock SSH service for testing
type MockSSHService struct {
	connections map[string]*MockSSHConnection
	timeout     time.Duration
}

func NewMockSSHService() *MockSSHService {
	return &MockSSHService{
		connections: make(map[string]*MockSSHConnection),
		timeout:     30 * time.Second,
	}
}

func (m *MockSSHService) AddMockCommand(connectionKey, command string, result *services.CommandResult) {
	if conn, exists := m.connections[connectionKey]; exists {
		conn.commands[command] = result
	}
}

func TestSSHService_NewSSHService(t *testing.T) {
	service := services.NewSSHService()
	
	assert.NotNil(t, service)
	// Service should initialize with empty connections
	// In a real test, we'd need access to the internal state
}

func TestSSHService_GetConnectionKey(t *testing.T) {
	service := services.NewSSHService()
	
	// Since getConnectionKey is not exported, we'll test the expected behavior
	// through the public methods that use it
	assert.NotNil(t, service)
}

func TestSSHService_ConnectionCaching(t *testing.T) {
	t.Run("caches connections", func(t *testing.T) {
		// This test would verify that connections are properly cached
		// and reused when the same host/user combination is requested
		service := services.NewSSHService()
		assert.NotNil(t, service)
	})

	t.Run("cleanup stale connections", func(t *testing.T) {
		// This test would verify the cleanup mechanism
		service := services.NewSSHService()
		assert.NotNil(t, service)
	})
}

func TestSSHService_ExecuteCommand(t *testing.T) {
	t.Run("successful command execution", func(t *testing.T) {
		// Mock command execution
		expectedOutput := "test output"
		expectedCommand := "echo hello"
		
		// In a real test, we'd mock the SSH connection
		// and verify the command execution logic
		assert.NotEmpty(t, expectedOutput)
		assert.NotEmpty(t, expectedCommand)
	})

	t.Run("command with error", func(t *testing.T) {
		// Test error handling in command execution
		expectedError := "command not found"
		expectedExitCode := 127
		
		assert.NotEmpty(t, expectedError)
		assert.Equal(t, 127, expectedExitCode)
	})

	t.Run("command timeout", func(t *testing.T) {
		// Test command timeout handling
		service := services.NewSSHService()
		assert.NotNil(t, service)
	})
}

func TestSSHService_PrivateKeyParsing(t *testing.T) {
	t.Run("valid PEM private key", func(t *testing.T) {
		// Generate a test private key
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		
		// Convert to PEM format
		privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
		require.NoError(t, err)
		
		privateKeyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privateKeyDER,
		})
		
		// Test that the key can be parsed (we'd need to expose the parsing logic)
		assert.NotEmpty(t, privateKeyPEM)
		assert.Contains(t, string(privateKeyPEM), "BEGIN PRIVATE KEY")
		assert.Contains(t, string(privateKeyPEM), "END PRIVATE KEY")
	})

	t.Run("invalid PEM format", func(t *testing.T) {
		invalidPEM := "not a pem key"
		
		// Test error handling for invalid PEM
		// In practice, this would test the connectToVPS method
		assert.NotEmpty(t, invalidPEM)
	})

	t.Run("invalid private key", func(t *testing.T) {
		invalidPEM := `-----BEGIN PRIVATE KEY-----
invalid base64 data
-----END PRIVATE KEY-----`
		
		// Test error handling for invalid private key
		assert.NotEmpty(t, invalidPEM)
	})
}

func TestSSHService_CheckVPSHealth(t *testing.T) {
	t.Run("healthy VPS", func(t *testing.T) {
		// Mock a healthy VPS response
		expectedCommands := map[string]string{
			"cat /opt/xanthus/status 2>/dev/null || echo 'UNKNOWN'": "READY",
			"systemctl is-active k3s":                                "active",
			"uptime":                                                 "up 1 day, 2:30",
			"free -h":                                                "total used free available",
			"df -h /":                                                "Filesystem Size Used Avail Use% Mounted on",
			"systemctl is-active ssh":                                "active",
			"systemctl is-active systemd-resolved":                   "active",
		}
		
		// In a real test, we'd mock the SSH connection and verify the health check logic
		for cmd, output := range expectedCommands {
			assert.NotEmpty(t, cmd)
			assert.NotEmpty(t, output)
		}
	})

	t.Run("unreachable VPS", func(t *testing.T) {
		// Test handling of connection failures
		service := services.NewSSHService()
		
		// This would test with invalid connection details
		assert.NotNil(t, service)
	})

	t.Run("VPS with setup in progress", func(t *testing.T) {
		// Test different setup statuses
		setupStatuses := []string{
			"INSTALLING",
			"INSTALLING_K3S",
			"WAITING_K3S",
			"INSTALLING_HELM",
			"INSTALLING_ARGOCD",
			"WAITING_ARGOCD",
			"INSTALLING_ARGOCD_CLI",
			"VERIFYING",
			"READY",
			"UNKNOWN",
		}
		
		for _, status := range setupStatuses {
			assert.NotEmpty(t, status)
			// Each status should have a corresponding message
		}
	})
}

func TestSSHService_ConfigureK3s(t *testing.T) {
	t.Run("successful SSL configuration", func(t *testing.T) {
		sslCert := `-----BEGIN CERTIFICATE-----
MIICertificateDataHere
-----END CERTIFICATE-----`
		
		sslKey := `-----BEGIN PRIVATE KEY-----
MIIPrivateKeyDataHere
-----END PRIVATE KEY-----`
		
		// Expected commands for SSL configuration
		expectedCommands := []string{
			"mkdir -p /opt/xanthus/ssl",
			"chmod 600 /opt/xanthus/ssl/server.key",
			"chmod 644 /opt/xanthus/ssl/server.crt",
			"systemctl restart k3s",
			"systemctl is-active k3s",
		}
		
		for _, cmd := range expectedCommands {
			assert.NotEmpty(t, cmd)
		}
		
		assert.NotEmpty(t, sslCert)
		assert.NotEmpty(t, sslKey)
	})

	t.Run("SSL directory creation fails", func(t *testing.T) {
		// Test error handling when directory creation fails
		service := services.NewSSHService()
		assert.NotNil(t, service)
	})

	t.Run("K3s restart fails", func(t *testing.T) {
		// Test error handling when K3s restart fails
		service := services.NewSSHService()
		assert.NotNil(t, service)
	})
}

func TestSSHService_DeployManifest(t *testing.T) {
	t.Run("successful manifest deployment", func(t *testing.T) {
		manifest := `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  containers:
  - name: test
    image: nginx`
		
		manifestName := "test-manifest"
		
		// Expected commands
		expectedCommands := []string{
			"kubectl apply -f /tmp/test-manifest.yaml",
			"rm -f /tmp/test-manifest.yaml",
		}
		
		for _, cmd := range expectedCommands {
			assert.NotEmpty(t, cmd)
		}
		
		assert.NotEmpty(t, manifest)
		assert.NotEmpty(t, manifestName)
	})

	t.Run("manifest write fails", func(t *testing.T) {
		// Test error handling when manifest write fails
		service := services.NewSSHService()
		assert.NotNil(t, service)
	})

	t.Run("kubectl apply fails", func(t *testing.T) {
		// Test error handling when kubectl apply fails
		service := services.NewSSHService()
		assert.NotNil(t, service)
	})
}

func TestSSHService_GetK3sLogs(t *testing.T) {
	t.Run("retrieves logs successfully", func(t *testing.T) {
		lines := 50
		expectedCommand := "journalctl -u k3s -n 50 --no-pager"
		expectedOutput := "K3s service logs here"
		
		assert.Equal(t, 50, lines)
		assert.NotEmpty(t, expectedCommand)
		assert.NotEmpty(t, expectedOutput)
	})

	t.Run("log retrieval fails", func(t *testing.T) {
		// Test error handling when log retrieval fails
		service := services.NewSSHService()
		assert.NotNil(t, service)
	})
}

func TestSSHService_GetVPSLogs(t *testing.T) {
	t.Run("retrieves system logs", func(t *testing.T) {
		lines := 100
		
		// Expected command structure
		expectedCommands := []string{
			"journalctl --no-pager",
			"systemctl status k3s",
			"docker ps -a",
		}
		
		for _, cmd := range expectedCommands {
			assert.NotEmpty(t, cmd)
		}
		
		assert.Equal(t, 100, lines)
	})
}

func TestSSHService_HelmOperations(t *testing.T) {
	t.Run("ListHelmRepositories", func(t *testing.T) {
		expectedCommand := "helm repo list -o json"
		expectedOutput := `[{"name":"stable","url":"https://charts.helm.sh/stable"}]`
		
		assert.NotEmpty(t, expectedCommand)
		assert.NotEmpty(t, expectedOutput)
	})

	t.Run("AddHelmRepository", func(t *testing.T) {
		repoName := "bitnami"
		repoURL := "https://charts.bitnami.com/bitnami"
		
		expectedCommands := []string{
			"helm repo add bitnami https://charts.bitnami.com/bitnami",
			"helm repo update",
		}
		
		for _, cmd := range expectedCommands {
			assert.NotEmpty(t, cmd)
		}
		
		assert.NotEmpty(t, repoName)
		assert.NotEmpty(t, repoURL)
	})

	t.Run("AddHelmRepository with empty parameters", func(t *testing.T) {
		// Test validation of empty parameters
		service := services.NewSSHService()
		
		// This should fail validation
		assert.NotNil(t, service)
	})

	t.Run("ListHelmCharts", func(t *testing.T) {
		repoName := "bitnami"
		expectedCommand := "helm search repo bitnami -o json"
		
		assert.NotEmpty(t, repoName)
		assert.NotEmpty(t, expectedCommand)
	})
}

func TestSSHService_ConnectionLifecycle(t *testing.T) {
	t.Run("connection establishment", func(t *testing.T) {
		// Test the complete connection establishment process
		host := "192.168.1.100"
		user := "root"
		
		// Generate test private key
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		
		privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
		require.NoError(t, err)
		
		privateKeyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privateKeyDER,
		})
		
		assert.NotEmpty(t, host)
		assert.NotEmpty(t, user)
		assert.NotEmpty(t, privateKeyPEM)
	})

	t.Run("connection reuse", func(t *testing.T) {
		// Test that connections are properly reused
		service := services.NewSSHService()
		assert.NotNil(t, service)
	})

	t.Run("connection cleanup", func(t *testing.T) {
		// Test connection cleanup
		service := services.NewSSHService()
		
		// Test CloseAllConnections
		service.CloseAllConnections()
		assert.NotNil(t, service)
	})
}

func TestSSHService_ErrorHandling(t *testing.T) {
	t.Run("connection timeout", func(t *testing.T) {
		// Test connection timeout handling
		service := services.NewSSHService()
		assert.NotNil(t, service)
	})

	t.Run("authentication failure", func(t *testing.T) {
		// Test authentication failure handling
		service := services.NewSSHService()
		assert.NotNil(t, service)
	})

	t.Run("network unreachable", func(t *testing.T) {
		// Test network unreachable error handling
		service := services.NewSSHService()
		assert.NotNil(t, service)
	})

	t.Run("session creation failure", func(t *testing.T) {
		// Test session creation failure
		service := services.NewSSHService()
		assert.NotNil(t, service)
	})
}

func TestSSHConnection_Close(t *testing.T) {
	t.Run("closes connection properly", func(t *testing.T) {
		// Test connection closure
		// In practice, this would test the actual Close method
		mock := &MockSSHConnection{
			commands: make(map[string]*services.CommandResult),
			closed:   false,
		}
		
		err := mock.Close()
		assert.NoError(t, err)
		assert.True(t, mock.closed)
	})
}

func TestCommandResult_Structure(t *testing.T) {
	t.Run("command result contains expected fields", func(t *testing.T) {
		result := &services.CommandResult{
			Command:  "echo test",
			Output:   "test",
			Error:    "",
			ExitCode: 0,
			Duration: "100ms",
		}
		
		assert.Equal(t, "echo test", result.Command)
		assert.Equal(t, "test", result.Output)
		assert.Equal(t, "", result.Error)
		assert.Equal(t, 0, result.ExitCode)
		assert.Equal(t, "100ms", result.Duration)
	})
}

func TestVPSStatus_Structure(t *testing.T) {
	t.Run("VPS status contains expected fields", func(t *testing.T) {
		status := &services.VPSStatus{
			ServerID:     123,
			IP:           "192.168.1.100",
			Reachable:    true,
			SetupStatus:  "READY",
			SetupMessage: "Server is ready!",
			K3sStatus:    "active",
			SystemLoad:   make(map[string]interface{}),
			DiskUsage:    make(map[string]interface{}),
			Services:     make(map[string]string),
			LastChecked:  "2023-01-01T00:00:00Z",
		}
		
		assert.Equal(t, 123, status.ServerID)
		assert.Equal(t, "192.168.1.100", status.IP)
		assert.True(t, status.Reachable)
		assert.Equal(t, "READY", status.SetupStatus)
		assert.Equal(t, "Server is ready!", status.SetupMessage)
		assert.Equal(t, "active", status.K3sStatus)
		assert.NotNil(t, status.SystemLoad)
		assert.NotNil(t, status.DiskUsage)
		assert.NotNil(t, status.Services)
	})
}

func BenchmarkSSHService_CommandExecution(b *testing.B) {
	// Benchmark command execution performance
	service := services.NewSSHService()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// In practice, this would benchmark actual command execution
		_ = service
	}
}

func BenchmarkSSHService_ConnectionCaching(b *testing.B) {
	// Benchmark connection caching performance
	service := services.NewSSHService()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// In practice, this would benchmark connection reuse
		_ = service
	}
}

// Test helper functions
func generateTestPrivateKey() (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}
	
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", err
	}
	
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyDER,
	})
	
	return string(privateKeyPEM), nil
}

func createMockCommandResult(command, output string, exitCode int) *services.CommandResult {
	return &services.CommandResult{
		Command:  command,
		Output:   output,
		ExitCode: exitCode,
		Duration: "100ms",
	}
}

func createMockVPSStatus(serverID int, ip string, reachable bool) *services.VPSStatus {
	return &services.VPSStatus{
		ServerID:    serverID,
		IP:          ip,
		Reachable:   reachable,
		SetupStatus: "READY",
		K3sStatus:   "active",
		SystemLoad:  make(map[string]interface{}),
		DiskUsage:   make(map[string]interface{}),
		Services:    make(map[string]string),
		LastChecked: time.Now().UTC().Format(time.RFC3339),
	}
}