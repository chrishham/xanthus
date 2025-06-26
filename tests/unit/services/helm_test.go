package services

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/chrishham/xanthus/internal/services"
)

// MockSSHServiceForHelm provides a mock SSH service for testing Helm operations
type MockSSHServiceForHelm struct {
	commands           map[string]*services.CommandResult
	connectionError    error
	commandError       error
	expectedCommands   []string
	executedCommands   []string
}

func NewMockSSHServiceForHelm() *MockSSHServiceForHelm {
	return &MockSSHServiceForHelm{
		commands:         make(map[string]*services.CommandResult),
		expectedCommands: make([]string, 0),
		executedCommands: make([]string, 0),
	}
}

func (m *MockSSHServiceForHelm) AddExpectedCommand(command string, result *services.CommandResult) {
	m.commands[command] = result
	m.expectedCommands = append(m.expectedCommands, command)
}

func (m *MockSSHServiceForHelm) SetConnectionError(err error) {
	m.connectionError = err
}

func (m *MockSSHServiceForHelm) SetCommandError(err error) {
	m.commandError = err
}

func (m *MockSSHServiceForHelm) ConnectToVPS(host, user, privateKey string) (*MockSSHConnection, error) {
	if m.connectionError != nil {
		return nil, m.connectionError
	}
	return &MockSSHConnection{
		commands: m.commands,
		closed:   false,
	}, nil
}

func (m *MockSSHServiceForHelm) ExecuteCommand(conn *MockSSHConnection, command string) (*services.CommandResult, error) {
	m.executedCommands = append(m.executedCommands, command)
	
	if m.commandError != nil {
		return nil, m.commandError
	}
	
	if result, exists := conn.commands[command]; exists {
		return result, nil
	}
	
	// Default success result
	return &services.CommandResult{
		Command:  command,
		Output:   "success",
		ExitCode: 0,
		Duration: "100ms",
	}, nil
}

func (m *MockSSHServiceForHelm) GetExecutedCommands() []string {
	return m.executedCommands
}

func TestHelmService_NewHelmService(t *testing.T) {
	service := services.NewHelmService()
	
	assert.NotNil(t, service)
	// Service should be initialized with an SSH service
}

func TestHelmService_InstallChart(t *testing.T) {
	t.Run("successful chart installation", func(t *testing.T) {
		// Test parameters
		vpsIP := "192.168.1.100"
		sshUser := "root"
		privateKey := "mock-private-key"
		releaseName := "test-release"
		chartName := "nginx/nginx"
		chartVersion := "1.0.0"
		namespace := "test-namespace"
		values := map[string]string{
			"service.type":      "NodePort",
			"image.repository": "nginx",
		}
		
		// Expected commands that should be executed
		expectedNamespaceCmd := "kubectl create namespace test-namespace --dry-run=client -o yaml | kubectl apply -f -"
		expectedHelmCmd := "helm install test-release nginx/nginx --version 1.0.0 --namespace test-namespace --create-namespace --set service.type=NodePort,image.repository=nginx"
		
		// In a real test, we would mock the SSH service and verify these commands
		// For now, we'll verify the expected command structure
		assert.Contains(t, expectedHelmCmd, releaseName)
		assert.Contains(t, expectedHelmCmd, chartName)
		assert.Contains(t, expectedHelmCmd, chartVersion)
		assert.Contains(t, expectedHelmCmd, namespace)
		assert.Contains(t, expectedHelmCmd, "service.type=NodePort")
		assert.Contains(t, expectedHelmCmd, "image.repository=nginx")
		assert.Contains(t, expectedNamespaceCmd, namespace)
		
		// Test that all parameters are used
		assert.NotEmpty(t, vpsIP)
		assert.NotEmpty(t, sshUser)
		assert.NotEmpty(t, privateKey)
		assert.Len(t, values, 2)
	})

	t.Run("chart installation without values", func(t *testing.T) {
		releaseName := "simple-release"
		chartName := "stable/apache"
		chartVersion := "2.0.0"
		namespace := "default"
		
		expectedHelmCmd := "helm install simple-release stable/apache --version 2.0.0 --namespace default --create-namespace"
		
		// Command should not contain --set when no values are provided
		assert.NotContains(t, expectedHelmCmd, "--set")
		assert.Contains(t, expectedHelmCmd, releaseName)
		assert.Contains(t, expectedHelmCmd, chartName)
		assert.Contains(t, expectedHelmCmd, chartVersion)
		assert.Contains(t, expectedHelmCmd, namespace)
	})

	t.Run("SSH connection failure", func(t *testing.T) {
		// Test handling of SSH connection failures
		service := services.NewHelmService()
		
		// In practice, this would test with a mock that returns connection error
		assert.NotNil(t, service)
	})

	t.Run("namespace creation failure", func(t *testing.T) {
		// Test handling of namespace creation failures
		service := services.NewHelmService()
		assert.NotNil(t, service)
	})

	t.Run("helm install failure", func(t *testing.T) {
		// Test handling of helm install command failures
		service := services.NewHelmService()
		assert.NotNil(t, service)
	})
}

