package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

type Agent struct {
	ID        string
	Hostname  string
	ServerURL string
}

type SystemInfo struct {
	CPUUsage    float64 `json:"cpuUsage"`
	MemoryUsage float64 `json:"memoryUsage"`
	DiskUsage   float64 `json:"diskUsage"`
}

func NewAgent(serverURL string) (*Agent, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return &Agent{
		ID:        fmt.Sprintf("%s-%d", hostname, time.Now().Unix()),
		Hostname:  hostname,
		ServerURL: serverURL,
	}, nil
}

func (a *Agent) register() error {
	data, err := json.Marshal(map[string]string{
		"id":       a.ID,
		"hostname": a.Hostname,
		"status":   "online",
		"lastSeen": time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}

	resp, err := http.Post(a.ServerURL+"/register", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (a *Agent) collectSystemInfo() (SystemInfo, error) {
	var info SystemInfo

	// CPU usage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return info, err
	}
	info.CPUUsage = cpuPercent[0]

	// Memory usage
	memStat, err := mem.VirtualMemory()
	if err != nil {
		return info, err
	}
	info.MemoryUsage = memStat.UsedPercent

	// Disk usage
	diskStat, err := disk.Usage("/")
	if err != nil {
		return info, err
	}
	info.DiskUsage = diskStat.UsedPercent

	return info, nil
}

func (a *Agent) reportSystemInfo() error {
	sysInfo, err := a.collectSystemInfo()
	if err != nil {
		return err
	}

	data, err := json.Marshal(sysInfo)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", a.ServerURL+"/system-info", bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-ID", a.ID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to report system info with status: %d", resp.StatusCode)
	}

	return nil
}

func main() {
	serverURL := "http://localhost:8080"
	if len(os.Args) > 1 {
		serverURL = os.Args[1]
	}

	agent, err := NewAgent(serverURL)
	if err != nil {
		log.Fatal(err)
	}

	if err := agent.register(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Agent registered with ID: %s\n", agent.ID)

	// Report system information periodically
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := agent.reportSystemInfo(); err != nil {
				log.Printf("Error reporting system info: %v", err)
			}
		}
	}
}
