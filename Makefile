.PHONY: help build run test clean deps docker-up docker-down docker-build

# Variables
APP_NAME=hexagonal-queue
BINARY_NAME=hexagonal-queue
DOCKER_IMAGE=hexagonal-queue:latest

# Default target
help:
	@echo "Hexagonal Queue - Wallet Service (Separated Architecture)"
	@echo ""
	@echo "🚀 Separated Architecture Commands:"
	@echo "  run-consumer       - Run consumer process only"
	@echo "  run-server         - Run server/publisher process only"
	@echo "  run-separated      - Instructions for running both processes"
	@echo ""
	@echo "📊 Load Testing Commands:"
	@echo "  load-test-rabbitmq - Load test RabbitMQ (100K messages)"
	@echo "  load-test-kafka    - Load test Kafka"
	@echo "  load-test-redis    - Load test Redis"
	@echo ""
	@echo "🔧 Legacy Monolithic Commands:"
	@echo "  run                - Run monolithic version (legacy)"
	@echo "  run-rabbitmq       - Run monolithic with RabbitMQ"
	@echo "  run-kafka          - Run monolithic with Kafka"  
	@echo "  run-redis          - Run monolithic with Redis"
	@echo ""
	@echo "🐳 Docker Commands:"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-up          - Start all services"
	@echo "  docker-up-minimal  - Start only MongoDB + RabbitMQ"
	@echo "  docker-up-infra    - Start all infrastructure services"
	@echo "  docker-down        - Stop all services"
	@echo "  docker-logs        - Show logs from all services"
	@echo "  docker-clean       - Clean up Docker resources"
	@echo ""
	@echo "🛠️  Development Commands:"
	@echo "  deps               - Download Go dependencies"
	@echo "  test               - Run tests"
	@echo "  test-api           - Test API endpoints"
	@echo "  clean              - Clean build artifacts"
	@echo "  dev-setup          - Set up development environment"

# Separated Architecture Commands
run-consumer:
	@echo "🚀 Starting consumer process..."
	@echo "This will consume messages from the queue and process transactions"
	QUEUE_PROVIDER=rabbitmq go run cmd/consumer/main.go

run-server:
	@echo "🚀 Starting server/publisher process..."
	@echo "This will start the HTTP API server for publishing messages"
	QUEUE_PROVIDER=rabbitmq go run cmd/server/main.go

run-consumer-kafka:
	@echo "🚀 Starting consumer process with Kafka..."
	QUEUE_PROVIDER=kafka go run cmd/consumer/main.go

run-server-kafka:
	@echo "🚀 Starting server/publisher process with Kafka..."
	QUEUE_PROVIDER=kafka go run cmd/server/main.go

run-consumer-redis:
	@echo "🚀 Starting consumer process with Redis..."
	QUEUE_PROVIDER=redis go run cmd/consumer/main.go

run-server-redis:
	@echo "🚀 Starting server/publisher process with Redis..."
	QUEUE_PROVIDER=redis go run cmd/server/main.go

run-separated:
	@echo "🚀 To run separated architecture:"
	@echo ""
	@echo "Terminal 1 (Consumer):"
	@echo "  make run-consumer"
	@echo ""
	@echo "Terminal 2 (Server):"
	@echo "  make run-server" 
	@echo ""
	@echo "Terminal 3 (Testing):"
	@echo "  make load-test-rabbitmq"
	@echo ""
	@echo "💡 Benefits of separation:"
	@echo "  - No RPC overhead (fire-and-forget pattern)"
	@echo "  - Independent scaling of consumer/publisher"
	@echo "  - Better performance for high-volume loads"
	@echo "  - Production-ready microservice architecture"

# Load Testing Commands
load-test-rabbitmq:
	@echo "🔥 Running RabbitMQ load test (100K messages)..."
	@echo "Make sure consumer is running in another terminal!"
	curl -X POST http://localhost:8080/test/queue/rabbitmq/deposit

load-test-kafka:
	@echo "🔥 Running Kafka load test..."
	@echo "Make sure Kafka consumer is running!"
	curl -X POST http://localhost:8080/test/queue/kafka/deposit

load-test-redis:
	@echo "🔥 Running Redis load test..."
	@echo "Make sure Redis consumer is running!"
	curl -X POST http://localhost:8080/test/queue/redis/deposit

# Legacy Monolithic Commands (for backward compatibility)
build:
	@echo "Building $(APP_NAME)..."
	go build -o bin/consumer cmd/consumer/main.go
	go build -o bin/server cmd/server/main.go

run:
	@echo "⚠️  Running monolithic version (legacy mode)"
	@echo "For better performance, use: make run-separated"
	MODE=MONOLITHIC QUEUE_PROVIDER=rabbitmq go run cmd/server/main.go

run-rabbitmq:
	@echo "⚠️  Running monolithic version with RabbitMQ"
	QUEUE_PROVIDER=rabbitmq go run cmd/server/main.go

run-kafka:
	@echo "⚠️  Running monolithic version with Kafka"
	QUEUE_PROVIDER=kafka go run cmd/server/main.go

run-redis:
	@echo "⚠️  Running monolithic version with Redis"
	QUEUE_PROVIDER=redis go run cmd/server/main.go

# Docker Commands
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

docker-up: docker-build
	@echo "Starting all services..."
	docker-compose up -d

docker-up-minimal:
	@echo "Starting minimal services (MongoDB + RabbitMQ)..."
	docker-compose up -d mongodb rabbitmq

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

# Development Setup
dev-setup: docker-up-minimal
	@echo "Setting up development environment..."
	@echo "Waiting for services to start..."
	@sleep 15
	@echo ""
	@echo "=== Development Environment Ready ==="
	@echo "MongoDB:    mongodb://admin:password123@localhost:27017/wallet_db?authSource=admin"
	@echo "RabbitMQ:   amqp://admin:password123@localhost:5672/ (UI: http://localhost:15672)"
	@echo ""
	@echo "🚀 Next Steps:"
	@echo "1. Terminal 1: make run-consumer"
	@echo "2. Terminal 2: make run-server"
	@echo "3. Terminal 3: make load-test-rabbitmq"

# Go Commands
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

# API Testing (requires server to be running)
test-api: test-health test-deposit test-balance

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

# Process Management
kill-processes:
	@echo "Killing any running Go processes..."
	pkill -f "go run" || true
	pkill -f "consumer/main.go" || true
	pkill -f "server/main.go" || true

# Service Status
status:
	@echo "Checking service status..."
	@echo ""
	@echo "Docker services:"
	docker-compose ps 2>/dev/null || echo "Docker Compose not running"
	@echo ""
	@echo "Go processes:"
	ps aux | grep -E "(consumer|server)/main.go" | grep -v grep || echo "No Go processes running"

# Quick Start
quick-start: dev-setup
	@echo ""
	@echo "🎯 Quick Start Complete!"
	@echo ""
	@echo "Run these commands in separate terminals:"
	@echo "  Terminal 1: make run-consumer"
	@echo "  Terminal 2: make run-server"
	@echo "  Terminal 3: make load-test-rabbitmq"
	@echo ""
	@echo "Or use: make run-separated (for instructions)"







