// game_logic.go
package main

import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

// displayGameState shows current game status to player
func (s *Server) displayGameState(conn net.Conn, playerNum int) {
	s.gameStateMux.RLock()
	defer s.gameStateMux.RUnlock()
	
	if s.gameState == nil {
		conn.Write([]byte("❌ Game not started yet.\n"))
		return
	}
	
	var player, opponent *PlayerData
	var playerMana, opponentMana float64
	
	if playerNum == 1 {
		player = s.gameState.Player1
		opponent = s.gameState.Player2
		playerMana = s.gameState.Player1Mana
		opponentMana = s.gameState.Player2Mana
	} else {
		player = s.gameState.Player2
		opponent = s.gameState.Player1
		playerMana = s.gameState.Player2Mana
		opponentMana = s.gameState.Player1Mana
	}
	
	output := fmt.Sprintf("\n=== 🎮 GAME STATUS 🎮 ===\n")
	output += fmt.Sprintf("💧 Your Mana: %.0f/10\n", playerMana)
	output += fmt.Sprintf("💧 Opponent Mana: %.0f/10\n", opponentMana)
	
	if s.gameState.IsGameActive {
		elapsed := time.Since(s.gameState.GameStartTime).Seconds()
		remaining := float64(s.gameState.GameDuration) - elapsed
		if remaining > 0 {
			output += fmt.Sprintf("⏰ Time Remaining: %.0f seconds\n", remaining)
		} else {
			output += fmt.Sprintf("⏰ Time: OVERTIME!\n")
		}
	}
	
	output += fmt.Sprintf("\n--- 🏰 YOUR TOWERS ---\n")
	for pos, tower := range player.Towers {
		status := "🟢 ALIVE"
		if tower.HP <= 0 {
			status = "💥 DESTROYED"
		}
		hpPercent := (tower.HP / tower.MaxHP) * 100
		output += fmt.Sprintf("%-12s: HP %.0f/%.0f (%.0f%%) [%s]\n", 
			tower.Type, tower.HP, tower.MaxHP, hpPercent, status)
	}
	
	output += fmt.Sprintf("\n--- 🏰 OPPONENT TOWERS ---\n")
	for pos, tower := range opponent.Towers {
		status := "🟢 ALIVE"
		if tower.HP <= 0 {
			status = "💥 DESTROYED"
		}
		hpPercent := (tower.HP / tower.MaxHP) * 100
		output += fmt.Sprintf("%-12s: HP %.0f/%.0f (%.0f%%) [%s]\n", 
			tower.Type, tower.HP, tower.MaxHP, hpPercent, status)
	}
	
	output += fmt.Sprintf("\n--- ⚔️ YOUR TROOPS ---\n")
	for i, troop := range player.Troops {
		output += fmt.Sprintf("%d. %-8s: HP %.0f, ATK %.0f, DEF %.0f, MANA %.0f\n", 
			i+1, troop.Name, troop.HP, troop.ATK, troop.DEF, troop.MANA)
		if troop.Special != "" {
			output += fmt.Sprintf("   ✨ Special: %s\n", troop.Special)
		}
	}
	
	output += fmt.Sprintf("\n========================\n")
	conn.Write([]byte(output))
}

// processAttack handles troop attacks
func (s *Server) processAttack(attackerConn net.Conn, playerNum int, troopIndex int, targetType string) {
	s.gameStateMux.Lock()
	defer s.gameStateMux.Unlock()
	
	if s.gameState == nil || !s.gameState.IsGameActive {
		attackerConn.Write([]byte("❌ Game not active.\n"))
		return
	}
	
	var attacker, defender *PlayerData
	var attackerMana *float64
	var attackerName, defenderName string
	
	if playerNum == 1 {
		attacker = s.gameState.Player1
		defender = s.gameState.Player2
		attackerMana = &s.gameState.Player1Mana
		attackerName = attacker.Username
		defenderName = defender.Username
	} else {
		attacker = s.gameState.Player2
		defender = s.gameState.Player1
		attackerMana = &s.gameState.Player2Mana
		attackerName = attacker.Username
		defenderName = defender.Username
	}
	
	if troopIndex < 0 || troopIndex >= len(attacker.Troops) {
		attackerConn.Write([]byte("❌ Invalid troop selection.\n"))
		return
	}
	
	troop := attacker.Troops[troopIndex]
	
	// Check mana
	if *attackerMana < troop.MANA {
		attackerConn.Write([]byte(fmt.Sprintf("❌ Not enough mana! Need %.0f, have %.0f\n", 
			troop.MANA, *attackerMana)))
		return
	}
	
	// Deduct mana
	*attackerMana -= troop.MANA
	
	// Handle special abilities
	if troop.Name == "Queen" {
		s.handleQueenSpecial(attackerConn, attacker, attackerName)
		return
	}
	
	// Find and validate target
	targetTower := s.findTargetTower(defender, targetType)
	if targetTower == nil {
		attackerConn.Write([]byte("❌ Invalid target or target already destroyed.\n"))
		return
	}
	
	// Validate attack rules
	if !s.canAttackTarget(defender, targetTower, attackerConn) {
		return
	}
	
	// Calculate and apply damage
	damage := calculateDamage(troop.ATK, targetTower.DEF, 0.05)
	targetTower.HP -= damage
	
	if targetTower.HP < 0 {
		targetTower.HP = 0
	}
	
	// Send attack results
	s.sendAttackResults(attackerConn, troop, targetTower, damage, attackerName, defenderName)
	
	// Check win conditions
	if targetTower.HP <= 0 {
		s.handleTowerDestruction(targetTower, playerNum, attackerName, defenderName)
	}
}

