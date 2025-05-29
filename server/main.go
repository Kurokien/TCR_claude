// main.go
package main

import (
	"log"
	"math/rand"
	"os"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// Initialize default data files
	initializeDefaultData()

	server := NewServer()

	port := "8080"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	log.Printf("Starting TCR Server on port %s...", port)
	if err := server.Start(port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
