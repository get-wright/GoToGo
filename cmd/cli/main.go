// cmd/cli/main.go
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/fatih/color"
)

var (
	serverURL = flag.String("server", "https://localhost:8443", "Server URL")
)

type CLI struct {
	client    *http.Client
	serverURL string
	scanner   *bufio.Scanner
	commands  map[string]func([]string) error
}

func NewCLI(serverURL string) *CLI {
	cli := &CLI{
		serverURL: serverURL,
		scanner:   bufio.NewScanner(os.Stdin),
		commands:  make(map[string]func([]string) error),
		client:    &http.Client{},
	}

	cli.registerCommands()
	return cli
}

func main() {
	flag.Parse()

	cli := NewCLI(*serverURL)
	cli.Run()
}

func (c *CLI) registerCommands() {
	c.commands = map[string]func([]string) error{
		"help":    c.cmdHelp,
		"list":    c.cmdListAgents,
		"connect": c.cmdConnectAgent,
		"exec":    c.cmdExecCommand,
		"gen":     c.cmdGenerateCert,
		"exit":    c.cmdExit,
	}
}

func (c *CLI) Run() {
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)

	bold.Println("Remote Management CLI")
	bold.Println("Type 'help' for available commands")

	for {
		cyan.Print("\n> ")
		if !c.scanner.Scan() {
			break
		}

		input := strings.TrimSpace(c.scanner.Text())
		if input == "" {
			continue
		}

		args := strings.Fields(input)
		cmd := args[0]

		if handler, exists := c.commands[cmd]; exists {
			if err := handler(args[1:]); err != nil {
				color.Red("Error: %v", err)
			}
		} else {
			color.Red("Unknown command. Type 'help' for available commands")
		}
	}
}

func (c *CLI) cmdHelp(args []string) error {
	fmt.Println("\nAvailable commands:")
	fmt.Println("  help                  - Show this help message")
	fmt.Println("  list                  - List all connected agents")
	fmt.Println("  connect <agent-id>    - Connect to a specific agent")
	fmt.Println("  exec <agent-id> <cmd> - Execute command on agent")
	fmt.Println("  gen <agent-id>        - Generate new agent certificate")
	fmt.Println("  exit                  - Exit the CLI")
	return nil
}

func (c *CLI) cmdListAgents(args []string) error {
	resp, err := c.sendRequest("GET", "/api/agents", nil)
	if err != nil {
		return err
	}

	var agents []struct {
		ID       string `json:"id"`
		Hostname string `json:"hostname"`
		Status   string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		return err
	}

	fmt.Println("\nConnected Agents:")
	for _, agent := range agents {
		status := color.GreenString("ACTIVE")
		if agent.Status != "active" {
			status = color.RedString("INACTIVE")
		}
		fmt.Printf("  %s (%s) - %s\n", agent.ID, agent.Hostname, status)
	}

	return nil
}

// Additional CLI command methods...