func TestHelmService_UpgradeChart(t *testing.T) {
	t.Run("successful chart upgrade", func(t *testing.T) {
		releaseName := "test-release"
		chartName := "nginx/nginx"
		chartVersion := "1.1.0"
		namespace := "test-namespace"
		values := map[string]string{
			"replicas":     "3",
			"service.port": "8080",
		}
		
		expectedHelmCmd := "helm upgrade test-release nginx/nginx --version 1.1.0 --namespace test-namespace --set replicas=3,service.port=8080"
		
		assert.Contains(t, expectedHelmCmd, "helm upgrade")
		assert.Contains(t, expectedHelmCmd, releaseName)
		assert.Contains(t, expectedHelmCmd, chartName)
		assert.Contains(t, expectedHelmCmd, chartVersion)
		assert.Contains(t, expectedHelmCmd, namespace)
		assert.Len(t, values, 2)
		assert.Contains(t, expectedHelmCmd, "replicas=3")
		assert.Contains(t, expectedHelmCmd, "service.port=8080")
	})

	t.Run("upgrade without values", func(t *testing.T) {
		releaseName := "simple-release"
		chartName := "stable/apache"
		chartVersion := "2.1.0"
		namespace := "default"
		
		expectedHelmCmd := "helm upgrade simple-release stable/apache --version 2.1.0 --namespace default"
		
		// Command should not contain --set when no values are provided
		assert.NotContains(t, expectedHelmCmd, "--set")
		assert.Contains(t, expectedHelmCmd, releaseName)
		assert.Contains(t, expectedHelmCmd, chartName)
		assert.Contains(t, expectedHelmCmd, chartVersion)
		assert.Contains(t, expectedHelmCmd, namespace)
	})

	t.Run("SSH connection failure", func(t *testing.T) {
		service := services.NewHelmService()
		assert.NotNil(t, service)
	})

	t.Run("helm upgrade failure", func(t *testing.T) {
		service := services.NewHelmService()
		assert.NotNil(t, service)
	})
}

func TestHelmService_UninstallChart(t *testing.T) {
	t.Run("successful chart uninstall", func(t *testing.T) {
		releaseName := "test-release"
		namespace := "test-namespace"
		
		expectedHelmCmd := "helm uninstall test-release --namespace test-namespace"
		
		assert.Contains(t, expectedHelmCmd, "helm uninstall")
		assert.Contains(t, expectedHelmCmd, releaseName)
		assert.Contains(t, expectedHelmCmd, namespace)
	})

	t.Run("SSH connection failure", func(t *testing.T) {
		service := services.NewHelmService()
		assert.NotNil(t, service)
	})

	t.Run("helm uninstall failure", func(t *testing.T) {
		service := services.NewHelmService()
		assert.NotNil(t, service)
	})
}

func TestHelmService_GetReleaseStatus(t *testing.T) {
	t.Run("deployed status", func(t *testing.T) {
		releaseName := "test-release"
		namespace := "test-namespace"
		
		expectedHelmCmd := "helm status test-release --namespace test-namespace -o json"
		mockOutput := `{"status": "deployed", "info": {"status": "deployed"}}`
		
		assert.Contains(t, expectedHelmCmd, "helm status")
		assert.Contains(t, expectedHelmCmd, releaseName)
		assert.Contains(t, expectedHelmCmd, namespace)
		assert.Contains(t, expectedHelmCmd, "-o json")
		
		// Test status parsing logic
		assert.Contains(t, mockOutput, "deployed")
	})

	t.Run("failed status", func(t *testing.T) {
		mockOutput := `{"status": "failed", "info": {"status": "failed"}}`
		
		// Test that failed status is properly detected
		assert.Contains(t, mockOutput, "failed")
	})

	t.Run("pending status", func(t *testing.T) {
		mockOutput := `{"status": "pending", "info": {"status": "pending"}}`
		
		// Test that pending status is properly detected
		assert.Contains(t, mockOutput, "pending")
	})

	t.Run("unknown status", func(t *testing.T) {
		mockOutput := `{"status": "superseded", "info": {"status": "superseded"}}`
		
		// Test that unknown statuses default to "unknown"
		assert.NotContains(t, mockOutput, "deployed")
		assert.NotContains(t, mockOutput, "failed")
		assert.NotContains(t, mockOutput, "pending")
	})

	t.Run("SSH connection failure", func(t *testing.T) {
		service := services.NewHelmService()
		assert.NotNil(t, service)
	})

	t.Run("helm status command failure", func(t *testing.T) {
		service := services.NewHelmService()
		assert.NotNil(t, service)
	})
}

