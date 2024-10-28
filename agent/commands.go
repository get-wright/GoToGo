// agent/commands.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

type CommandResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error"`
}

func (a *Agent) handleCommands() {
	for {
		select {
		case <-a.stopChan:
			return
		default:
			if err := a.pollCommands(); err != nil {
				log.Printf("Error polling commands: %v", err)
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (a *Agent) pollCommands() error {
	resp, err := a.sendRequest("GET", "/api/agent/commands", map[string]string{
		"agent_id":   a.id,
		"session_id": a.sessionID,
	})
	if err != nil {
		return err
	}

	var commands []struct {
		ID      string `json:"id"`
		Command string `json:"command"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&commands); err != nil {
		return err
	}

	for _, cmd := range commands {
		result := a.executeCommand(cmd.Command)

		// Send result back to server
		a.sendRequest("POST", "/api/agent/result", map[string]interface{}{
			"agent_id":   a.id,
			"command_id": cmd.ID,
			"result":     result,
		})
	}

	return nil
}

func (a *Agent) executeCommand(command string) CommandResult {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return CommandResult{
			Success: false,
			Error:   fmt.Sprintf("%v: %s", err, stderr.String()),
		}
	}

	return CommandResult{
		Success: true,
		Output:  stdout.String(),
	}
}

func (a *Agent) sendHeartbeat() error {
	payload := map[string]string{
		"agent_id":   a.id,
		"session_id": a.sessionID,
	}

	resp, err := a.sendRequest("POST", "/api/agent/heartbeat", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (a *Agent) sendRequest(method, endpoint string, payload interface{}) (*http.Response, error) {
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, a.serverURL+endpoint, &body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return a.client.Do(req)
}
