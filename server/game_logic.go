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
		output += fmt.Sprintf("%-12s (%s): HP %.0f/%.0f (%.0f%%) [%s]\n",
			tower.Type, pos, tower.HP, tower.MaxHP, hpPercent, status)
	}

	output += fmt.Sprintf("\n--- 🏰 OPPONENT TOWERS ---\n")
	for pos, tower := range opponent.Towers {
		status := "🟢 ALIVE"
		if tower.HP <= 0 {
			status = "💥 DESTROYED"
		}
		hpPercent := (tower.HP / tower.MaxHP) * 100
		output += fmt.Sprintf("%-12s (%s): HP %.0f/%.0f (%.0f%%) [%s]\n",
			tower.Type, pos, tower.HP, tower.MaxHP, hpPercent, status)
	}

	output += fmt.Sprintf("\n--- ⚔️ YOUR TROOPS ---\n")
	for i, troop := range player.Troops {
		levelBonus := ""
		if troop.Level > 1 {
			levelBonus = fmt.Sprintf(" (Lv.%d +%.0f%%)", troop.Level, float64(troop.Level-1)*10)
		}
		output += fmt.Sprintf("%d. %-8s: HP %.0f, ATK %.0f, DEF %.0f, MANA %.0f%s\n",
			i+1, troop.Name, troop.HP, troop.ATK, troop.DEF, troop.MANA, levelBonus)
		if troop.Special != "" {
			output += fmt.Sprintf("   ✨ Special: %s\n", troop.Special)
		}
	}

	// Display tower destruction requirements
	output += fmt.Sprintf("\n--- 🎯 ATTACK RULES ---\n")
	guardTowersAlive := 0
	for pos, tower := range opponent.Towers {
		if (pos == "guard1" || pos == "guard2") && tower.HP > 0 {
			guardTowersAlive++
		}
	}

	if guardTowersAlive > 0 {
		output += fmt.Sprintf("⚠️  Must destroy all Guard Towers (%d remaining) before attacking King Tower!\n", guardTowersAlive)
	} else {
		output += fmt.Sprintf("✅ King Tower is now vulnerable to attack!\n")
	}

	output += fmt.Sprintf("\n========================\n")
	conn.Write([]byte(output))
}

// processAttack handles troop attacks with enhanced logic
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
	var defenderConn net.Conn

	if playerNum == 1 {
		attacker = s.gameState.Player1
		defender = s.gameState.Player2
		attackerMana = &s.gameState.Player1Mana
		attackerName = attacker.Username
		defenderName = defender.Username
		// Get defender connection
		s.clientsMux.RLock()
		defenderConn = s.clients[defenderName]
		s.clientsMux.RUnlock()
	} else {
		attacker = s.gameState.Player2
		defender = s.gameState.Player1
		attackerMana = &s.gameState.Player2Mana
		attackerName = attacker.Username
		defenderName = defender.Username
		// Get defender connection
		s.clientsMux.RLock()
		defenderConn = s.clients[defenderName]
		s.clientsMux.RUnlock()
	}

	if troopIndex < 0 || troopIndex >= len(attacker.Troops) {
		attackerConn.Write([]byte("❌ Invalid troop selection.\n"))
		return
	}

	troop := attacker.Troops[troopIndex]

	// Check mana requirement
	if *attackerMana < troop.MANA {
		attackerConn.Write([]byte(fmt.Sprintf("❌ Not enough mana! Need %.0f, have %.0f\n",
			troop.MANA, *attackerMana)))
		return
	}

	// Deduct mana
	*attackerMana -= troop.MANA

	// Handle special abilities (Queen healing)
	if troop.Name == "Queen" {
		s.handleQueenSpecial(attackerConn, attacker, attackerName, defenderConn)
		return
	}

	// Find and validate target
	targetTower := s.findTargetTower(defender, targetType)
	if targetTower == nil {
		attackerConn.Write([]byte("❌ Invalid target or target already destroyed.\n"))
		// Refund mana since attack failed
		*attackerMana += troop.MANA
		return
	}

	// Validate attack rules (Simple TCR Rule: must destroy Guard Towers first)
	if !s.canAttackTarget(defender, targetTower, attackerConn) {
		// Refund mana since attack failed
		*attackerMana += troop.MANA
		return
	}

	// Calculate damage with enhanced logic
	damage := s.calculateEnhancedDamage(troop, targetTower)
	targetTower.HP -= damage

	if targetTower.HP < 0 {
		targetTower.HP = 0
	}

	// Send attack results to both players
	s.sendAttackResults(attackerConn, defenderConn, troop, targetTower,
		damage, attackerName, defenderName)

	// Check if tower was destroyed and handle continuous attack
	if targetTower.HP <= 0 {
		destroyed := s.handleTowerDestruction(targetTower, playerNum, attackerName, defenderName)
		if destroyed {
			// If game didn't end, allow troop to continue attacking (Simple TCR Rule)
			if s.gameState != nil && s.gameState.IsGameActive {
				s.handleContinuousAttack(attackerConn, playerNum, troopIndex,
					attackerName, defenderName)
			}
		}
	}

	// Tower counter-attack if still alive
	if targetTower.HP > 0 {
		s.handleTowerCounterAttack(attackerConn, defenderConn, targetTower,
			troop, attackerName, defenderName)
	}
}

