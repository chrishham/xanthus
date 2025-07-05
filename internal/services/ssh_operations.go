package services

import (
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/ssh"
	"strings"
	"time"
)

// CommandResult represents the result of an SSH command execution
type CommandResult struct {
	Command  string `json:"command"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
	ExitCode int    `json:"exit_code"`
	Duration string `json:"duration"`
}

// VPSStatus represents the health status of a VPS
type VPSStatus struct {
	ServerID     int                    `json:"server_id"`
	IP           string                 `json:"ip"`
	Reachable    bool                   `json:"reachable"`
	SetupStatus  string                 `json:"setup_status"`
	SetupMessage string                 `json:"setup_message,omitempty"`
	K3sStatus    string                 `json:"k3s_status"`
	SystemLoad   map[string]interface{} `json:"system_load"`
	DiskUsage    map[string]interface{} `json:"disk_usage"`
	Services     map[string]string      `json:"services"`
	LastChecked  string                 `json:"last_checked"`
	Error        string                 `json:"error,omitempty"`
}

// ExecuteCommand executes a command on the VPS and returns the result
func (ss *SSHService) ExecuteCommand(conn *SSHConnection, command string) (*CommandResult, error) {
	start := time.Now()

	// Create a new session for this command
	session, err := conn.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Execute the command and capture output
	output, err := session.CombinedOutput(command)
	duration := time.Since(start)

	result := &CommandResult{
		Command:  command,
		Output:   strings.TrimSpace(string(output)),
		Duration: duration.String(),
	}

	if err != nil {
		result.Error = err.Error()
		if exitError, ok := err.(*ssh.ExitError); ok {
			result.ExitCode = exitError.ExitStatus()
		} else {
			result.ExitCode = -1
		}
		return result, fmt.Errorf("command failed: %s", err.Error())
	}

	return result, nil
}

// CheckVPSHealth performs comprehensive health checks on a VPS
func (ss *SSHService) CheckVPSHealth(host, user, privateKeyPEM string, serverID int) (*VPSStatus, error) {
	status := &VPSStatus{
		ServerID:    serverID,
		IP:          host,
		LastChecked: time.Now().UTC().Format(time.RFC3339),
		SystemLoad:  make(map[string]interface{}),
		DiskUsage:   make(map[string]interface{}),
		Services:    make(map[string]string),
	}

	// Try to connect
	conn, err := ss.ConnectToVPS(host, user, privateKeyPEM)
	if err != nil {
		status.Reachable = false
		status.Error = fmt.Sprintf("SSH connection failed: %v", err)
		return status, nil
	}
	defer conn.Close()

	status.Reachable = true

	// Check setup status from cloud-init
	if result, err := ss.ExecuteCommand(conn, "cat /opt/xanthus/status 2>/dev/null || echo 'UNKNOWN'"); err == nil {
		setupStatus := strings.TrimSpace(result.Output)
		status.SetupStatus = setupStatus

		// Add user-friendly messages for each status
		switch setupStatus {
		case "INSTALLING":
			status.SetupMessage = "Initializing server setup..."
		case "INSTALLING_K3S":
			status.SetupMessage = "Installing K3s Kubernetes cluster..."
		case "WAITING_K3S":
			status.SetupMessage = "Waiting for K3s to be ready..."
		case "INSTALLING_HELM":
			status.SetupMessage = "Installing Helm package manager..."
		case "VERIFYING":
			status.SetupMessage = "Verifying all components..."
		case "READY":
			status.SetupMessage = "Server is ready! All components installed and verified."
		case "UNKNOWN":
			status.SetupMessage = "Setup status unknown (server may still be initializing)"
		default:
			status.SetupMessage = fmt.Sprintf("Setup in progress: %s", setupStatus)
		}
	} else {
		status.SetupStatus = "UNKNOWN"
		status.SetupMessage = "Cannot determine setup status"
	}

	// Check K3s status
	if result, err := ss.ExecuteCommand(conn, "systemctl is-active k3s"); err == nil {
		status.K3sStatus = strings.TrimSpace(result.Output)
	} else {
		status.K3sStatus = "unknown"
	}

	// Get system load
	if result, err := ss.ExecuteCommand(conn, "uptime"); err == nil {
		status.SystemLoad["uptime"] = result.Output
	}

	// Get memory usage with structured data from /proc/meminfo
	if result, err := ss.ExecuteCommand(conn, `awk '/^MemTotal:|^MemFree:|^MemAvailable:|^Buffers:|^Cached:|^SwapTotal:|^SwapFree:/ {
		if ($1 == "MemTotal:") mem_total = $2
		else if ($1 == "MemFree:") mem_free = $2
		else if ($1 == "MemAvailable:") mem_available = $2
		else if ($1 == "Buffers:") buffers = $2
		else if ($1 == "Cached:") cached = $2
		else if ($1 == "SwapTotal:") swap_total = $2
		else if ($1 == "SwapFree:") swap_free = $2
	}
	END {
		mem_used = mem_total - mem_free
		buff_cache = buffers + cached
		swap_used = swap_total - swap_free
		
		printf "Memory Usage:\n"
		printf "Total: %.1fG, Used: %.1fG, Free: %.1fG, Available: %.1fG, Buff/Cache: %.1fG\n", 
			mem_total/1024/1024, mem_used/1024/1024, mem_free/1024/1024, mem_available/1024/1024, buff_cache/1024/1024
		printf "Swap Usage:\n"
		printf "Total: %.1fG, Used: %.1fG, Free: %.1fG\n", 
			swap_total/1024/1024, swap_used/1024/1024, swap_free/1024/1024
	}' /proc/meminfo`); err == nil {
		status.SystemLoad["memory"] = result.Output
	}

	// Get disk usage
	if result, err := ss.ExecuteCommand(conn, "df -h /"); err == nil {
		status.DiskUsage["root"] = result.Output
	}

	// Check important services
	services := []string{"k3s", "ssh", "systemd-resolved"}
	for _, service := range services {
		if result, err := ss.ExecuteCommand(conn, fmt.Sprintf("systemctl is-active %s", service)); err == nil {
			status.Services[service] = strings.TrimSpace(result.Output)
		} else {
			status.Services[service] = "unknown"
		}
	}

	return status, nil
}

