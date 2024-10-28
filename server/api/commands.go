// server/api/commands.go
package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type Command struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	Command   string    `json:"command"`
	Status    string    `json:"status"`
	Result    string    `json:"result"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CommandQueue struct {
	commands map[string]*Command
	lock     sync.RWMutex
}

func NewCommandQueue() *CommandQueue {
	return &CommandQueue{
		commands: make(map[string]*Command),
	}
}

func (h *Handler) handleAgentExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID string `json:"agent_id"`
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Verify agent exists and is active
	h.agentsMux.RLock()
	agent, exists := h.agents[req.AgentID]
	h.agentsMux.RUnlock()

	if !exists || agent.Status != "active" {
		http.Error(w, "Agent not found or inactive", http.StatusNotFound)
		return
	}

	// Queue command
	cmd := &Command{
		ID:        generateID(),
		AgentID:   req.AgentID,
		Command:   req.Command,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	h.commandQueue.lock.Lock()
	h.commandQueue.commands[cmd.ID] = cmd
	h.commandQueue.lock.Unlock()

	// Wait for result (with timeout)
	result := h.waitForResult(cmd.ID, 30*time.Second)
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) handleAgentCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		http.Error(w, "Agent ID required", http.StatusBadRequest)
		return
	}

	// Get pending commands for agent
	h.commandQueue.lock.RLock()
	pending := make([]*Command, 0)
	for _, cmd := range h.commandQueue.commands {
		if cmd.AgentID == agentID && cmd.Status == "pending" {
			pending = append(pending, cmd)
		}
	}
	h.commandQueue.lock.RUnlock()

	json.NewEncoder(w).Encode(pending)
}

func (h *Handler) handleCommandResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var result struct {
		CommandID string `json:"command_id"`
		Success   bool   `json:"success"`
		Output    string `json:"output"`
		Error     string `json:"error"`
	}
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	h.commandQueue.lock.Lock()
	if cmd, exists := h.commandQueue.commands[result.CommandID]; exists {
		cmd.Status = "completed"
		if result.Success {
			cmd.Result = result.Output
		} else {
			cmd.Result = result.Error
		}
		cmd.UpdatedAt = time.Now()
	}
	h.commandQueue.lock.Unlock()

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) waitForResult(commandID string, timeout time.Duration) *Command {
	deadline := time.After(timeout)
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-deadline:
			return &Command{
				Status: "timeout",
				Result: "Command timed out",
			}
		case <-tick.C:
			h.commandQueue.lock.RLock()
			cmd, exists := h.commandQueue.commands[commandID]
			completed := exists && cmd.Status == "completed"
			if completed {
				result := *cmd // Make a copy of the command
				h.commandQueue.lock.RUnlock()
				return &result
			}
			h.commandQueue.lock.RUnlock()
		}
	}
}
