package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"hexagonal-queue/internal/application/usecases"
	"hexagonal-queue/internal/infrastructure/adapters/db"
	"hexagonal-queue/internal/infrastructure/adapters/queue"
	"hexagonal-queue/internal/infrastructure/config"

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

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start consumer in goroutine
	go func() {
		log.Printf("🚀 Starting %s consumer...", cfg.Queue.Provider)
		if err := queueAdapter.StartConsumer(walletUseCase); err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("🛑 Shutdown signal received, stopping consumer...")
	log.Println("✅ Consumer stopped gracefully")
}
