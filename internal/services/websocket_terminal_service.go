package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

// WebSocketTerminalService manages WebSocket terminal sessions with SSH connections
type WebSocketTerminalService struct {
	sessions map[string]*WebSocketTerminalSession
	mutex    sync.RWMutex
}

// WebSocketTerminalSession represents a WebSocket terminal session with SSH bridge
type WebSocketTerminalSession struct {
	ID            string                 `json:"id"`
	ServerID      int                    `json:"server_id"`
	Host          string                 `json:"host"`
	User          string                 `json:"user"`
	Status        string                 `json:"status"`
	StartedAt     time.Time              `json:"started_at"`
	LastActivity  time.Time              `json:"last_activity"`
	AccountID     string                 `json:"account_id"`
	sshClient     *ssh.Client
	sshSession    *ssh.Session
	stdin         io.WriteCloser
	stdout        io.Reader
	stderr        io.Reader
	context       context.Context
	cancel        context.CancelFunc
	connections   map[*websocket.Conn]bool
	connMutex     sync.RWMutex
}

// TerminalMessage represents a message sent over WebSocket
type TerminalMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

// NewWebSocketTerminalService creates a new WebSocket terminal service
func NewWebSocketTerminalService() *WebSocketTerminalService {
	service := &WebSocketTerminalService{
		sessions: make(map[string]*WebSocketTerminalSession),
	}
	
	// Start cleanup routine
	go service.cleanupRoutine()
	
	return service
}

// CreateSession creates a new WebSocket terminal session with SSH connection
func (s *WebSocketTerminalService) CreateSession(serverID int, host, user, privateKey, token, accountID string) (*WebSocketTerminalSession, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Generate unique session ID
	sessionID, err := generateSecureSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %v", err)
	}

	// Parse private key
	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	// Create SSH client configuration
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, should verify host keys
		Timeout:         30 * time.Second,
	}

	// Create context for session management
	ctx, cancel := context.WithCancel(context.Background())

	// Create session
	session := &WebSocketTerminalSession{
		ID:           sessionID,
		ServerID:     serverID,
		Host:         host,
		User:         user,
		Status:       "connecting",
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
		AccountID:    accountID,
		context:      ctx,
		cancel:       cancel,
		connections:  make(map[*websocket.Conn]bool),
	}

	// Store session immediately
	s.sessions[sessionID] = session

	// Connect to SSH in background
	go func() {
		if err := s.connectSSH(session, config); err != nil {
			log.Printf("Failed to connect SSH for session %s: %v", sessionID, err)
			session.Status = "failed"
		}
	}()

	log.Printf("Created WebSocket terminal session %s for server %d", sessionID, serverID)
	return session, nil
}

// connectSSH establishes SSH connection for the session
func (s *WebSocketTerminalService) connectSSH(session *WebSocketTerminalSession, config *ssh.ClientConfig) error {
	// Connect to SSH server
	client, err := ssh.Dial("tcp", session.Host+":22", config)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server: %v", err)
	}

	session.sshClient = client
	session.Status = "connected"
	session.LastActivity = time.Now()

	log.Printf("SSH connection established for session %s", session.ID)
	return nil
}

// HandleWebSocketConnection handles a WebSocket connection for a terminal session
func (s *WebSocketTerminalService) HandleWebSocketConnection(sessionID string, conn *websocket.Conn) error {
	s.mutex.RLock()
	session, exists := s.sessions[sessionID]
	s.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("session not found")
	}

	// Add connection to session
	session.connMutex.Lock()
	session.connections[conn] = true
	session.connMutex.Unlock()

	// Remove connection when done
	defer func() {
		session.connMutex.Lock()
		delete(session.connections, conn)
		session.connMutex.Unlock()
	}()

	// Wait for SSH connection if still connecting
	for session.Status == "connecting" {
		select {
		case <-session.context.Done():
			return fmt.Errorf("session cancelled")
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}

	if session.Status != "connected" {
		return fmt.Errorf("SSH connection failed")
	}

	// Create SSH session if not exists
	if session.sshSession == nil {
		sshSession, err := session.sshClient.NewSession()
		if err != nil {
			return fmt.Errorf("failed to create SSH session: %v", err)
		}

		// Set up terminal
		if err := sshSession.RequestPty("xterm-256color", 24, 80, ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}); err != nil {
			sshSession.Close()
			return fmt.Errorf("failed to request pty: %v", err)
		}

		// Get stdin/stdout pipes
		stdin, err := sshSession.StdinPipe()
		if err != nil {
			sshSession.Close()
			return fmt.Errorf("failed to get stdin pipe: %v", err)
		}

		stdout, err := sshSession.StdoutPipe()
		if err != nil {
			sshSession.Close()
			return fmt.Errorf("failed to get stdout pipe: %v", err)
		}

		stderr, err := sshSession.StderrPipe()
		if err != nil {
			sshSession.Close()
			return fmt.Errorf("failed to get stderr pipe: %v", err)
		}

		session.sshSession = sshSession
		session.stdin = stdin
		session.stdout = stdout
		session.stderr = stderr

		// Start shell
		if err := sshSession.Shell(); err != nil {
			sshSession.Close()
			return fmt.Errorf("failed to start shell: %v", err)
		}

		session.Status = "running"

		// Start output forwarding to all WebSocket connections
		go s.forwardOutput(session, stdout, "stdout")
		go s.forwardOutput(session, stderr, "stderr")
	}

	// Handle WebSocket messages (input from client)
	return s.handleWebSocketMessages(session, conn)
}

