// cmd/setup/main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	installDir = flag.String("dir", ".", "Installation directory")
)

// downloadDependencies ensures all required dependencies are downloaded
func downloadDependencies() error {
	fmt.Println("Downloading dependencies...")

	deps := []string{
		"github.com/fatih/color@v1.15.0",
		"github.com/google/uuid@v1.6.0",
		"github.com/shirou/gopsutil/v3@v3.24.5",
		"golang.org/x/crypto@v0.14.0",
	}

	for _, dep := range deps {
		fmt.Printf("Getting %s...\n", dep)
		cmd := exec.Command("go", "get", dep)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to download %s: %v", dep, err)
		}
	}

	// Run go mod tidy to clean up dependencies
	fmt.Println("Running go mod tidy...")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run go mod tidy: %v", err)
	}

	return nil
}

func main() {
	flag.Parse()

	// Initialize go module if needed
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		fmt.Println("Initializing Go module...")
		cmd := exec.Command("go", "mod", "init", "GoToGo")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("Failed to initialize Go module: %v", err)
		}
	}

	// Download dependencies
	if err := downloadDependencies(); err != nil {
		log.Fatalf("Failed to download dependencies: %v", err)
	}

	// Create directory structure
	dirs := []string{
		"certs",
		"logs",
		"config",
		"bin",
	}

	for _, dir := range dirs {
		path := filepath.Join(*installDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", path, err)
		}
	}

	// Build binaries
	binaries := map[string]string{
		"server": "server/main.go",
		"agent":  "agent/main.go agent/commands.go",
		"cli":    "cmd/cli/main.go cmd/cli/commands.go",
	}

	for name, src := range binaries {
		fmt.Printf("Building %s...\n", name)

		// Prepare command arguments
		args := []string{"build", "-o", filepath.Join(*installDir, "bin", "gotogo-"+name)}
		args = append(args, strings.Fields(src)...)

		cmd := exec.Command("go", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("Failed to build %s: %v", name, err)
		}
	}

	// Generate default configs
	configFiles := map[string]interface{}{
		"config/server.json": ServerConfig{
			Port:           "8443",
			CertDirectory:  "certs",
			LogDirectory:   "logs",
			SessionTimeout: "24h",
		},
		"config/agent.json": AgentConfig{
			ServerURL:     "https://localhost:8443",
			PollInterval:  "5s",
			HeartbeatRate: "30s",
		},
		"config/cli.json": CLIConfig{
			ServerURL: "https://localhost:8443",
		},
	}

	for file, defaultConfig := range configFiles {
		path := filepath.Join(*installDir, file)
		if err := saveConfig(path, defaultConfig); err != nil {
			log.Fatalf("Failed to create config %s: %v", file, err)
		}
	}

	fmt.Println("\nGoToGo installation complete!")
	fmt.Println("\nTo start using GoToGo:")
	fmt.Println("1. Start the server:   ./bin/gotogo-server")
	fmt.Println("2. Generate an agent certificate:   ./bin/gotogo-cli gen <agent-id>")
	fmt.Println("3. Start an agent:     ./bin/gotogo-agent -id <agent-id>")
	fmt.Println("4. Use the CLI:        ./bin/gotogo-cli")
}

// Configuration structures
type ServerConfig struct {
	Port           string `json:"port"`
	CertDirectory  string `json:"cert_directory"`
	LogDirectory   string `json:"log_directory"`
	SessionTimeout string `json:"session_timeout"`
}

type AgentConfig struct {
	ServerURL     string `json:"server_url"`
	PollInterval  string `json:"poll_interval"`
	HeartbeatRate string `json:"heartbeat_rate"`
}

type CLIConfig struct {
	ServerURL string `json:"server_url"`
}

// saveConfig saves a configuration structure to a JSON file
func saveConfig(path string, config interface{}) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create and write to file
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(config)
}
