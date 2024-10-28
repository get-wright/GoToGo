// server/api/handlers.go
package api

import (
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
	"sync"
	"time"

	"GoToGo/server/cert"
	"GoToGo/server/session"
)

type Handler struct {
	certManager    *cert.CertManager
	sessionManager *session.SessionManager
	agents         map[string]*Agent
	agentsMux      sync.RWMutex
	commandQueue   *CommandQueue
}

type Agent struct {
	ID       string    `json:"id"`
	Hostname string    `json:"hostname"`
	IP       string    `json:"ip"`
	OS       string    `json:"os"`
	LastSeen time.Time `json:"last_seen"`
	Status   string    `json:"status"`
}

func NewHandler(cm *cert.CertManager, sm *session.SessionManager) *Handler {
	return &Handler{
		certManager:    cm,
		sessionManager: sm,
		agents:         make(map[string]*Agent),
		commandQueue:   NewCommandQueue(),
	}
}

func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()

	// Agent endpoints
	mux.HandleFunc("/api/agent/register", h.handleAgentRegister)
	mux.HandleFunc("/api/agent/heartbeat", h.handleAgentHeartbeat)
	mux.HandleFunc("/api/agent/execute", h.handleAgentExecute)
	mux.HandleFunc("/api/agent/commands", h.handleAgentCommands)
	mux.HandleFunc("/api/agent/result", h.handleCommandResult)

	// Management endpoints
	mux.HandleFunc("/api/agents", h.handleListAgents)
	mux.HandleFunc("/api/generate-cert", h.handleGenerateCert)

	return mux
}

func (h *Handler) handleAgentRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var agent Agent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	agent.LastSeen = time.Now()
	agent.Status = "active"

	h.agentsMux.Lock()
	h.agents[agent.ID] = &agent
	h.agentsMux.Unlock()

	// Create session
	session, err := h.sessionManager.CreateSession(agent.ID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"session_id": session.ID,
		"status":     "registered",
	})
}

func (h *Handler) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var heartbeat struct {
		AgentID   string `json:"agent_id"`
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&heartbeat); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Verify session
	session, exists := h.sessionManager.GetSession(heartbeat.SessionID)
	if !exists || session.AgentID != heartbeat.AgentID {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	h.agentsMux.Lock()
	if agent, exists := h.agents[heartbeat.AgentID]; exists {
		agent.LastSeen = time.Now()
		agent.Status = "active"
	}
	h.agentsMux.Unlock()

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleListAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.agentsMux.RLock()
	agents := make([]*Agent, 0, len(h.agents))
	for _, agent := range h.agents {
		agents = append(agents, agent)
	}
	h.agentsMux.RUnlock()

	json.NewEncoder(w).Encode(agents)
}

func (h *Handler) handleGenerateCert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	certPath, keyPath, err := h.certManager.GenerateClientCert(req.AgentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"cert_path": certPath,
		"key_path":  keyPath,
	})
}

// generateID generates a unique ID using UUID
func generateID() string {
	return uuid.New().String()
}