// handleQueenSpecial processes Queen's healing ability
func (s *Server) handleQueenSpecial(conn net.Conn, player *PlayerData, playerName string) {
	var lowestTower *Tower
	lowestHP := float64(99999)
	
	for _, tower := range player.Towers {
		if tower.HP > 0 && tower.HP < lowestHP {
			lowestHP = tower.HP
			lowestTower = tower
		}
	}
	
	if lowestTower != nil {
		healAmount := 300.0
		oldHP := lowestTower.HP
		lowestTower.HP += healAmount
		if lowestTower.HP > lowestTower.MaxHP {
			lowestTower.HP = lowestTower.MaxHP
		}
		
		actualHeal := lowestTower.HP - oldHP
		message := fmt.Sprintf("👑 Queen healed %s for %.0f HP! (%.0f -> %.0f)\n", 
			lowestTower.Type, actualHeal, oldHP, lowestTower.HP)
		conn.Write([]byte(message))
		
		s.broadcastToOthers(conn, fmt.Sprintf("🔮 %s's Queen healed their %s!\n", 
			playerName, lowestTower.Type))
	} else {
		conn.Write([]byte("👑 Queen found no towers to heal.\n"))
	}
}

// findTargetTower locates the target tower
func (s *Server) findTargetTower(defender *PlayerData, targetType string) *Tower {
	for pos, tower := range defender.Towers {
		if tower.HP <= 0 {
			continue
		}
		
		switch targetType {
		case "king":
			if pos == "king" {
				return tower
			}
		case "guard1":
			if pos == "guard1" {
				return tower
			}
		case "guard2":
			if pos == "guard2" {
				return tower
			}
		case "guard":
			if pos == "guard1" || pos == "guard2" {
				return tower
			}
		}
	}
	return nil
}

// canAttackTarget validates attack rules
func (s *Server) canAttackTarget(defender *PlayerData, target *Tower, conn net.Conn) bool {
	if target.Type == "King Tower" {
		// Check if any guard towers are still alive
		for pos, tower := range defender.Towers {
			if (pos == "guard1" || pos == "guard2") && tower.HP > 0 {
				conn.Write([]byte("❌ Must destroy all Guard Towers before attacking King Tower!\n"))
				return false
			}
		}
	}
	return true
}

// sendAttackResults notifies players of attack outcome
func (s *Server) sendAttackResults(attackerConn net.Conn, troop *Troop, target *Tower, 
	damage float64, attackerName, defenderName string) {
	
	message := fmt.Sprintf("⚔️ %s attacked %s for %.0f damage!\n", 
		troop.Name, target.Type, damage)
	message += fmt.Sprintf("🎯 Target HP: %.0f/%.0f\n", target.HP, target.MaxHP)
	
	attackerConn.Write([]byte(message))
	
	s.broadcastToOthers(attackerConn, 
		fmt.Sprintf("🚨 %s's %s attacked your %s for %.0f damage! HP: %.0f/%.0f\n", 
			attackerName, troop.Name, target.Type, damage, target.HP, target.MaxHP))
}

// handleTowerDestruction manages tower destruction and win conditions
func (s *Server) handleTowerDestruction(tower *Tower, winnerNum int, attackerName, defenderName string) {
	destructionMsg := fmt.Sprintf("💥 %s DESTROYED!\n", tower.Type)
	s.broadcastToAll(destructionMsg)
	
	if tower.Type == "King Tower" {
		s.endGame(winnerNum, fmt.Sprintf("👑 %s wins by destroying the King Tower!", attackerName))
	}
}

// calculateDamage computes damage with critical hit chance
func calculateDamage(atkStat, defStat, critChance float64) float64 {
	damage := atkStat
	
	// Apply critical hit
	if rand.Float64() < critChance {
		damage *= 1.2
	}
	
	// Apply defense
	damage = damage - defStat
	if damage < 0 {
		damage = 0
	}
	
	return damage
}

