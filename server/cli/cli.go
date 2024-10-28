package cli

import (
	"GoToGo/server/config"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

type CLI struct {
	server *Server
	config *config.ServerConfig
}

func NewCLI(server *Server, config *config.ServerConfig) *CLI {
	return &CLI{
		server: server,
		config: config,
	}
}

func (c *CLI) Run() {
	c.printBanner()

	for {
		cmd := c.readCommand()
		c.handleCommand(cmd)
	}
}

func (c *CLI) printBanner() {
	color.Cyan(`
    Remote Management Server
    Type 'help' for available commands
    `)
}

func (c *CLI) readCommand() string {
	color.Green("> ")
	var cmd string
	fmt.Scanln(&cmd)
	return strings.TrimSpace(strings.ToLower(cmd))
}

func (c *CLI) handleCommand(cmd string) {
	switch cmd {
	case "help":
		c.showHelp()
	case "agents":
		c.listAgents()
	case "config":
		c.showConfig()
	case "edit":
		c.editConfig()
	case "sessions":
		c.showSessions()
	case "kill":
		c.killSession()
	case "exit":
		os.Exit(0)
	default:
		color.Red("Unknown command. Type 'help' for available commands.")
	}
}

func (c *CLI) showHelp() {
	fmt.Println(`
Available Commands:
    help     - Show this help message
    agents   - List all connected agents
    config   - Show current configuration
    edit     - Edit server configuration
    sessions - Show active sessions
    kill     - Kill an agent session
    exit     - Exit the server
    `)
}

func (c *CLI) listAgents() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Hostname", "Status", "Last Seen"})

	c.server.mutex.RLock()
	for _, agent := range c.server.agents {
		table.Append([]string{
			agent.ID,
			agent.Hostname,
			agent.Status,
			agent.LastSeen,
		})
	}
	c.server.mutex.RUnlock()

	table.Render()
}

func (c *CLI) showConfig() {
	fmt.Printf(`
Current Configuration:
    Port: %d
    Log File: %s
    Agent Update Frequency: %d seconds
    TLS Enabled: %v
    TLS Certificate: %s
    TLS Key: %s
    `,
		c.config.Port,
		c.config.LogFile,
		c.config.AgentUpdateFreq,
		c.config.TLSEnabled,
		c.config.TLSCert,
		c.config.TLSKey,
	)
}

func (c *CLI) editConfig() {
	fmt.Println("Enter new values (press Enter to keep current value):")

	// Port
	fmt.Printf("Port [%d]: ", c.config.Port)
	if input := c.readLine(); input != "" {
		if port, err := strconv.Atoi(input); err == nil {
			c.config.Port = port
		}
	}

	// Log file
	fmt.Printf("Log File [%s]: ", c.config.LogFile)
	if input := c.readLine(); input != "" {
		c.config.LogFile = input
	}

	// Agent update frequency
	fmt.Printf("Agent Update Frequency [%d]: ", c.config.AgentUpdateFreq)
	if input := c.readLine(); input != "" {
		if freq, err := strconv.Atoi(input); err == nil {
			c.config.AgentUpdateFreq = freq
		}
	}

	// TLS
	fmt.Printf("Enable TLS (true/false) [%v]: ", c.config.TLSEnabled)
	if input := c.readLine(); input != "" {
		c.config.TLSEnabled = strings.ToLower(input) == "true"
	}

	if c.config.TLSEnabled {
		fmt.Printf("TLS Certificate Path [%s]: ", c.config.TLSCert)
		if input := c.readLine(); input != "" {
			c.config.TLSCert = input
		}

		fmt.Printf("TLS Key Path [%s]: ", c.config.TLSKey)
		if input := c.readLine(); input != "" {
			c.config.TLSKey = input
		}
	}

	if err := config.SaveConfig(c.config); err != nil {
		color.Red("Error saving configuration: %v", err)
		return
	}

	color.Green("Configuration updated successfully!")
}

func (c *CLI) showSessions() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Session ID", "Agent ID", "Start Time", "Duration"})

	c.server.mutex.RLock()
	for _, session := range c.server.sessions {
		duration := time.Since(session.StartTime)
		table.Append([]string{
			session.ID,
			session.AgentID,
			session.StartTime.Format(time.RFC3339),
			duration.Round(time.Second).String(),
		})
	}
	c.server.mutex.RUnlock()

	table.Render()
}

func (c *CLI) killSession() {
	fmt.Print("Enter session ID to kill: ")
	sessionID := c.readLine()

	if err := c.server.terminateSession(sessionID); err != nil {
		color.Red("Error terminating session: %v", err)
		return
	}

	color.Green("Session terminated successfully!")
}

func (c *CLI) readLine() string {
	var input string
	fmt.Scanln(&input)
	return strings.TrimSpace(input)
}
