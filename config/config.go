// config/config.go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ServerConfig struct {
	Port           string `json:"port"`
	CertDirectory  string `json:"cert_directory"`
	LogDirectory   string `json:"log_directory"`
	SessionTimeout string `json:"session_timeout"`
}

type AgentConfig struct {
	ServerURL     string `json:"server_url"`
	ID            string `json:"id"`
	CertFile      string `json:"cert_file"`
	KeyFile       string `json:"key_file"`
	CAFile        string `json:"ca_file"`
	LogFile       string `json:"log_file"`
	PollInterval  string `json:"poll_interval"`
	HeartbeatRate string `json:"heartbeat_rate"`
}

type CLIConfig struct {
	ServerURL string `json:"server_url"`
}

func LoadConfig[T any](configPath string, defaultConfig T) (*T, error) {
	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	// If config file doesn't exist, create it with default values
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		file, err := os.Create(configPath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(defaultConfig); err != nil {
			return nil, err
		}
		return &defaultConfig, nil
	}

	// Load existing config
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config T
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
