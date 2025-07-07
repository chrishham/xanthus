package services

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
	"net"
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

// SSHConnection represents an active SSH connection
type SSHConnection struct {
	client  *ssh.Client
	session *ssh.Session
}

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

// Close closes the SSH connection
func (conn *SSHConnection) Close() error {
	if conn.session != nil {
		conn.session.Close()
	}
	return conn.client.Close()
}
