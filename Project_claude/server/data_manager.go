// data_manager.go
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
)

// PlayerStorage handles player data persistence
type PlayerStorage struct {
	Players map[string]*PlayerData `json:"players"`
}

// TroopTemplate defines troop specifications
type TroopTemplate struct {
	Name    string  `json:"name"`
	HP      float64 `json:"hp"`
	ATK     float64 `json:"atk"`
	DEF     float64 `json:"def"`
	MANA    float64 `json:"mana"`
	EXP     float64 `json:"exp"`
	Special string  `json:"special"`
}

// TowerTemplate defines tower specifications
type TowerTemplate struct {
	Type string  `json:"type"`
	HP   float64 `json:"hp"`
	ATK  float64 `json:"atk"`
	DEF  float64 `json:"def"`
	CRIT float64 `json:"crit"`
	EXP  float64 `json:"exp"`
}

// GameTemplates stores all game specifications
type GameTemplates struct {
	Troops []TroopTemplate `json:"troops"`
	Towers []TowerTemplate `json:"towers"`
}

// initializeDefaultData creates default JSON files if they don't exist
func initializeDefaultData() {
	// Initialize player data file
	if _, err := os.Stat("players.json"); os.IsNotExist(err) {
		createDefaultPlayersFile()
	}

	// Initialize game templates file
	if _, err := os.Stat("game_templates.json"); os.IsNotExist(err) {
		createDefaultTemplatesFile()
	}
}

// createDefaultPlayersFile creates the initial players.json
func createDefaultPlayersFile() {
	storage := &PlayerStorage{
		Players: make(map[string]*PlayerData),
	}

	// Create default test players
	player1 := createNewPlayer("player1", "password1")
	player2 := createNewPlayer("player2", "password2")

	storage.Players["player1"] = player1
	storage.Players["player2"] = player2

	savePlayerStorage(storage)
	fmt.Println("Created default players.json with test accounts")
}

// createDefaultTemplatesFile creates the game_templates.json
func createDefaultTemplatesFile() {
	templates := &GameTemplates{
		Troops: []TroopTemplate{
			{Name: "Pawn", HP: 50, ATK: 150, DEF: 100, MANA: 3, EXP: 5, Special: ""},
			{Name: "Bishop", HP: 100, ATK: 200, DEF: 150, MANA: 4, EXP: 10, Special: ""},
			{Name: "Rook", HP: 250, ATK: 200, DEF: 200, MANA: 5, EXP: 25, Special: ""},
			{Name: "Knight", HP: 200, ATK: 300, DEF: 150, MANA: 5, EXP: 25, Special: ""},
			{Name: "Prince", HP: 500, ATK: 400, DEF: 300, MANA: 6, EXP: 50, Special: ""},
			{Name: "Queen", HP: 99, ATK: 0, DEF: 0, MANA: 5, EXP: 30, Special: "Heal 300 to lowest HP tower"},
		},
		Towers: []TowerTemplate{
			{Type: "King Tower", HP: 2000, ATK: 500, DEF: 300, CRIT: 0.1, EXP: 200},
			{Type: "Guard Tower", HP: 1000, ATK: 300, DEF: 100, CRIT: 0.05, EXP: 100},
		},
	}

	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling templates: %v\n", err)
		return
	}

	err = os.WriteFile("game_templates.json", data, 0644)
	if err != nil {
		fmt.Printf("Error writing templates file: %v\n", err)
		return
	}

	fmt.Println("Created default game_templates.json")
}

// loadPlayerStorage loads all player data from JSON
func loadPlayerStorage() *PlayerStorage {
	data, err := os.ReadFile("players.json")
	if err != nil {
		fmt.Printf("Error reading players file: %v\n", err)
		return &PlayerStorage{Players: make(map[string]*PlayerData)}
	}

	var storage PlayerStorage
	err = json.Unmarshal(data, &storage)
	if err != nil {
		fmt.Printf("Error unmarshaling players: %v\n", err)
		return &PlayerStorage{Players: make(map[string]*PlayerData)}
	}

	return &storage
}

// savePlayerStorage saves all player data to JSON
func savePlayerStorage(storage *PlayerStorage) {
	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling player storage: %v\n", err)
		return
	}

	err = os.WriteFile("players.json", data, 0644)
	if err != nil {
		fmt.Printf("Error writing players file: %v\n", err)
	}
}

