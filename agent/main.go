// agent/main.go
package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/host"
)

var (
	serverURL = flag.String("server", "https://localhost:8443", "Server URL")
	agentID   = flag.String("id", "", "Agent ID")
	certFile  = flag.String("cert", "agent-cert.pem", "Certificate file")
	keyFile   = flag.String("key", "agent-key.pem", "Key file")
	caFile    = flag.String("ca", "ca-cert.pem", "CA certificate file")
	logFile   = flag.String("log", "agent.log", "Log file")
)

type Agent struct {
	client    *http.Client
	id        string
	sessionID string
	serverURL string
	stopChan  chan struct{}
}

func NewAgent(id, serverURL string, tlsConfig *tls.Config) *Agent {
	return &Agent{
		id:        id,
		serverURL: serverURL,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
			Timeout: 30 * time.Second,
		},
		stopChan: make(chan struct{}),
	}
}

func main() {
	flag.Parse()

	if *agentID == "" {
		log.Fatal("Agent ID is required")
	}

	// Setup logging
	f, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// Load certificates
	cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
	if err != nil {
		log.Fatalf("Failed to load certificates: %v", err)
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(*caFile)
	if err != nil {
		log.Fatalf("Failed to load CA cert: %v", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Configure TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}

	// Create and start agent
	agent := NewAgent(*agentID, *serverURL, tlsConfig)
	if err := agent.Start(); err != nil {
		log.Fatalf("Failed to start agent: %v", err)
	}

	// Wait for shutdown signal
	select {
	case <-agent.stopChan:
		log.Println("Agent shutting down...")
	}
}

func (a *Agent) Start() error {
	// Register with server
	if err := a.register(); err != nil {
		return fmt.Errorf("registration failed: %v", err)
	}

	// Start heartbeat
	go a.heartbeatLoop()

	// Start command handler
	go a.handleCommands()

	return nil
}

func (a *Agent) register() error {
	hostInfo, err := host.Info()
	if err != nil {
		return err
	}

	payload := map[string]string{
		"id":       a.id,
		"hostname": hostInfo.Hostname,
		"os":       hostInfo.OS,
		"ip":       "127.0.0.1", // In production, implement proper IP detection
	}

	resp, err := a.sendRequest("POST", "/api/agent/register", payload)
	if err != nil {
		return err
	}

	var result struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	a.sessionID = result.SessionID
	return nil
}

func (a *Agent) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := a.sendHeartbeat(); err != nil {
				log.Printf("Heartbeat failed: %v", err)
			}
		case <-a.stopChan:
			return
		}
	}
}

// Additional agent methods...
