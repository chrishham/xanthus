package services

import (
	"fmt"
	"log"
	"sync"
	"time"
)

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
