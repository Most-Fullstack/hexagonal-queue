.PHONY: help build run test clean deps docker-up docker-down docker-build

# Variables
APP_NAME=hexagonal-queue
BINARY_NAME=hexagonal-queue
DOCKER_IMAGE=hexagonal-queue:latest

# Default target
help:
	@echo "Hexagonal Queue - Wallet Service"
	@echo ""
	@echo "Available commands:"
	@echo "  build           - Build the Go application"
	@echo "  run             - Run the application locally"
	@echo "  run-rabbitmq    - Run locally with RabbitMQ"
	@echo "  run-kafka       - Run locally with Kafka"  
	@echo "  run-redis       - Run locally with Redis"
	@echo ""
	@echo "Docker commands:"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-up       - Start all services (including wallet service)"
	@echo "  docker-up-infra - Start only infrastructure services"
	@echo "  docker-down     - Stop all services"
	@echo "  docker-logs     - Show logs from all services"
	@echo "  docker-clean    - Clean up Docker resources"
	@echo ""
	@echo "Development:"
	@echo "  deps            - Download Go dependencies"
	@echo "  test            - Run tests"
	@echo "  clean           - Clean build artifacts"
	@echo "  dev-setup       - Set up development environment"

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	go build -o bin/$(BINARY_NAME) cmd/main.go

# Run the application locally
run:
	@echo "Running $(APP_NAME) locally..."
	go run cmd/main.go

# Run with specific queue providers
run-rabbitmq:
	@echo "Running with RabbitMQ..."
	QUEUE_PROVIDER=rabbitmq go run cmd/main.go

run-kafka:
	@echo "Running with Kafka..."
	QUEUE_PROVIDER=kafka go run cmd/main.go

run-redis:
	@echo "Running with Redis..."
	QUEUE_PROVIDER=redis go run cmd/main.go

# Docker commands
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

docker-up: docker-build
	@echo "Starting all services..."
	docker-compose up -d

docker-up-infra:
	@echo "Starting infrastructure services only..."
	docker-compose up -d mongodb rabbitmq kafka zookeeper redis

docker-down:
	@echo "Stopping all services..."
	docker-compose down

docker-logs:
	@echo "Showing logs..."
	docker-compose logs -f

docker-clean:
	@echo "Cleaning up Docker resources..."
	docker-compose down -v
	docker system prune -f

# Development setup
dev-setup: docker-up-infra
	@echo "Setting up development environment..."
	@echo "Waiting for services to start..."
	@sleep 15
	@echo ""
	@echo "=== Development Environment Ready ==="
	@echo "MongoDB:    mongodb://admin:password123@localhost:27017/wallet_db?authSource=admin"
	@echo "RabbitMQ:   amqp://admin:password123@localhost:5672/ (UI: http://localhost:15672)"
	@echo "Kafka:      localhost:9092"
	@echo "Redis:      localhost:6379 (password: password123)"
	@echo ""
	@echo "Use 'make run', 'make run-kafka', or 'make run-redis' to start the application"

# Go commands
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

test:
	@echo "Running tests..."
	go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# API testing (requires services to be running)
test-health:
	@echo "Testing health endpoint..."
	curl -s http://localhost:8080/health | jq '.' || echo "Service not running or jq not installed"

test-deposit:
	@echo "Creating test user and depositing money..."
	@curl -X POST http://localhost:8080/api/v1/wallet/member \
		-H "Content-Type: application/json" \
		-H "Authorization: test-parent-token" \
		-d '{"username":"testuser","token":"test-token"}' 2>/dev/null || true
	@echo ""
	@curl -X POST http://localhost:8080/api/v1/wallet/deposit \
		-H "Content-Type: application/json" \
		-H "Authorization: test-parent-token" \
		-d '{"username":"testuser","token":"test-token","type_name":"DEPOSIT","amount":100.0,"channel":"api","description":"Test deposit"}' | jq '.' || echo "Test failed or jq not installed"

test-balance:
	@echo "Checking balance..."
	curl -X POST http://localhost:8080/api/v1/wallet/balance \
		-H "Content-Type: application/json" \
		-H "Authorization: test-parent-token" \
		-d '{"username":"testuser"}' | jq '.' || echo "Test failed or jq not installed"

# Queue provider testing
test-rabbitmq:
	@echo "Testing RabbitMQ queue..."
	curl -X POST http://localhost:8080/api/v1/test/queue \
		-H "Content-Type: application/json" \
		-d '{"provider":"rabbitmq","username":"testuser","action":"deposit","amount":50.0}' | jq '.' || echo "Test failed"

test-kafka:
	@echo "Testing Kafka queue..."
	curl -X POST http://localhost:8080/api/v1/test/queue \
		-H "Content-Type: application/json" \
		-d '{"provider":"kafka","username":"testuser","action":"deposit","amount":50.0}' | jq '.' || echo "Test failed"

test-redis-queue:
	@echo "Testing Redis queue..."
	curl -X POST http://localhost:8080/api/v1/test/queue \
		-H "Content-Type: application/json" \
		-d '{"provider":"redis","username":"testuser","action":"deposit","amount":50.0}' | jq '.' || echo "Test failed"

# Service status
status:
	@echo "Checking service status..."
	docker-compose ps