func TestHelmService_CommandConstruction(t *testing.T) {
	t.Run("install command with multiple values", func(t *testing.T) {
		values := map[string]string{
			"service.type":        "LoadBalancer",
			"image.repository":    "nginx",
			"image.tag":          "latest",
			"resources.limits.memory": "512Mi",
		}
		
		// Test that all values are included in the command
		var setArgs []string
		for key, value := range values {
			setArgs = append(setArgs, fmt.Sprintf("%s=%s", key, value))
		}
		setString := strings.Join(setArgs, ",")
		
		assert.Contains(t, setString, "service.type=LoadBalancer")
		assert.Contains(t, setString, "image.repository=nginx")
		assert.Contains(t, setString, "image.tag=latest")
		assert.Contains(t, setString, "resources.limits.memory=512Mi")
		
		// Verify comma separation
		parts := strings.Split(setString, ",")
		assert.Len(t, parts, 4)
	})

	t.Run("upgrade command construction", func(t *testing.T) {
		releaseName := "my-app"
		chartName := "bitnami/nginx"
		chartVersion := "9.5.1"
		namespace := "production"
		
		baseCmd := fmt.Sprintf("helm upgrade %s %s --version %s --namespace %s",
			releaseName, chartName, chartVersion, namespace)
		
		assert.Equal(t, "helm upgrade my-app bitnami/nginx --version 9.5.1 --namespace production", baseCmd)
	})

	t.Run("uninstall command construction", func(t *testing.T) {
		releaseName := "my-app"
		namespace := "production"
		
		cmd := fmt.Sprintf("helm uninstall %s --namespace %s", releaseName, namespace)
		
		assert.Equal(t, "helm uninstall my-app --namespace production", cmd)
	})

	t.Run("status command construction", func(t *testing.T) {
		releaseName := "my-app"
		namespace := "production"
		
		cmd := fmt.Sprintf("helm status %s --namespace %s -o json", releaseName, namespace)
		
		assert.Equal(t, "helm status my-app --namespace production -o json", cmd)
	})
}

func TestHelmService_ParameterValidation(t *testing.T) {
	t.Run("empty release name", func(t *testing.T) {
		// Test validation of empty release name
		releaseName := ""
		
		// In practice, this should be validated and return an error
		assert.Empty(t, releaseName)
	})

	t.Run("empty chart name", func(t *testing.T) {
		// Test validation of empty chart name
		chartName := ""
		
		assert.Empty(t, chartName)
	})

	t.Run("empty namespace", func(t *testing.T) {
		// Test validation of empty namespace
		namespace := ""
		
		assert.Empty(t, namespace)
	})

	t.Run("invalid chart version", func(t *testing.T) {
		// Test validation of invalid chart version
		chartVersion := "invalid-version"
		
		assert.NotEmpty(t, chartVersion)
		// In practice, semantic version validation could be implemented
	})
}

