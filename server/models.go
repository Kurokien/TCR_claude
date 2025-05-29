// models.go
package main

import "time"

// Tower represents a defensive structure
type Tower struct {
	Type     string  `json:"type"`
	HP       float64 `json:"hp"`
	MaxHP    float64 `json:"max_hp"`
	ATK      float64 `json:"atk"`
	DEF      float64 `json:"def"`
	CRIT     float64 `json:"crit"`
	EXP      float64 `json:"exp"`
	Level    int     `json:"level"`
	Position string  `json:"position"`
}

// Troop represents an attacking unit
type Troop struct {
	Name    string  `json:"name"`
	HP      float64 `json:"hp"`
	MaxHP   float64 `json:"max_hp"`
	ATK     float64 `json:"atk"`
	DEF     float64 `json:"def"`
	MANA    float64 `json:"mana"`
	EXP     float64 `json:"exp"`
	Level   int     `json:"level"`
	Special string  `json:"special"`
}

// PlayerData stores all player information
type PlayerData struct {
	Username string            `json:"username"`
	Password string            `json:"password"`
	EXP      float64           `json:"exp"`
	Level    int               `json:"level"`
	Towers   map[string]*Tower `json:"towers"`
	Troops   []*Troop          `json:"troops"`
}

// GameState manages the current game session
type GameState struct {
	Player1       *PlayerData `json:"player1"`
	Player2       *PlayerData `json:"player2"`
	Player1Mana   float64     `json:"player1_mana"`
	Player2Mana   float64     `json:"player2_mana"`
	GameStartTime time.Time   `json:"game_start_time"`
	GameDuration  int         `json:"game_duration"` // seconds
	IsGameActive  bool        `json:"is_game_active"`
	Turn          int         `json:"turn"` // 1 for player1, 2 for player2
}

// Message types for network communication
type Message struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

type AttackMessage struct {
	TroopIndex int    `json:"troop_index"`
	Target     string `json:"target"`
}

type GameStatusMessage struct {
	PlayerMana     float64           `json:"player_mana"`
	OpponentMana   float64           `json:"opponent_mana"`
	TimeRemaining  float64           `json:"time_remaining"`
	PlayerTowers   map[string]*Tower `json:"player_towers"`
	OpponentTowers map[string]*Tower `json:"opponent_towers"`
	PlayerTroops   []*Troop          `json:"player_troops"`
}
