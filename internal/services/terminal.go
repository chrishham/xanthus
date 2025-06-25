package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strconv"
	"time"
)

// TerminalService handles web terminal sessions using GoTTY
type TerminalService struct {
	sessions map[string]*TerminalSession
}

// TerminalSession represents an active terminal session
type TerminalSession struct {
	ID       string    `json:"id"`
	ServerID int       `json:"server_id"`
	Host     string    `json:"host"`
	User     string    `json:"user"`
	Port     int       `json:"port"`
	Status   string    `json:"status"`
	PID      int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
	process  *exec.Cmd
	cancel   context.CancelFunc
}

// NewTerminalService creates a new terminal service instance
func NewTerminalService() *TerminalService {
	return &TerminalService{
		sessions: make(map[string]*TerminalSession),
	}
}

// CreateSession creates a new terminal session for SSH connection to VPS
func (ts *TerminalService) CreateSession(serverID int, host, user, privateKey string) (*TerminalSession, error) {
	// Generate unique session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %v", err)
	}

	// Find available port for GoTTY
	port, err := findAvailablePort(9000, 9100)
	if err != nil {
		return nil, fmt.Errorf("no available ports: %v", err)
	}

	// Create temporary SSH key file for this session
	keyFile, err := writeSSHKeyToTemp(privateKey, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH key file: %v", err)
	}

	// Create session
	session := &TerminalSession{
		ID:        sessionID,
		ServerID:  serverID,
		Host:      host,
		User:      user,
		Port:      port,
		Status:    "starting",
		StartedAt: time.Now(),
	}

	// Start GoTTY with SSH connection
	ctx, cancel := context.WithCancel(context.Background())
	session.cancel = cancel

	// GoTTY command with SSH
	cmd := exec.CommandContext(ctx, "gotty",
		"--port", strconv.Itoa(port),
		"--permit-write",
		"--reconnect",
		"--title-format", fmt.Sprintf("Xanthus SSH - %s@%s", user, host),
		"--close-signal", "9",
		"--close-timeout", "10",
		"ssh",
		"-i", keyFile,
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "StrictHostKeyChecking=no",
		"-o", "ConnectTimeout=10",
		fmt.Sprintf("%s@%s", user, host),
	)

	session.process = cmd

	// Start the process
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start GoTTY: %v", err)
	}

	session.PID = cmd.Process.Pid
	session.Status = "running"

	// Store session
	ts.sessions[sessionID] = session

	// Monitor the process in a goroutine
	go func() {
		defer func() {
			session.Status = "stopped"
			// Clean up temporary SSH key file
			exec.Command("rm", "-f", keyFile).Run()
			delete(ts.sessions, sessionID)
		}()

		err := cmd.Wait()
		if err != nil {
			log.Printf("Terminal session %s ended with error: %v", sessionID, err)
		} else {
			log.Printf("Terminal session %s ended normally", sessionID)
		}
	}()

	log.Printf("Started terminal session %s for server %d on port %d", sessionID, serverID, port)
	return session, nil
}

// GetSession retrieves a terminal session by ID
func (ts *TerminalService) GetSession(sessionID string) (*TerminalSession, error) {
	session, exists := ts.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

// StopSession stops a terminal session
func (ts *TerminalService) StopSession(sessionID string) error {
	session, exists := ts.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	if session.cancel != nil {
		session.cancel()
	}

	if session.process != nil && session.process.Process != nil {
		if err := session.process.Process.Kill(); err != nil {
			log.Printf("Failed to kill process for session %s: %v", sessionID, err)
		}
	}

	session.Status = "stopped"
	delete(ts.sessions, sessionID)

	log.Printf("Stopped terminal session %s", sessionID)
	return nil
}

// ListSessions returns all active sessions
func (ts *TerminalService) ListSessions() []*TerminalSession {
	sessions := make([]*TerminalSession, 0, len(ts.sessions))
	for _, session := range ts.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// CleanupSessions removes stopped sessions
func (ts *TerminalService) CleanupSessions() {
	for sessionID, session := range ts.sessions {
		if session.Status == "stopped" || session.process == nil {
			delete(ts.sessions, sessionID)
		}
	}
}

// generateSessionID creates a random session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// findAvailablePort finds an available port in the given range
func findAvailablePort(start, end int) (int, error) {
	for port := start; port <= end; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports in range %d-%d", start, end)
}

// writeSSHKeyToTemp writes the private key to a temporary file
func writeSSHKeyToTemp(privateKey, sessionID string) (string, error) {
	keyFile := fmt.Sprintf("/tmp/xanthus-ssh-%s.pem", sessionID)
	
	cmd := exec.Command("bash", "-c", fmt.Sprintf("echo '%s' > %s && chmod 600 %s", privateKey, keyFile, keyFile))
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to write SSH key file: %v", err)
	}
	
	return keyFile, nil
}