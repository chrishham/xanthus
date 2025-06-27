package services

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// SSHService handles SSH connections to VPS instances
type SSHService struct {
	timeout     time.Duration
	connections map[string]*CachedSSHConnection
	mutex       sync.RWMutex
}

// CachedSSHConnection represents a cached SSH connection with metadata
type CachedSSHConnection struct {
	conn     *SSHConnection
	lastUsed time.Time
	serverID int
	host     string
	user     string
}

// SessionManager manages persistent SSH sessions for multi-step operations
type SessionManager struct {
	sessions map[string]*PersistentSSHSession
	mutex    sync.RWMutex
}

// PersistentSSHSession represents a session-bound SSH connection
type PersistentSSHSession struct {
	ID          string
	VPSServerID int
	Connection  *SSHConnection
	CreatedAt   time.Time
	LastUsed    time.Time
	UserToken   string
	Host        string
	User        string
}

var globalSessionManager *SessionManager

// NewSSHService creates a new SSH service instance
func NewSSHService() *SSHService {
	service := &SSHService{
		timeout:     30 * time.Second,
		connections: make(map[string]*CachedSSHConnection),
	}

	// Start cleanup goroutine for stale connections
	go service.cleanupStaleConnections()

	// Initialize global session manager if not already done
	if globalSessionManager == nil {
		globalSessionManager = NewSessionManager()
	}

	return service
}

// NewSessionManager creates a new session manager instance
func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*PersistentSSHSession),
	}

	// Start cleanup goroutine for expired sessions
	go sm.cleanupExpiredSessions()

	return sm
}

// GetGlobalSessionManager returns the global session manager instance
func GetGlobalSessionManager() *SessionManager {
	if globalSessionManager == nil {
		globalSessionManager = NewSessionManager()
	}
	return globalSessionManager
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

// getConnectionKey generates a unique key for caching connections
func (ss *SSHService) getConnectionKey(host, user string) string {
	return fmt.Sprintf("%s@%s", user, host)
}

// GetOrCreateConnection gets an existing connection or creates a new one
func (ss *SSHService) GetOrCreateConnection(host, user, privateKeyPEM string, serverID int) (*SSHConnection, error) {
	connectionKey := ss.getConnectionKey(host, user)

	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	// Check if we have a cached connection
	if cached, exists := ss.connections[connectionKey]; exists {
		// Test if the connection is still alive
		if ss.isConnectionAlive(cached.conn) {
			cached.lastUsed = time.Now()
			return cached.conn, nil
		} else {
			// Connection is dead, remove it
			delete(ss.connections, connectionKey)
		}
	}

	// Create new connection
	conn, err := ss.connectToVPS(host, user, privateKeyPEM)
	if err != nil {
		return nil, err
	}

	// Cache the connection
	ss.connections[connectionKey] = &CachedSSHConnection{
		conn:     conn,
		lastUsed: time.Now(),
		serverID: serverID,
		host:     host,
		user:     user,
	}

	return conn, nil
}

// isConnectionAlive tests if an SSH connection is still alive
func (ss *SSHService) isConnectionAlive(conn *SSHConnection) bool {
	if conn == nil || conn.client == nil {
		return false
	}

	// Try to create a session to test the connection
	session, err := conn.client.NewSession()
	if err != nil {
		return false
	}
	session.Close()
	return true
}

// cleanupStaleConnections removes connections that haven't been used recently
func (ss *SSHService) cleanupStaleConnections() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ss.mutex.Lock()
		now := time.Now()
		for key, cached := range ss.connections {
			// Remove connections older than 10 minutes
			if now.Sub(cached.lastUsed) > 10*time.Minute {
				if cached.conn != nil {
					cached.conn.Close()
				}
				delete(ss.connections, key)
			}
		}
		ss.mutex.Unlock()
	}
}

// CloseAllConnections closes all cached connections
func (ss *SSHService) CloseAllConnections() {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	for key, cached := range ss.connections {
		if cached.conn != nil {
			cached.conn.Close()
		}
		delete(ss.connections, key)
	}
}

// ConnectToVPS establishes an SSH connection to a VPS using private key authentication (deprecated, use GetOrCreateConnection)
func (ss *SSHService) ConnectToVPS(host, user, privateKeyPEM string) (*SSHConnection, error) {
	return ss.connectToVPS(host, user, privateKeyPEM)
}

