package main

import (
	"log"

	"hexagonal-queue/internal/application/usecases"
	"hexagonal-queue/internal/infrastructure/adapters/db"
	"hexagonal-queue/internal/infrastructure/adapters/queue"
	"hexagonal-queue/internal/infrastructure/config"
	"hexagonal-queue/internal/infrastructure/web"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	dbAdapter, err := db.NewMongoAdapter(cfg.Database.MongoDB.URI, cfg.Database.MongoDB.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbAdapter.Close()

	// Initialize queue adapter
	queueFactory := queue.NewQueueFactory(cfg)
	queueAdapter, err := queueFactory.CreateQueueAdapter(cfg.Queue.Provider)
	if err != nil {
		log.Fatalf("Failed to initialize queue adapter: %v", err)
	}
	defer queueAdapter.Close()

	// Initialize use cases
	walletUseCase := usecases.NewWalletUseCase(dbAdapter, queueAdapter)

	// Start web server (publisher only - no consumer)
	webServer := web.NewServer(cfg.Server.Port, walletUseCase, queueAdapter)

	log.Printf("🚀 Starting publisher server on port %s with queue provider: %s", cfg.Server.Port, cfg.Queue.Provider)
	if err := webServer.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