// forwardOutput forwards SSH output to all WebSocket connections
func (s *WebSocketTerminalService) forwardOutput(session *WebSocketTerminalSession, reader io.Reader, streamType string) {
	buffer := make([]byte, 1024)
	for {
		select {
		case <-session.context.Done():
			return
		default:
			n, err := reader.Read(buffer)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading from SSH %s: %v", streamType, err)
				}
				return
			}

			if n > 0 {
				session.LastActivity = time.Now()
				message := TerminalMessage{
					Type: "output",
					Data: string(buffer[:n]),
				}

				data, err := json.Marshal(message)
				if err != nil {
					continue
				}

				// Send to all connected WebSocket clients
				session.connMutex.RLock()
				for conn := range session.connections {
					conn.WriteMessage(websocket.TextMessage, data)
				}
				session.connMutex.RUnlock()
			}
		}
	}
}

// handleWebSocketMessages handles incoming WebSocket messages (user input)
func (s *WebSocketTerminalService) handleWebSocketMessages(session *WebSocketTerminalSession, conn *websocket.Conn) error {
	for {
		select {
		case <-session.context.Done():
			return nil
		default:
			_, messageBytes, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return err
			}

			var message TerminalMessage
			if err := json.Unmarshal(messageBytes, &message); err != nil {
				continue
			}

			session.LastActivity = time.Now()

			switch message.Type {
			case "input":
				if session.stdin != nil {
					session.stdin.Write([]byte(message.Data))
				}
			case "resize":
				// Handle terminal resize
				// Note: Would need to parse resize data and call session.WindowChange()
			}
		}
	}
}

// GetSession retrieves a terminal session by ID
func (s *WebSocketTerminalService) GetSession(sessionID string) (*WebSocketTerminalSession, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

// StopSession stops a terminal session
func (s *WebSocketTerminalService) StopSession(sessionID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Cancel context to stop all goroutines
	if session.cancel != nil {
		session.cancel()
	}

	// Close SSH session
	if session.sshSession != nil {
		session.sshSession.Close()
	}

	// Close SSH client
	if session.sshClient != nil {
		session.sshClient.Close()
	}

	// Close all WebSocket connections
	session.connMutex.Lock()
	for conn := range session.connections {
		conn.Close()
	}
	session.connMutex.Unlock()

	session.Status = "stopped"
	delete(s.sessions, sessionID)

	log.Printf("Stopped WebSocket terminal session %s", sessionID)
	return nil
}

// ListSessionsForAccount returns all sessions for a specific account
func (s *WebSocketTerminalService) ListSessionsForAccount(accountID string) []*WebSocketTerminalSession {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var sessions []*WebSocketTerminalSession
	for _, session := range s.sessions {
		if session.AccountID == accountID {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

// cleanupRoutine periodically cleans up inactive sessions
func (s *WebSocketTerminalService) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanupInactiveSessions()
	}
}

// cleanupInactiveSessions removes sessions that have been inactive for too long
func (s *WebSocketTerminalService) cleanupInactiveSessions() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	cutoff := time.Now().Add(-30 * time.Minute) // 30 minutes timeout
	
	for sessionID, session := range s.sessions {
		if session.LastActivity.Before(cutoff) {
			log.Printf("Cleaning up inactive session %s", sessionID)
			
			// Cancel context and close connections
			if session.cancel != nil {
				session.cancel()
			}
			
			if session.sshSession != nil {
				session.sshSession.Close()
			}
			
			if session.sshClient != nil {
				session.sshClient.Close()
			}
			
			session.connMutex.Lock()
			for conn := range session.connections {
				conn.Close()
			}
			session.connMutex.Unlock()
			
			delete(s.sessions, sessionID)
		}
	}
}

// generateSecureSessionID generates a cryptographically secure session ID
func generateSecureSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}