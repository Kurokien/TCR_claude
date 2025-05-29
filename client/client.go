// client.go - Simple clean version
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
			// Xử lý message đặc biệt
			if strings.Contains(message, "Enter username:") {
				fmt.Print(message)
			} else if strings.Contains(message, "Enter password:") {
				fmt.Print(message)
			} else if strings.Contains(message, "YOUR TURN") {
				fmt.Printf("\n🔔 %s\n", message)
			} else if strings.Contains(message, "'s TURN") {
				fmt.Printf("\n⏳ %s\n", message)
			} else {
				// In message bình thường
				fmt.Print(message)
				if !strings.HasSuffix(message, "\n") {
					fmt.Println()
				}
			}
		}
		c.mu.Unlock()
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

		// Gửi input trực tiếp đến server
		if input != "" {
			_, err = c.conn.Write([]byte(input + "\n"))
			if err != nil {
				fmt.Printf("Error sending command: %v\n", err)
				break
			}
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
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║          Text-Based Clash Royale Client          ║")
	fmt.Println("║                   TCR v2.0                       ║")
	fmt.Println("║                Turn-Based Edition                ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()
}

// main function for client
func main() {
	printWelcome()

	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		fmt.Println("Usage: go run client.go [server_address]")
		fmt.Println("Default server address: localhost:8080")
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
		return
	}

	fmt.Println("Starting game session...")
	fmt.Println("Type 'quit' anytime to exit")
	fmt.Println("═══════════════════════════════════════════════════")

	client.Start()

	fmt.Println("\nDisconnected from server. Thanks for playing!")
}
