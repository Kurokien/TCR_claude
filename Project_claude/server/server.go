// server.go
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Server manages client connections and game state
type Server struct {
	listener     net.Listener
	clients      map[string]net.Conn
	clientsMux   sync.RWMutex
	gameState    *GameState
	gameStateMux sync.RWMutex
	playerData   map[string]*PlayerData
	dataMux      sync.RWMutex
}

// NewServer creates a new server instance
func NewServer() *Server {
	return &Server{
		clients:    make(map[string]net.Conn),
		playerData: make(map[string]*PlayerData),
	}
}

// Start begins listening for client connections
func (s *Server) Start(port string) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	s.listener = listener
	fmt.Printf("TCR Server started on port %s\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go s.handleClient(conn)
	}
}

// handleClient manages individual client connections
func (s *Server) handleClient(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	var username string
	var playerNum int

	// Authentication phase
	conn.Write([]byte("=== Welcome to Text-Based Clash Royale! ===\n"))
	conn.Write([]byte("Enter username: "))

	if !scanner.Scan() {
		return
	}
	username = strings.TrimSpace(scanner.Text())

	conn.Write([]byte("Enter password: "))
	if !scanner.Scan() {
		return
	}
	password := strings.TrimSpace(scanner.Text())

	player := s.authenticatePlayer(username, password)
	if player == nil {
		conn.Write([]byte("Authentication failed!\n"))
		return
	}

	conn.Write([]byte(fmt.Sprintf("Welcome %s! Level: %d, EXP: %.0f\n",
		username, player.Level, player.EXP)))

	// Add to clients and determine player number
	s.clientsMux.Lock()
	s.clients[username] = conn
	clientCount := len(s.clients)

	if clientCount == 1 {
		playerNum = 1
		conn.Write([]byte("Waiting for opponent...\n"))
	} else if clientCount == 2 {
		playerNum = 2
		s.startNewGame()
	}
	s.clientsMux.Unlock()

	if clientCount > 2 {
		conn.Write([]byte("Game is full. Please wait for next round.\n"))
		return
	}

	// Show initial help
	s.sendHelp(conn)

	// Game command loop
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		s.processCommand(conn, playerNum, input)
	}

	// Clean up on disconnect
	s.removeClient(username)
}

// processCommand handles client commands
func (s *Server) processCommand(conn net.Conn, playerNum int, input string) {
	command := strings.ToLower(input)
	parts := strings.Split(command, " ")

	switch parts[0] {
	case "help":
		s.sendHelp(conn)

	case "status":
		s.displayGameState(conn, playerNum)

	case "attack":
		if len(parts) == 3 {
			troopIdx, err := strconv.Atoi(parts[1])
			target := strings.ToLower(parts[2])

			if err == nil && troopIdx >= 1 && troopIdx <= 3 {
				s.processAttack(conn, playerNum, troopIdx-1, target)
			} else {
				conn.Write([]byte("Invalid troop index. Use 1-3.\n"))
			}
		} else {
			conn.Write([]byte("Usage: attack <troop_index> <target>\n"))
			conn.Write([]byte("Example: attack 1 guard1\n"))
		}

	case "quit":
		conn.Write([]byte("Thanks for playing! Goodbye!\n"))
		return

	default:
		conn.Write([]byte("Unknown command. Type 'help' for available commands.\n"))
	}
}

// sendHelp displays available commands
func (s *Server) sendHelp(conn net.Conn) {
	help := `
=== TCR Commands ===
status          - Show current game state
attack <1-3> <target> - Attack with troop to target
                       Targets: king, guard1, guard2
quit            - Leave the game
help            - Show this help

Examples:
  attack 1 guard1   - Attack guard1 with your first troop
  attack 2 king     - Attack king tower with your second troop

Rules:
- Each troop costs mana to deploy
- Mana regenerates 1 per second (max 10)
- Must destroy guard towers before attacking king
- Game lasts 3 minutes
- Win by destroying king tower or more towers when time ends
====================
`
	conn.Write([]byte(help))
}

// removeClient handles client disconnection
func (s *Server) removeClient(username string) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	delete(s.clients, username)

	// If game was active, end it
	s.gameStateMux.Lock()
	if s.gameState != nil && s.gameState.IsGameActive {
		s.gameState.IsGameActive = false
		s.broadcastToAll("Game ended due to player disconnect.\n")
	}
	s.gameStateMux.Unlock()
}

// startNewGame initializes a new game session
func (s *Server) startNewGame() {
	usernames := make([]string, 0, 2)
	for username := range s.clients {
		usernames = append(usernames, username)
	}

	if len(usernames) != 2 {
		return
	}

	s.gameStateMux.Lock()
	s.gameState = &GameState{
		Player1:       s.loadPlayerData(usernames[0]),
		Player2:       s.loadPlayerData(usernames[1]),
		Player1Mana:   5,
		Player2Mana:   5,
		GameStartTime: time.Now(),
		GameDuration:  180, // 3 minutes
		IsGameActive:  true,
		Turn:          1,
	}
	s.gameStateMux.Unlock()

	// Reset towers HP to max
	s.resetTowersHP()

	s.broadcastToAll(fmt.Sprintf("🎮 GAME STARTED! 🎮\n"))
	s.broadcastToAll(fmt.Sprintf("Players: %s vs %s\n", usernames[0], usernames[1]))
	s.broadcastToAll("⏰ 3 minutes battle begins now!\n")
	s.broadcastToAll("Type 'status' to see current game state.\n")

	// Start background systems
	s.startManaRegeneration()
	s.startGameTimer()
}

// resetTowersHP resets all towers to full HP
func (s *Server) resetTowersHP() {
	s.gameStateMux.Lock()
	defer s.gameStateMux.Unlock()

	if s.gameState == nil {
		return
	}

	for _, tower := range s.gameState.Player1.Towers {
		tower.HP = tower.MaxHP
	}

	for _, tower := range s.gameState.Player2.Towers {
		tower.HP = tower.MaxHP
	}
}

// broadcastToAll sends message to all connected clients
func (s *Server) broadcastToAll(message string) {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()

	for _, conn := range s.clients {
		conn.Write([]byte(message))
	}
}

// broadcastToOthers sends message to all clients except sender
func (s *Server) broadcastToOthers(sender net.Conn, message string) {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()

	for _, conn := range s.clients {
		if conn != sender {
			conn.Write([]byte(message))
		}
	}
}