// calculateEnhancedDamage implements Enhanced TCR damage formula with CRIT
func (s *Server) calculateEnhancedDamage(attacker *Troop, target *Tower) float64 {
	baseDamage := attacker.ATK

	// Enhanced TCR: Apply critical hit chance
	// Default 5% crit chance for troops, towers have their own crit values
	critChance := 0.05
	isCritical := rand.Float64() < critChance

	if isCritical {
		baseDamage *= 1.2 // 20% bonus damage on crit
	}

	// Apply defense reduction
	finalDamage := baseDamage - target.DEF
	if finalDamage < 0 {
		finalDamage = 0
	}

	return finalDamage
}

// handleContinuousAttack allows troop to continue attacking after destroying a tower
func (s *Server) handleContinuousAttack(attackerConn net.Conn, playerNum int,
	troopIndex int, attackerName, defenderName string) {

	var defender *PlayerData
	if playerNum == 1 {
		defender = s.gameState.Player2
	} else {
		defender = s.gameState.Player1
	}

	// Check if there are more valid targets
	availableTargets := s.getAvailableTargets(defender)
	if len(availableTargets) == 0 {
		return // No more targets
	}

	attackerConn.Write([]byte(fmt.Sprintf("🔥 %s can continue attacking! Available targets: %s\n",
		s.gameState.Player1.Troops[troopIndex].Name,
		fmt.Sprintf("%v", availableTargets))))
	attackerConn.Write([]byte("Choose next target (or type 'skip' to end turn): "))
}

// getAvailableTargets returns list of available attack targets
func (s *Server) getAvailableTargets(defender *PlayerData) []string {
	var targets []string

	// Check guard towers first
	for pos, tower := range defender.Towers {
		if tower.HP > 0 {
			if pos == "guard1" || pos == "guard2" {
				targets = append(targets, pos)
			}
		}
	}

	// If no guard towers, king tower is available
	if len(targets) == 0 {
		if kingTower, exists := defender.Towers["king"]; exists && kingTower.HP > 0 {
			targets = append(targets, "king")
		}
	}

	return targets
}

// handleTowerCounterAttack implements tower counter-attack mechanism
func (s *Server) handleTowerCounterAttack(attackerConn, defenderConn net.Conn,
	tower *Tower, attackingTroop *Troop, attackerName, defenderName string) {

	// Tower counter-attacks with its own crit chance
	damage := tower.ATK
	isCritical := rand.Float64() < tower.CRIT

	if isCritical {
		damage *= 1.2
	}

	// Apply troop defense
	finalDamage := damage - attackingTroop.DEF
	if finalDamage < 0 {
		finalDamage = 0
	}

	attackingTroop.HP -= finalDamage
	if attackingTroop.HP < 0 {
		attackingTroop.HP = 0
	}

	// Notify both players
	counterMsg := fmt.Sprintf("🏰 %s counter-attacks %s for %.0f damage!",
		tower.Type, attackingTroop.Name, finalDamage)
	if isCritical {
		counterMsg += " 💥 CRITICAL HIT!"
	}
	counterMsg += fmt.Sprintf(" (%s HP: %.0f/%.0f)\n",
		attackingTroop.Name, attackingTroop.HP, attackingTroop.MaxHP)

	if attackerConn != nil {
		attackerConn.Write([]byte(counterMsg))
	}
	if defenderConn != nil {
		defenderConn.Write([]byte(fmt.Sprintf("🛡️ Your %s counter-attacked! %s",
			tower.Type, counterMsg)))
	}

	// Check if troop was defeated
	if attackingTroop.HP <= 0 {
		defeatMsg := fmt.Sprintf("💀 %s's %s was defeated!\n", attackerName, attackingTroop.Name)
		if attackerConn != nil {
			attackerConn.Write([]byte(defeatMsg))
		}
		if defenderConn != nil {
			defenderConn.Write([]byte(defeatMsg))
		}
	}
}