// loadGameTemplates loads troop and tower specifications
func loadGameTemplates() *GameTemplates {
	data, err := os.ReadFile("game_templates.json")
	if err != nil {
		fmt.Printf("Error reading templates file: %v\n", err)
		return nil
	}

	var templates GameTemplates
	err = json.Unmarshal(data, &templates)
	if err != nil {
		fmt.Printf("Error unmarshaling templates: %v\n", err)
		return nil
	}

	return &templates
}

// authenticatePlayer verifies player credentials and returns player data
func (s *Server) authenticatePlayer(username, password string) *PlayerData {
	storage := loadPlayerStorage()

	player, exists := storage.Players[username]
	if !exists {
		// Create new player if doesn't exist
		player = createNewPlayer(username, password)
		storage.Players[username] = player
		savePlayerStorage(storage)
		fmt.Printf("Created new player: %s\n", username)
		return player
	}

	if player.Password != password {
		return nil
	}

	return player
}

// createNewPlayer creates a new player with default stats
func createNewPlayer(username, password string) *PlayerData {
	templates := loadGameTemplates()
	if templates == nil {
		fmt.Println("Warning: Could not load game templates")
		return nil
	}

	player := &PlayerData{
		Username: username,
		Password: password,
		EXP:      0,
		Level:    1,
		Towers:   make(map[string]*Tower),
		Troops:   make([]*Troop, 0),
	}

	// Create towers
	for _, towerTemplate := range templates.Towers {
		var position string
		if towerTemplate.Type == "King Tower" {
			position = "king"
		} else if towerTemplate.Type == "Guard Tower" {
			// Create two guard towers
			for i := 1; i <= 2; i++ {
				pos := fmt.Sprintf("guard%d", i)
				tower := &Tower{
					Type:     towerTemplate.Type,
					HP:       towerTemplate.HP,
					MaxHP:    towerTemplate.HP,
					ATK:      towerTemplate.ATK,
					DEF:      towerTemplate.DEF,
					CRIT:     towerTemplate.CRIT,
					EXP:      towerTemplate.EXP,
					Level:    1,
					Position: pos,
				}
				player.Towers[pos] = tower
			}
			continue
		}

		if position != "" {
			tower := &Tower{
				Type:     towerTemplate.Type,
				HP:       towerTemplate.HP,
				MaxHP:    towerTemplate.HP,
				ATK:      towerTemplate.ATK,
				DEF:      towerTemplate.DEF,
				CRIT:     towerTemplate.CRIT,
				EXP:      towerTemplate.EXP,
				Level:    1,
				Position: position,
			}
			player.Towers[position] = tower
		}
	}

	// Randomly select 3 troops
	selectedTroops := make(map[int]bool)
	for len(player.Troops) < 3 {
		idx := rand.Intn(len(templates.Troops))
		if !selectedTroops[idx] {
			selectedTroops[idx] = true
			troopTemplate := templates.Troops[idx]

			troop := &Troop{
				Name:    troopTemplate.Name,
				HP:      troopTemplate.HP,
				MaxHP:   troopTemplate.HP,
				ATK:     troopTemplate.ATK,
				DEF:     troopTemplate.DEF,
				MANA:    troopTemplate.MANA,
				EXP:     troopTemplate.EXP,
				Level:   1,
				Special: troopTemplate.Special,
			}
			player.Troops = append(player.Troops, troop)
		}
	}

	return player
}

// loadPlayerData loads specific player data
func (s *Server) loadPlayerData(username string) *PlayerData {
	s.dataMux.RLock()
	defer s.dataMux.RUnlock()

	if player, exists := s.playerData[username]; exists {
		return player
	}

	// Load from file if not in memory
	storage := loadPlayerStorage()
	if player, exists := storage.Players[username]; exists {
		s.playerData[username] = player
		return player
	}

	return nil
}

// savePlayerData saves specific player data
func (s *Server) savePlayerData(username string, player *PlayerData) {
	s.dataMux.Lock()
	s.playerData[username] = player
	s.dataMux.Unlock()

	// Save to file
	storage := loadPlayerStorage()
	storage.Players[username] = player
	savePlayerStorage(storage)
}