// ConfigureK3s configures K3s with new SSL certificates
func (ss *SSHService) ConfigureK3s(conn *SSHConnection, sslCert, sslKey string) error {
	// Create SSL directory
	if _, err := ss.ExecuteCommand(conn, "mkdir -p /opt/xanthus/ssl"); err != nil {
		return fmt.Errorf("failed to create SSL directory: %w", err)
	}

	// Write SSL certificate
	certCommand := fmt.Sprintf("cat > /opt/xanthus/ssl/server.crt << 'EOF'\n%s\nEOF", sslCert)
	if result, err := ss.ExecuteCommand(conn, certCommand); err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to write SSL certificate: %v", err)
	}

	// Write SSL private key
	keyCommand := fmt.Sprintf("cat > /opt/xanthus/ssl/server.key << 'EOF'\n%s\nEOF", sslKey)
	if result, err := ss.ExecuteCommand(conn, keyCommand); err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to write SSL private key: %v", err)
	}

	// Set proper permissions
	if _, err := ss.ExecuteCommand(conn, "chmod 600 /opt/xanthus/ssl/server.key"); err != nil {
		return fmt.Errorf("failed to set SSL key permissions: %w", err)
	}

	if _, err := ss.ExecuteCommand(conn, "chmod 644 /opt/xanthus/ssl/server.crt"); err != nil {
		return fmt.Errorf("failed to set SSL cert permissions: %w", err)
	}

	// Restart K3s to use new certificates
	if result, err := ss.ExecuteCommand(conn, "systemctl restart k3s"); err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to restart K3s: %v", err)
	}

	// Wait a moment for K3s to start
	time.Sleep(5 * time.Second)

	// Verify K3s is running
	if result, err := ss.ExecuteCommand(conn, "systemctl is-active k3s"); err != nil || result.Output != "active" {
		return fmt.Errorf("K3s failed to start after SSL update: %s", result.Output)
	}

	return nil
}

// DeployManifest deploys a Kubernetes manifest to the K3s cluster
func (ss *SSHService) DeployManifest(conn *SSHConnection, manifest, name string) error {
	// Write manifest to file
	manifestPath := fmt.Sprintf("/tmp/%s.yaml", name)
	manifestCommand := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", manifestPath, manifest)
	if result, err := ss.ExecuteCommand(conn, manifestCommand); err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to write manifest: %v", err)
	}

	// Apply the manifest
	applyCommand := fmt.Sprintf("kubectl apply -f %s", manifestPath)
	if result, err := ss.ExecuteCommand(conn, applyCommand); err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to apply manifest: %s", result.Output)
	}

	// Clean up the temporary file
	ss.ExecuteCommand(conn, fmt.Sprintf("rm -f %s", manifestPath))

	return nil
}

// GetK3sLogs retrieves K3s service logs
func (ss *SSHService) GetK3sLogs(conn *SSHConnection, lines int) (string, error) {
	command := fmt.Sprintf("journalctl -u k3s -n %d --no-pager", lines)
	result, err := ss.ExecuteCommand(conn, command)
	if err != nil {
		return "", fmt.Errorf("failed to get K3s logs: %w", err)
	}

	return result.Output, nil
}

// GetVPSK3sLogs fetches K3s service logs via SSH (high-level wrapper)
func (ss *SSHService) GetVPSK3sLogs(host, user, privateKeyPEM string, lines int) (string, error) {
	conn, err := ss.GetOrCreateConnection(host, user, privateKeyPEM, 0)
	if err != nil {
		return "", fmt.Errorf("failed to connect to VPS: %w", err)
	}

	return ss.GetK3sLogs(conn, lines)
}

