# Makefile for Text-Based Clash Royale (TCR)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
SERVER_BINARY=tcr-server
CLIENT_BINARY=tcr-client

# Server files
SERVER_SOURCES=main.go server.go game_logic.go data_manager.go models.go

# Client files  
CLIENT_SOURCES=client.go

.PHONY: all build clean run-server run-client test help setup

# Default target
all: build

# Build both server and client
build: build-server build-client

# Build server
build-server:
	@echo "Building TCR Server..."
	$(GOBUILD) -o $(SERVER_BINARY) $(SERVER_SOURCES)
	@echo "Server built successfully: $(SERVER_BINARY)"

# Build client
build-client:
	@echo "Building TCR Client..."
	$(GOBUILD) -o $(CLIENT_BINARY) $(CLIENT_SOURCES)
	@echo "Client built successfully: $(CLIENT_BINARY)"

# Run server on default port 8080
run-server: build-server
	@echo "Starting TCR Server on port 8080..."
	./$(SERVER_BINARY)

# Run server on custom port
run-server-port: build-server
	@echo "Starting TCR Server on port $(PORT)..."
	./$(SERVER_BINARY) $(PORT)

# Run client connecting to localhost:8080
run-client: build-client
	@echo "Starting TCR Client..."
	./$(CLIENT_BINARY)

# Run client connecting to custom server
run-client-custom: build-client
	@echo "Starting TCR Client connecting to $(SERVER)..."
	./$(CLIENT_BINARY) $(SERVER)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -f $(SERVER_BINARY)
	rm -f $(CLIENT_BINARY)
	@echo "Clean completed"

# Initialize Go modules (if needed)
mod-init:
	$(GOMOD) init tcr-game

# Download dependencies
mod-tidy:
	$(GOMOD) tidy

# Setup project (initialize modules and create data files)
setup: mod-tidy
	@echo "Setting up TCR project..."
	@echo "Creating data directories..."
	@mkdir -p data
	@echo "Building initial setup..."
	$(GOBUILD) -o setup-temp $(SERVER_SOURCES)
	@echo "Running initial setup to create JSON files..."
	@timeout 2s ./setup-temp || true
	@rm -f setup-temp
	@echo "Setup completed!"

# Test the application
test:
	$(GOTEST) -v ./...

# Run development setup (builds and runs server in background, then client)
dev: build
	@echo "Starting development environment..."
	@echo "Starting server in background..."
	@./$(SERVER_BINARY) &
	@echo "Server PID: $$!"
	@sleep 2
	@echo "Starting client..."
	@./$(CLIENT_BINARY)

# Kill all running TCR processes
kill:
	@echo "Stopping all TCR processes..."
	@pkill -f $(SERVER_BINARY) || true
	@pkill -f $(CLIENT_BINARY) || true
	@echo "All processes stopped"

# Install dependencies (if any external packages needed)
deps:
	@echo "Installing dependencies..."
	$(GOGET) -u all

# Show help
help:
	@echo "TCR - Text-Based Clash Royale Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  build          - Build both server and client"
	@echo "  build-server   - Build server only"
	@echo "  build-client   - Build client only"
	@echo "  run-server     - Build and run server on port 8080"
	@echo "  run-client     - Build and run client"
	@echo "  setup          - Initialize project and create data files"
	@echo "  clean          - Remove build artifacts"
	@echo "  test           - Run tests"
	@echo "  dev            - Start development environment"
	@echo "  kill           - Stop all TCR processes"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Custom usage:"
	@echo "  make run-server-port PORT=9090"
	@echo "  make run-client-custom SERVER=192.168.1.100:8080"
	@echo ""
	@echo "Quick start:"
	@echo "  1. make setup"
	@echo "  2. make run-server (in one terminal)"
	@echo "  3. make run-client (in another terminal)"

# Create distribution package
dist: clean build
	@echo "Creating distribution package..."
	@mkdir -p dist/tcr-game
	@cp $(SERVER_BINARY) dist/tcr-game/
	@cp $(CLIENT_BINARY) dist/tcr-game/
	@cp README.md dist/tcr-game/ 2>/dev/null || true
	@cp *.json dist/tcr-game/ 2>/dev/null || true
	@cd dist && tar -czf tcr-game.tar.gz tcr-game/
	@echo "Distribution package created: dist/tcr-game.tar.gz"

# Development tools
lint:
	@echo "Running Go lint..."
	@gofmt -l *.go

format:
	@echo "Formatting Go code..."
	@gofmt -w *.go

# Quick development cycle
dev-cycle: clean build run-server

# Create project structure
init-project:
	@echo "Initializing TCR project structure..."
	@mkdir -p {docs,data,logs,scripts}
	@touch docs/architecture.md
	@touch docs/api.md
	@touch logs/.gitkeep
	@echo "Project structure created"

# Run with race detection (for development)
run-server-race: 
	$(GOBUILD) -race -o $(SERVER_BINARY)-race $(SERVER_SOURCES)
	./$(SERVER_BINARY)-race

run-client-race:
	$(GOBUILD) -race -o $(CLIENT_BINARY)-race $(CLIENT_SOURCES)
	./$(CLIENT_BINARY)-race