// handleQueenSpecial processes Queen's healing ability with enhanced feedback
func (s *Server) handleQueenSpecial(conn net.Conn, player *PlayerData, playerName string, opponentConn net.Conn) {
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

		if opponentConn != nil {
			opponentConn.Write([]byte(fmt.Sprintf("🔮 %s's Queen healed their %s for %.0f HP!\n",
				playerName, lowestTower.Type, actualHeal)))
		}
	} else {
		conn.Write([]byte("👑 Queen found no damaged towers to heal.\n"))
	}
}

// findTargetTower locates the target tower with improved logic
func (s *Server) findTargetTower(defender *PlayerData, targetType string) *Tower {
	for pos, tower := range defender.Towers {
		if tower.HP <= 0 {
			continue // Skip destroyed towers
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
			// Target any available guard tower
			if pos == "guard1" || pos == "guard2" {
				return tower
			}
		}
	}
	return nil
}

// canAttackTarget validates attack rules (Simple TCR Rule implementation)
func (s *Server) canAttackTarget(defender *PlayerData, target *Tower, conn net.Conn) bool {
	if target.Position == "king" {
		// Simple TCR Rule: Must destroy Guard Towers before attacking King Tower
		guardTowersAlive := 0
		for pos, tower := range defender.Towers {
			if (pos == "guard1" || pos == "guard2") && tower.HP > 0 {
				guardTowersAlive++
			}
		}

		if guardTowersAlive > 0 {
			conn.Write([]byte(fmt.Sprintf("❌ Must destroy all Guard Towers (%d remaining) before attacking King Tower!\n",
				guardTowersAlive)))
			return false
		}
	}
	return true
}

// sendAttackResults notifies players of attack outcome with enhanced details
func (s *Server) sendAttackResults(attackerConn, defenderConn net.Conn, troop *Troop,
	target *Tower, damage float64, attackerName, defenderName string) {

	attackerMsg := fmt.Sprintf("⚔️ %s attacked %s for %.0f damage!\n",
		troop.Name, target.Type, damage)
	attackerMsg += fmt.Sprintf("🎯 Target HP: %.0f/%.0f (%.1f%%)\n",
		target.HP, target.MaxHP, (target.HP/target.MaxHP)*100)

	defenderMsg := fmt.Sprintf("🚨 %s's %s attacked your %s for %.0f damage!\n",
		attackerName, troop.Name, target.Type, damage)
	defenderMsg += fmt.Sprintf("🏰 %s HP: %.0f/%.0f (%.1f%%)\n",
		target.Type, target.HP, target.MaxHP, (target.HP/target.MaxHP)*100)

	if attackerConn != nil {
		attackerConn.Write([]byte(attackerMsg))
	}
	if defenderConn != nil {
		defenderConn.Write([]byte(defenderMsg))
	}
}

// handleTowerDestruction manages tower destruction and win conditions
func (s *Server) handleTowerDestruction(tower *Tower, winnerNum int, attackerName, defenderName string) bool {
	destructionMsg := fmt.Sprintf("💥 %s (%s) DESTROYED!\n", tower.Type, tower.Position)
	s.broadcastToAll(destructionMsg)

	// Award EXP for tower destruction
	var winner *PlayerData
	if winnerNum == 1 {
		winner = s.gameState.Player1
	} else {
		winner = s.gameState.Player2
	}
	winner.EXP += tower.EXP

	s.broadcastToAll(fmt.Sprintf("🏆 %s gained %.0f EXP for destroying %s!\n",
		attackerName, tower.EXP, tower.Type))

	// Check for immediate win condition (King Tower destroyed)
	if tower.Type == "King Tower" {
		s.endGame(winnerNum, fmt.Sprintf("👑 %s wins by destroying the King Tower!", attackerName))
		return true
	}

	return true
}