// startManaRegeneration begins mana regeneration system
func (s *Server) startManaRegeneration() {
	ticker := time.NewTicker(time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			s.gameStateMux.Lock()
			if s.gameState != nil && s.gameState.IsGameActive {
				if s.gameState.Player1Mana < 10 {
					s.gameState.Player1Mana++
				}
				if s.gameState.Player2Mana < 10 {
					s.gameState.Player2Mana++
				}
			} else {
				s.gameStateMux.Unlock()
				return
			}
			s.gameStateMux.Unlock()
		}
	}()
}

// startGameTimer manages game duration and timeout
func (s *Server) startGameTimer() {
	go func() {
		time.Sleep(time.Duration(s.gameState.GameDuration) * time.Second)
		
		s.gameStateMux.Lock()
		defer s.gameStateMux.Unlock()
		
		if s.gameState != nil && s.gameState.IsGameActive {
			s.handleGameTimeout()
		}
	}()
}

// handleGameTimeout processes game end by timeout
func (s *Server) handleGameTimeout() {
	// Count surviving towers
	p1Towers := 0
	p2Towers := 0
	
	for _, tower := range s.gameState.Player1.Towers {
		if tower.HP > 0 {
			p1Towers++
		}
	}
	
	for _, tower := range s.gameState.Player2.Towers {
		if tower.HP > 0 {
			p2Towers++
		}
	}
	
	if p1Towers > p2Towers {
		s.endGame(1, fmt.Sprintf("⏰ Time's up! %s wins with %d towers remaining!", 
			s.gameState.Player1.Username, p1Towers))
	} else if p2Towers > p1Towers {
		s.endGame(2, fmt.Sprintf("⏰ Time's up! %s wins with %d towers remaining!", 
			s.gameState.Player2.Username, p2Towers))
	} else {
		s.endGameDraw()
	}
}

// endGame handles game completion with winner
func (s *Server) endGame(winnerNum int, message string) {
	s.gameState.IsGameActive = false
	
	var winner, loser *PlayerData
	if winnerNum == 1 {
		winner = s.gameState.Player1
		loser = s.gameState.Player2
	} else {
		winner = s.gameState.Player2
		loser = s.gameState.Player1
	}
	
	// Award EXP
	winner.EXP += 30
	
	// Check for level ups
	s.checkLevelUp(winner)
	s.checkLevelUp(loser)
	
	// Save player data
	s.savePlayerData(winner.Username, winner)
	s.savePlayerData(loser.Username, loser)
	
	// Announce results
	s.broadcastToAll(fmt.Sprintf("\n🎉 GAME OVER! 🎉\n%s\n", message))
	s.broadcastToAll(fmt.Sprintf("🏆 %s gained 30 EXP!\n", winner.Username))
	s.broadcastToAll("Type 'quit' to leave or wait for next game.\n")
}

// endGameDraw handles draw games
func (s *Server) endGameDraw() {
	s.gameState.IsGameActive = false
	
	// Award EXP for draw
	s.gameState.Player1.EXP += 10
	s.gameState.Player2.EXP += 10
	
	s.checkLevelUp(s.gameState.Player1)
	s.checkLevelUp(s.gameState.Player2)
	
	s.savePlayerData(s.gameState.Player1.Username, s.gameState.Player1)
	s.savePlayerData(s.gameState.Player2.Username, s.gameState.Player2)
	
	s.broadcastToAll("\n🤝 GAME OVER - IT'S A DRAW! 🤝\n")
	s.broadcastToAll("Both players gained 10 EXP!\n")
	s.broadcastToAll("Type 'quit' to leave or wait for next game.\n")
}

// checkLevelUp handles player leveling system
func (s *Server) checkLevelUp(player *PlayerData) {
	requiredEXP := 100.0 * (1.1 * float64(player.Level))
	
	for player.EXP >= requiredEXP {
		player.EXP -= requiredEXP
		player.Level++
		
		// Increase stats by 10%
		for _, tower := range player.Towers {
			tower.HP *= 1.1
			tower.MaxHP *= 1.1
			tower.ATK *= 1.1
			tower.DEF *= 1.1
			tower.Level = player.Level
		}
		
		for _, troop := range player.Troops {
			troop.HP *= 1.1
			troop.MaxHP *= 1.1
			troop.ATK *= 1.1
			troop.DEF *= 1.1
			troop.Level = player.Level
		}
		
		s.broadcastToAll(fmt.Sprintf("🎊 %s leveled up to Level %d!\n", 
			player.Username, player.Level))
		
		requiredEXP = 100.0 * (1.1 * float64(player.Level))