// connectToVPS is the internal method that actually establishes SSH connections
func (ss *SSHService) connectToVPS(host, user, privateKeyPEM string) (*SSHConnection, error) {
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

// Session Management Methods

// CreateSession creates a new persistent SSH session
func (sm *SessionManager) CreateSession(vpsServerID int, host, user, privateKey, userToken string) (string, error) {
	// Generate unique session ID
	sessionID := fmt.Sprintf("ssh-%s-%d-%d", userToken[:min(8, len(userToken))], vpsServerID, time.Now().Unix())

	// Create SSH connection
	sshService := NewSSHService()
	conn, err := sshService.connectToVPS(host, user, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to create SSH connection: %v", err)
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Store session
	sm.sessions[sessionID] = &PersistentSSHSession{
		ID:          sessionID,
		VPSServerID: vpsServerID,
		Connection:  conn,
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		UserToken:   userToken,
		Host:        host,
		User:        user,
	}

	log.Printf("Created SSH session %s for VPS %d", sessionID, vpsServerID)
	return sessionID, nil
}

// GetSession retrieves an existing session by ID
func (sm *SessionManager) GetSession(sessionID string) (*PersistentSSHSession, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, false
	}

	// Check if session is still valid
	if !session.isValid() {
		// Session is invalid, remove it
		go sm.removeSession(sessionID)
		return nil, false
	}

	// Update last used time
	session.LastUsed = time.Now()
	return session, true
}

// GetSessionConnection returns the SSH connection for a session
func (sm *SessionManager) GetSessionConnection(sessionID string) (*SSHConnection, bool) {
	session, exists := sm.GetSession(sessionID)
	if !exists {
		return nil, false
	}
	return session.Connection, true
}

// RemoveSession removes a session by ID
func (sm *SessionManager) RemoveSession(sessionID string) {
	sm.removeSession(sessionID)
}

// removeSession is the internal method to remove a session
func (sm *SessionManager) removeSession(sessionID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		if session.Connection != nil {
			session.Connection.Close()
		}
		delete(sm.sessions, sessionID)
		log.Printf("Removed SSH session %s", sessionID)
	}
}

// isValid checks if a session is still valid
func (s *PersistentSSHSession) isValid() bool {
	// Check if connection is too old (30 minutes max)
	if time.Since(s.CreatedAt) > 30*time.Minute {
		return false
	}

	// Check if connection hasn't been used recently (15 minutes)
	if time.Since(s.LastUsed) > 15*time.Minute {
		return false
	}

	// Check if SSH connection is still alive
	if s.Connection == nil || s.Connection.client == nil {
		return false
	}

	// Try to create a test session
	session, err := s.Connection.client.NewSession()
	if err != nil {
		return false
	}
	session.Close()

	return true
}

// cleanupExpiredSessions removes expired sessions periodically
func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.mutex.Lock()
		var toRemove []string

		for id, session := range sm.sessions {
			if !session.isValid() {
				toRemove = append(toRemove, id)
			}
		}

		for _, id := range toRemove {
			if session, exists := sm.sessions[id]; exists {
				if session.Connection != nil {
					session.Connection.Close()
				}
				delete(sm.sessions, id)
				log.Printf("Cleaned up expired SSH session %s", id)
			}
		}

		sm.mutex.Unlock()
	}
}

// GetActiveSessionCount returns the number of active sessions
func (sm *SessionManager) GetActiveSessionCount() int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return len(sm.sessions)
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CreateTLSSecret creates a Kubernetes TLS secret using Cloudflare certificates
func (ss *SSHService) CreateTLSSecret(conn *SSHConnection, domain, certificate, privateKey string) error {
	// Generate secret name based on domain
	secretName := strings.ReplaceAll(domain, ".", "-") + "-cloudflare-tls"

	// Write certificate to temporary file
	certPath := fmt.Sprintf("/tmp/%s-cert.crt", secretName)
	certCommand := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", certPath, certificate)
	if result, err := ss.ExecuteCommand(conn, certCommand); err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to write certificate: %v", err)
	}

	// Write private key to temporary file
	keyPath := fmt.Sprintf("/tmp/%s-key.key", secretName)
	keyCommand := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", keyPath, privateKey)
	if result, err := ss.ExecuteCommand(conn, keyCommand); err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to write private key: %v", err)
	}

	// Delete existing secret if it exists (ignore errors)
	deleteCommand := fmt.Sprintf("kubectl delete secret %s -n argocd --ignore-not-found=true", secretName)
	ss.ExecuteCommand(conn, deleteCommand)

	// Create the TLS secret in argocd namespace
	createSecretCommand := fmt.Sprintf("kubectl create secret tls %s --cert=%s --key=%s -n argocd",
		secretName, certPath, keyPath)
	if result, err := ss.ExecuteCommand(conn, createSecretCommand); err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to create TLS secret: %s", result.Output)
	}

	// Clean up temporary files
	ss.ExecuteCommand(conn, fmt.Sprintf("rm -f %s %s", certPath, keyPath))

	return nil
}

// CreateArgoCDIngress creates an ingress configuration for ArgoCD
func (ss *SSHService) CreateArgoCDIngress(conn *SSHConnection, domain string) error {
	secretName := strings.ReplaceAll(domain, ".", "-") + "-cloudflare-tls"
	argoCDSubdomain := "argocd." + domain

	ingressManifest := fmt.Sprintf(`apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: %s-argocd-ingress
  namespace: argocd
  annotations:
    traefik.ingress.kubernetes.io/router.tls: "true"
    traefik.ingress.kubernetes.io/router.entrypoints: "websecure"
spec:
  ingressClassName: traefik
  rules:
  - host: "%s"
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: argocd-server
            port:
              number: 80
  tls:
  - hosts:
    - "%s"
    secretName: %s
`, strings.ReplaceAll(domain, ".", "-"), argoCDSubdomain, argoCDSubdomain, secretName)

	// Deploy the ingress manifest
	return ss.DeployManifest(conn, ingressManifest, "argocd-ingress")
}
