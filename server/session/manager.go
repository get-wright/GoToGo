// Package session provides session management functionality for the remote management system
package session

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Session represents an active agent session
type Session struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
	Active    bool      `json:"active"`
}

// SessionManager handles session lifecycle and management
type SessionManager struct {
	sessions      map[string]*Session
	sessionsMu    sync.RWMutex
	maxAge        time.Duration
	cleanupTicker *time.Ticker
}

// NewSessionManager creates a new session manager with the specified session max age
func NewSessionManager(maxAge time.Duration) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
		maxAge:   maxAge,
	}

	// Start cleanup routine
	sm.cleanupTicker = time.NewTicker(time.Minute)
	go sm.cleanupRoutine()

	return sm
}

// generateSessionID creates a new random session identifier
func generateSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateSession creates a new session for the specified agent
func (sm *SessionManager) CreateSession(agentID string) (*Session, error) {
	sm.sessionsMu.Lock()
	defer sm.sessionsMu.Unlock()

	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        sessionID,
		AgentID:   agentID,
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
		Active:    true,
	}

	sm.sessions[session.ID] = session
	return session, nil
}

// GetSession retrieves a session by its ID and updates its last seen time
func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.sessionsMu.RLock()
	defer sm.sessionsMu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if exists && session.Active {
		session.LastSeen = time.Now()
	}
	return session, exists
}

// InvalidateSession marks a session as inactive
func (sm *SessionManager) InvalidateSession(sessionID string) {
	sm.sessionsMu.Lock()
	defer sm.sessionsMu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.Active = false
	}
}

// cleanupRoutine periodically removes expired sessions
func (sm *SessionManager) cleanupRoutine() {
	for range sm.cleanupTicker.C {
		sm.sessionsMu.Lock()
		now := time.Now()
		for id, session := range sm.sessions {
			if now.Sub(session.LastSeen) > sm.maxAge {
				delete(sm.sessions, id)
			}
		}
		sm.sessionsMu.Unlock()
	}
}

// Stop stops the session manager's cleanup routine
func (sm *SessionManager) Stop() {
	if sm.cleanupTicker != nil {
		sm.cleanupTicker.Stop()
	}
}