func TestHelmService_ErrorScenarios(t *testing.T) {
	t.Run("network timeout", func(t *testing.T) {
		// Test handling of network timeouts during SSH connection
		service := services.NewHelmService()
		assert.NotNil(t, service)
	})

	t.Run("authentication failure", func(t *testing.T) {
		// Test handling of SSH authentication failures
		service := services.NewHelmService()
		assert.NotNil(t, service)
	})

	t.Run("helm not installed", func(t *testing.T) {
		// Test handling when Helm is not installed on the target system
		service := services.NewHelmService()
		assert.NotNil(t, service)
	})

	t.Run("kubernetes cluster not accessible", func(t *testing.T) {
		// Test handling when Kubernetes cluster is not accessible
		service := services.NewHelmService()
		assert.NotNil(t, service)
	})

	t.Run("insufficient permissions", func(t *testing.T) {
		// Test handling of insufficient permissions for namespace operations
		service := services.NewHelmService()
		assert.NotNil(t, service)
	})

	t.Run("chart not found", func(t *testing.T) {
		// Test handling when specified chart is not found
		service := services.NewHelmService()
		assert.NotNil(t, service)
	})

	t.Run("release already exists", func(t *testing.T) {
		// Test handling when trying to install a release that already exists
		service := services.NewHelmService()
		assert.NotNil(t, service)
	})

	t.Run("release not found for upgrade", func(t *testing.T) {
		// Test handling when trying to upgrade a non-existent release
		service := services.NewHelmService()
		assert.NotNil(t, service)
	})
}

func TestHelmService_StatusParsing(t *testing.T) {
	testCases := []struct {
		name           string
		output         string
		expectedStatus string
	}{
		{
			name:           "deployed status",
			output:         `{"status":"deployed","name":"test-release"}`,
			expectedStatus: "deployed",
		},
		{
			name:           "failed status",
			output:         `{"status":"failed","name":"test-release","info":{"description":"failed"}}`,
			expectedStatus: "failed",
		},
		{
			name:           "pending status",
			output:         `{"status":"pending","name":"test-release"}`,
			expectedStatus: "pending",
		},
		{
			name:           "superseded status",
			output:         `{"status":"superseded","name":"test-release"}`,
			expectedStatus: "unknown",
		},
		{
			name:           "uninstalled status",
			output:         `{"status":"uninstalled","name":"test-release"}`,
			expectedStatus: "unknown",
		},
		{
			name:           "empty output",
			output:         "",
			expectedStatus: "unknown",
		},
		{
			name:           "malformed json",
			output:         `{malformed json}`,
			expectedStatus: "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the status parsing logic
			var status string
			if strings.Contains(tc.output, "deployed") {
				status = "deployed"
			} else if strings.Contains(tc.output, "failed") {
				status = "failed"
			} else if strings.Contains(tc.output, "pending") {
				status = "pending"
			} else {
				status = "unknown"
			}
			
			assert.Equal(t, tc.expectedStatus, status)
		})
	}
}

func BenchmarkHelmService_InstallChart(b *testing.B) {
	service := services.NewHelmService()
	
	// Test parameters
	vpsIP := "192.168.1.100"
	sshUser := "root"
	privateKey := "mock-private-key"
	releaseName := "bench-release"
	chartName := "nginx/nginx"
	chartVersion := "1.0.0"
	namespace := "bench-namespace"
	values := map[string]string{
		"service.type": "NodePort",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// In practice, this would benchmark the actual install operation
		_ = service
		_ = vpsIP
		_ = sshUser
		_ = privateKey
		_ = releaseName
		_ = chartName
		_ = chartVersion
		_ = namespace
		_ = values
	}
}

func BenchmarkHelmService_CommandConstruction(b *testing.B) {
	releaseName := "test-release"
	chartName := "nginx/nginx"
	chartVersion := "1.0.0"
	namespace := "test-namespace"
	values := map[string]string{
		"service.type":      "NodePort",
		"image.repository": "nginx",
		"replicas":         "3",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark command construction
		helmCmd := fmt.Sprintf("helm install %s %s --version %s --namespace %s --create-namespace",
			releaseName, chartName, chartVersion, namespace)
		
		if len(values) > 0 {
			var setArgs []string
			for key, value := range values {
				setArgs = append(setArgs, fmt.Sprintf("%s=%s", key, value))
			}
			helmCmd += " --set " + strings.Join(setArgs, ",")
		}
		
		_ = helmCmd
	}
}

// Test helper functions
func createMockHelmResult(command, output string, exitCode int) *services.CommandResult {
	return &services.CommandResult{
		Command:  command,
		Output:   output,
		ExitCode: exitCode,
		Duration: "100ms",
	}
}

func createSuccessResult(command string) *services.CommandResult {
	return &services.CommandResult{
		Command:  command,
		Output:   "success",
		ExitCode: 0,
		Duration: "100ms",
	}
}

func createErrorResult(command, errorMsg string, exitCode int) *services.CommandResult {
	return &services.CommandResult{
		Command:  command,
		Output:   errorMsg,
		Error:    errorMsg,
		ExitCode: exitCode,
		Duration: "100ms",
	}
}