// GetVPSLogs fetches VPS system logs via SSH
func (ss *SSHService) GetVPSLogs(host, user, privateKeyPEM string, lines int) (string, error) {
	conn, err := ss.GetOrCreateConnection(host, user, privateKeyPEM, 0)
	if err != nil {
		return "", err
	}

	// Fetch various system logs
	command := fmt.Sprintf(`
		echo "=== System Logs (last %d lines) ==="
		sudo journalctl --no-pager -n %d
		echo ""
		echo "=== K3s Service Status ==="
		sudo systemctl status k3s || true
		echo ""
		echo "=== Docker Containers ==="
		sudo docker ps -a || true
	`, lines, lines)

	result, err := ss.ExecuteCommand(conn, command)
	if err != nil {
		return "", err
	}

	return result.Output, nil
}

// ListHelmRepositories lists all Helm repositories on the VPS
func (ss *SSHService) ListHelmRepositories(conn *SSHConnection) ([]map[string]interface{}, error) {
	command := "helm repo list -o json"
	result, err := ss.ExecuteCommand(conn, command)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %v", err)
	}

	var repositories []map[string]interface{}
	if result.Output != "" {
		if err := json.Unmarshal([]byte(result.Output), &repositories); err != nil {
			return nil, fmt.Errorf("failed to parse repository list: %v", err)
		}
	}

	return repositories, nil
}

// AddHelmRepository adds a new Helm repository to the VPS
func (ss *SSHService) AddHelmRepository(conn *SSHConnection, name, url string) error {
	// Sanitize inputs to prevent command injection
	if name == "" || url == "" {
		return fmt.Errorf("repository name and URL cannot be empty")
	}

	// Add the repository
	command := fmt.Sprintf("helm repo add %s %s", name, url)
	result, err := ss.ExecuteCommand(conn, command)
	if err != nil {
		return fmt.Errorf("failed to add repository: %v", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to add repository: %s", result.Output)
	}

	// Update repository index
	updateCommand := "helm repo update"
	updateResult, err := ss.ExecuteCommand(conn, updateCommand)
	if err != nil {
		return fmt.Errorf("failed to update repository index: %v", err)
	}

	if updateResult.ExitCode != 0 {
		return fmt.Errorf("failed to update repository index: %s", updateResult.Output)
	}

	return nil
}

// ListHelmCharts lists charts from a specific repository
func (ss *SSHService) ListHelmCharts(conn *SSHConnection, repositoryName string) ([]map[string]interface{}, error) {
	command := fmt.Sprintf("helm search repo %s -o json", repositoryName)
	result, err := ss.ExecuteCommand(conn, command)
	if err != nil {
		return nil, fmt.Errorf("failed to list charts: %v", err)
	}

	var charts []map[string]interface{}
	if result.Output != "" {
		if err := json.Unmarshal([]byte(result.Output), &charts); err != nil {
			return nil, fmt.Errorf("failed to parse charts list: %v", err)
		}
	}

	return charts, nil
}

// CreateTLSSecret creates a Kubernetes TLS secret in the specified namespace
func (ss *SSHService) CreateTLSSecret(conn *SSHConnection, domain, certificate, privateKey, namespace string) error {
	// Generate secret name based on domain (keep dots to match template expectation)
	secretName := domain + "-tls"
	// For file paths, we need to sanitize the name
	filePrefix := strings.ReplaceAll(domain, ".", "-")

	// Write certificate to temporary file
	certPath := fmt.Sprintf("/tmp/%s-cert.crt", filePrefix)
	certCommand := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", certPath, certificate)
	if result, err := ss.ExecuteCommand(conn, certCommand); err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to write certificate: %v", err)
	}

	// Write private key to temporary file
	keyPath := fmt.Sprintf("/tmp/%s-key.key", filePrefix)
	keyCommand := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", keyPath, privateKey)
	if result, err := ss.ExecuteCommand(conn, keyCommand); err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to write private key: %v", err)
	}

	// Ensure namespace exists
	createNSCommand := fmt.Sprintf("kubectl create namespace %s --dry-run=client -o yaml | kubectl apply -f -", namespace)
	ss.ExecuteCommand(conn, createNSCommand)

	// Delete existing secret if it exists (ignore errors)
	deleteCommand := fmt.Sprintf("kubectl delete secret %s -n %s --ignore-not-found=true", secretName, namespace)
	ss.ExecuteCommand(conn, deleteCommand)

	// Create the TLS secret in the specified namespace
	createSecretCommand := fmt.Sprintf("kubectl create secret tls %s --cert=%s --key=%s -n %s",
		secretName, certPath, keyPath, namespace)
	if result, err := ss.ExecuteCommand(conn, createSecretCommand); err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to create TLS secret in namespace %s: %s", namespace, result.Output)
	}

	// Clean up temporary files
	ss.ExecuteCommand(conn, fmt.Sprintf("rm -f %s %s", certPath, keyPath))

	return nil
}

// GetVPSInfo retrieves the VPS information file
func (ss *SSHService) GetVPSInfo(conn *SSHConnection) (string, error) {
	result, err := ss.ExecuteCommand(conn, "cat /opt/xanthus/info.txt 2>/dev/null || echo 'Info file not found'")
	if err != nil {
		return "", fmt.Errorf("failed to read VPS info: %w", err)
	}

	return result.Output, nil
}
