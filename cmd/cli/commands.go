// cmd/cli/commands.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"net/http"
	"os"
	"strings"
)

func (c *CLI) cmdConnectAgent(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: connect <agent-id>")
	}

	agentID := args[0]
	yellow := color.New(color.FgYellow)
	yellow.Printf("Connecting to agent %s...\n", agentID)

	// Verify agent exists and is active
	resp, err := c.sendRequest("GET", fmt.Sprintf("/api/agents/%s", agentID), nil)
	if err != nil {
		return err
	}

	var agent struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&agent); err != nil {
		return err
	}

	if agent.Status != "active" {
		return fmt.Errorf("agent is not active")
	}

	// Enter interactive mode
	return c.interactiveMode(agentID)
}

func (c *CLI) interactiveMode(agentID string) error {
	cyan := color.New(color.FgCyan)

	for {
		cyan.Printf("%s> ", agentID)
		if !c.scanner.Scan() {
			break
		}

		cmd := strings.TrimSpace(c.scanner.Text())
		if cmd == "exit" || cmd == "quit" {
			break
		}

		if cmd == "" {
			continue
		}

		// Execute command on agent
		err := c.cmdExecCommand([]string{agentID, cmd})
		if err != nil {
			color.Red("Error: %v", err)
		}
	}

	return nil
}

func (c *CLI) cmdExecCommand(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: exec <agent-id> <command>")
	}

	agentID := args[0]
	command := strings.Join(args[1:], " ")

	payload := map[string]string{
		"agent_id": agentID,
		"command":  command,
	}

	resp, err := c.sendRequest("POST", "/api/agent/execute", payload)
	if err != nil {
		return err
	}

	var result struct {
		Success bool   `json:"success"`
		Output  string `json:"output"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("command failed: %s", result.Error)
	}

	fmt.Println(result.Output)
	return nil
}

func (c *CLI) cmdGenerateCert(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: gen <agent-id>")
	}

	agentID := args[0]
	payload := map[string]string{
		"agent_id": agentID,
	}

	resp, err := c.sendRequest("POST", "/api/generate-cert", payload)
	if err != nil {
		return err
	}

	var result struct {
		CertPath string `json:"cert_path"`
		KeyPath  string `json:"key_path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	color.Green("Generated certificates for agent %s:", agentID)
	fmt.Printf("  Certificate: %s\n", result.CertPath)
	fmt.Printf("  Private Key: %s\n", result.KeyPath)
	return nil
}

func (c *CLI) cmdExit(args []string) error {
	fmt.Println("Goodbye!")
	os.Exit(0)
	return nil
}

func (c *CLI) sendRequest(method, endpoint string, payload interface{}) (*http.Response, error) {
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, c.serverURL+endpoint, &body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return c.client.Do(req)
}
