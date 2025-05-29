// client.go
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// Client represents a game client
type Client struct {
	conn     net.Conn
	scanner  *bufio.Scanner
	username string
	running  bool
	mu       sync.Mutex
}

// NewClient creates a new client instance
func NewClient() *Client {
	return &Client{
		running: true,
	}
}

// Connect establishes connection to the server
func (c *Client) Connect(serverAddr string) error {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
	}

	c.conn = conn
	c.scanner = bufio.NewScanner(conn)

	fmt.Println("Connected to TCR Server!")
	return nil
}

// Start begins the client session
func (c *Client) Start() {
	defer c.conn.Close()

	// Start listening for server messages
	go c.listenForMessages()

	// Handle user input
	c.handleUserInput()
}

// listenForMessages receives and displays server messages
func (c *Client) listenForMessages() {
	for c.scanner.Scan() {
		message := c.scanner.Text()

		c.mu.Lock()
		if c.running {
			fmt.Print(message)
			if !strings.HasSuffix(message, "\n") {
				fmt.Println()
			}
		}
		c.mu.Unlock()

		// Check for specific prompts
		if strings.Contains(message, "Enter username:") ||
			strings.Contains(message, "Enter password:") {
			// Don't add extra newline for input prompts
		}
	}

	c.mu.Lock()
	c.running = false
	c.mu.Unlock()
}

// handleUserInput processes user commands
func (c *Client) handleUserInput() {
	reader := bufio.NewReader(os.Stdin)

	for c.running {
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			break
		}

		input = strings.TrimSpace(input)

		if input == "quit" || input == "exit" {
			c.mu.Lock()
			c.running = false
			c.mu.Unlock()
			break
		}

		// Send command to server
		_, err = c.conn.Write([]byte(input + "\n"))
		if err != nil {
			fmt.Printf("Error sending command: %v\n", err)
			break
		}
	}
}

// Disconnect closes the connection
func (c *Client) Disconnect() {
	c.mu.Lock()
	c.running = false
	c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
	}
}

// printWelcome displays client welcome message
func printWelcome() {
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║     Text-Based Clash Royale Client   ║")
	fmt.Println("║              TCR v2.0                ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()
}

// printUsage shows command line usage
func printUsage() {
	fmt.Println("Usage: go run client.go [server_address]")
	fmt.Println("Default server address: localhost:8080")
	fmt.Println()
	fmt.Println("Game Commands (once connected):")
	fmt.Println("  help           - Show available commands")
	fmt.Println("  status         - Show current game state")
	fmt.Println("  attack <1-3> <target> - Attack with troop")
	fmt.Println("  quit           - Leave the game")
	fmt.Println()
	fmt.Println("Attack Targets:")
	fmt.Println("  king           - King Tower")
	fmt.Println("  guard1         - First Guard Tower")
	fmt.Println("  guard2         - Second Guard Tower")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  attack 1 guard1   - Attack first guard with first troop")
	fmt.Println("  attack 2 king     - Attack king with second troop")
}

// main function for client
func main() {
	printWelcome()

	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		printUsage()
		return
	}

	serverAddr := "localhost:8080"
	if len(os.Args) > 1 {
		serverAddr = os.Args[1]
	}

	fmt.Printf("Connecting to server: %s\n", serverAddr)

	client := NewClient()

	err := client.Connect(serverAddr)
	if err != nil {
		fmt.Printf("Connection failed: %v\n", err)
		fmt.Println("\nTroubleshooting:")
		fmt.Println("1. Make sure the server is running")
		fmt.Println("2. Check the server address and port")
		fmt.Println("3. Check firewall settings")
		return
	}

	fmt.Println("Starting game session...")
	fmt.Println("Type 'quit' anytime to exit")
	fmt.Println("═══════════════════════════════════════")

	client.Start()

	fmt.Println("\nDisconnected from server. Goodbye!")
}

// Additional helper functions for client

// validateCommand checks if user command is valid
func validateCommand(command string) bool {
	parts := strings.Split(strings.TrimSpace(command), " ")
	if len(parts) == 0 {
		return false
	}

	validCommands := map[string]bool{
		"help":   true,
		"status": true,
		"attack": true,
		"quit":   true,
		"exit":   true,
	}

	return validCommands[strings.ToLower(parts[0])]
}

// formatCommand formats user input for consistent server communication
func formatCommand(input string) string {
	return strings.TrimSpace(strings.ToLower(input))
}

// isAttackCommand checks if command is an attack command and validates syntax
func isAttackCommand(command string) (bool, error) {
	parts := strings.Split(strings.TrimSpace(command), " ")

	if len(parts) != 3 || strings.ToLower(parts[0]) != "attack" {
		return false, nil
	}

	// Validate troop index
	troopIdx := parts[1]
	if troopIdx != "1" && troopIdx != "2" && troopIdx != "3" {
		return false, fmt.Errorf("invalid troop index: use 1, 2, or 3")
	}

	// Validate target
	target := strings.ToLower(parts[2])
	validTargets := map[string]bool{
		"king":   true,
		"guard1": true,
		"guard2": true,
		"guard":  true,
	}

	if !validTargets[target] {
		return false, fmt.Errorf("invalid target: use king, guard1, or guard2")
	}

	return true, nil
}
