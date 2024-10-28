package main

import (
	"GoToGo/server/cli"
	"GoToGo/server/config"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type Agent struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
	Status   string `json:"status"`
	LastSeen string `json:"lastSeen"`
}

type Server struct {
	agents map[string]Agent
	mutex  sync.RWMutex
}

type Command struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type SystemInfo struct {
	CPUUsage    float64 `json:"cpuUsage"`
	MemoryUsage float64 `json:"memoryUsage"`
	DiskUsage   float64 `json:"diskUsage"`
}

func NewServer() *Server {
	return &Server{
		agents: make(map[string]Agent),
	}
}

func (s *Server) handleAgentRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var agent Agent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mutex.Lock()
	s.agents[agent.ID] = agent
	s.mutex.Unlock()

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.Header.Get("X-Agent-ID")
	if agentID == "" {
		http.Error(w, "Agent ID not provided", http.StatusBadRequest)
		return
	}

	var sysInfo SystemInfo
	if err := json.NewDecoder(r.Body).Decode(&sysInfo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Received system info from agent %s: CPU: %.2f%%, Memory: %.2f%%, Disk: %.2f%%",
		agentID, sysInfo.CPUUsage, sysInfo.MemoryUsage, sysInfo.DiskUsage)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mutex.RLock()
	agents := make([]Agent, 0, len(s.agents))
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}
	s.mutex.RUnlock()

	json.NewEncoder(w).Encode(agents)
}

func main() {
	// Load configuration
	config, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Initialize server
	server := NewServer()

	// Initialize CLI
	cli := cli.NewCLI(server, config)

	// Start HTTP server in a goroutine
	go func() {
		addr := fmt.Sprintf(":%d", config.Port)
		if config.TLSEnabled {
			log.Printf("Server starting with TLS on %s...", addr)
			log.Fatal(http.ListenAndServeTLS(addr, config.TLSCert, config.TLSKey, nil))
		} else {
			log.Printf("Server starting on %s...", addr)
			log.Fatal(http.ListenAndServe(addr, nil))
		}
	}()

	// Start CLI
	cli.Run()
}
