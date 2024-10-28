#!/bin/bash

echo "Setting up Remote Management Server..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Please install Go first."
    exit 1
fi

# Check and install dependencies
echo "Checking dependencies..."
go mod init remote-management
go get github.com/shirou/gopsutil
go get github.com/spf13/cobra
go get github.com/spf13/viper
go get github.com/fatih/color
go get github.com/olekukonko/tablewriter

# Compile server binary
echo "Compiling server..."
go build -o rmserver server/main.go

echo "Server setup complete. Run ./rmserver to start the server."

# setup-agent.sh
#!/bin/bash

echo "Setting up Remote Management Agent..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Please install Go first."
    exit 1
fi

# Check and install dependencies
echo "Checking dependencies..."
go mod init remote-management-agent
go get github.com/shirou/gopsutil

# Compile agent binary
echo "Compiling agent..."
go build -o rmagent agent/main.go

echo "Agent setup complete. Run ./rmagent <server-url> to start the agent."