// startManaRegeneration begins mana regeneration system (Enhanced TCR: 1 mana/sec)
func (s *Server) startManaRegeneration() {
	ticker := time.NewTicker(time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			s.gameStateMux.Lock()
			if s.gameState != nil && s.gameState.IsGameActive {
				// Enhanced TCR: Regenerate 1 mana per second, max 10
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

// startGameTimer manages game duration (Enhanced TCR: 3 minutes)
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

// handleGameTimeout processes game end by timeout (Enhanced TCR win conditions)
func (s *Server) handleGameTimeout() {
	// Count surviving towers for each player
	p1Towers := 0
	p2Towers := 0
	p1KingAlive := false
	p2KingAlive := false

	for pos, tower := range s.gameState.Player1.Towers {
		if tower.HP > 0 {
			p1Towers++
			if pos == "king" {
				p1KingAlive = true
			}
		}
	}

	for pos, tower := range s.gameState.Player2.Towers {
		if tower.HP > 0 {
			p2Towers++
			if pos == "king" {
				p2KingAlive = true
			}
		}
	}

	// Enhanced TCR Win Conditions
	if !p1KingAlive && p2KingAlive {
		s.endGame(2, fmt.Sprintf("⏰ Time's up! %s wins - %s's King Tower was destroyed!",
			s.gameState.Player2.Username, s.gameState.Player1.Username))
	} else if !p2KingAlive && p1KingAlive {
		s.endGame(1, fmt.Sprintf("⏰ Time's up! %s wins - %s's King Tower was destroyed!",
			s.gameState.Player1.Username, s.gameState.Player2.Username))
	} else if p1Towers > p2Towers {
		s.endGame(1, fmt.Sprintf("⏰ Time's up! %s wins with %d towers remaining! (%d vs %d)",
			s.gameState.Player1.Username, p1Towers, p1Towers, p2Towers))
	} else if p2Towers > p1Towers {
		s.endGame(2, fmt.Sprintf("⏰ Time's up! %s wins with %d towers remaining! (%d vs %d)",
			s.gameState.Player2.Username, p2Towers, p2Towers, p1Towers))
	} else {
		s.endGameDraw()
	}
}

// endGame handles game completion with winner (Enhanced TCR EXP System)
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

	// Enhanced TCR EXP System: Winner gets 30 EXP
	winner.EXP += 30

	// Check for level ups
	s.checkLevelUp(winner)
	s.checkLevelUp(loser)

	// Save player data
	s.savePlayerData(winner.Username, winner)
	s.savePlayerData(loser.Username, loser)

	// Announce results
	s.broadcastToAll(fmt.Sprintf("\n🎉 GAME OVER! 🎉\n%s\n", message))
	s.broadcastToAll(fmt.Sprintf("🏆 %s gained 30 EXP for winning!\n", winner.Username))
	s.broadcastToAll("Type 'quit' to leave or wait for next game.\n")
}

// endGameDraw handles draw games (Enhanced TCR EXP System)
func (s *Server) endGameDraw() {
	s.gameState.IsGameActive = false

	// Enhanced TCR EXP System: Draw gives 10 EXP each
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

// checkLevelUp handles player leveling system (Enhanced TCR Leveling System)
func (s *Server) checkLevelUp(player *PlayerData) {
	// Enhanced TCR: Required EXP increases by 10% each level
	requiredEXP := 100.0 * (1.1 * float64(player.Level))

	for player.EXP >= requiredEXP {
		player.EXP -= requiredEXP
		player.Level++

		// Enhanced TCR: EXP increases troop/tower stats by 10% per level
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

		s.broadcastToAll(fmt.Sprintf("🎊 %s leveled up to Level %d! All stats increased by 10%%!\n",
			player.Username, player.Level))

		requiredEXP = 100.0 * (1.1 * float64(player.Level))
	}
}

// Additional helper functions for enhanced gameplay

// getTowerCount returns the number of remaining towers for a player
func (s *Server) getTowerCount(player *PlayerData) int {
	count := 0
	for _, tower := range player.Towers {
		if tower.HP > 0 {
			count++
		}
	}
	return count
}

// isKingTowerDestroyed checks if king tower is destroyed
func (s *Server) isKingTowerDestroyed(player *PlayerData) bool {
	if kingTower, exists := player.Towers["king"]; exists {
		return kingTower.HP <= 0
	}
	return false
}

// displayWinConditions shows current win conditions to players
func (s *Server) displayWinConditions(conn net.Conn) {
	conditions := `
=== 🏆 WIN CONDITIONS 🏆 ===
1. Destroy opponent's King Tower (Instant Win)
2. When time runs out (3 minutes):
   - Player with more towers remaining wins
   - If equal towers: DRAW

=== 📋 GAME RULES 📋 ===
• Must destroy Guard Towers before King Tower
• Mana regenerates 1 per second (max 10)
• Each troop costs mana to deploy
• Towers can counter-attack
• Critical hits deal 20% bonus damage
• Queen heals lowest HP tower for 300

=== 💎 EXP REWARDS 💎 ===
• Win: 30 EXP
• Draw: 10 EXP each
• Tower destroyed: EXP based on tower type
• Level up: +10% to all stats
=============================
`
	conn.Write([]byte(conditions))
}
