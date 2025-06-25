package services

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
	"net"
	"strings"
	"time"
)

// SSHService handles SSH connections to VPS instances
type SSHService struct {
	timeout time.Duration
}

// NewSSHService creates a new SSH service instance
func NewSSHService() *SSHService {
	return &SSHService{
		timeout: 30 * time.Second,
	}
}

// SSHConnection represents an active SSH connection
type SSHConnection struct {
	client  *ssh.Client
	session *ssh.Session
}

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

// ConnectToVPS establishes an SSH connection to a VPS using private key authentication
func (ss *SSHService) ConnectToVPS(host, user, privateKeyPEM string) (*SSHConnection, error) {
	// Parse the private key
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing the private key")
	}

	// Parse the private key
	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create SSH signer
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH signer: %w", err)
	}

	// SSH client configuration
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For now, we'll accept any host key
		Timeout:         ss.timeout,
	}

	// Connect to the SSH server
	address := net.JoinHostPort(host, "22")
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	return &SSHConnection{
		client: client,
	}, nil
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
		case "INSTALLING_ARGOCD":
			status.SetupMessage = "Installing ArgoCD for GitOps..."
		case "WAITING_ARGOCD":
			status.SetupMessage = "Waiting for ArgoCD to be ready..."
		case "INSTALLING_ARGOCD_CLI":
			status.SetupMessage = "Installing ArgoCD CLI..."
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

	// Get memory usage
	if result, err := ss.ExecuteCommand(conn, "free -h"); err == nil {
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

// Close closes the SSH connection
func (conn *SSHConnection) Close() error {
	if conn.session != nil {
		conn.session.Close()
	}
	return conn.client.Close()
}
