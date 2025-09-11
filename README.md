# Hexagonal Queue - Wallet Service

A microservice implementing hexagonal architecture (ports and adapters pattern) with multiple queue provider support for wallet operations.

## Features

- **Hexagonal Architecture**: Clean separation between business logic and infrastructure
- **Multiple Queue Providers**: Support for RabbitMQ, Apache Kafka, and Redis
- **ACID Transactions**: MongoDB transactions for data consistency
- **Wallet Operations**: Deposit, withdraw, balance check, member creation
- **REST API**: HTTP endpoints for wallet operations
- **Configuration Management**: Environment-based configuration
- **Health Checks**: Health and metrics endpoints

## Architecture

```
├── cmd/                    # Application entry point
├── internal/
│   ├── domain/            # Business logic (entities, services)
│   │   ├── models/        # Domain models
│   │   └── services/      # Domain services
│   ├── application/       # Application layer
│   │   ├── ports/         # Interfaces (repository, queue)
│   │   └── usecases/      # Use cases (application services)
│   └── infrastructure/    # Infrastructure layer
│       ├── adapters/      # External service adapters
│       │   ├── db/        # Database adapters
│       │   └── queue/     # Queue provider adapters
│       │       ├── rabbitmq/
│       │       ├── kafka/
│       │       └── redis/
│       ├── config/        # Configuration management
│       └── web/           # HTTP handlers and server
└── pkg/                   # Shared utilities
    └── utils/             # Utility functions
```

## Queue Providers

### RabbitMQ
- **Pattern**: RPC (Request-Reply)
- **Features**: Reliable message delivery, acknowledgments
- **Use Case**: Traditional messaging scenarios

### Apache Kafka
- **Pattern**: Topic-based messaging
- **Features**: High throughput, distributed processing
- **Use Case**: High-volume event streaming

### Redis
- **Pattern**: Streams and Lists
- **Features**: In-memory performance, pub/sub
- **Use Case**: Fast caching and lightweight messaging

## API Endpoints

### Wallet Operations
- `POST /api/v1/wallet/deposit` - Deposit money
- `POST /api/v1/wallet/withdraw` - Withdraw money
- `POST /api/v1/wallet/balance` - Check balance
- `POST /api/v1/wallet/balances` - Check multiple balances
- `POST /api/v1/wallet/member` - Create member

### Testing
- `POST /api/v1/test/queue` - Test queue operations

### Health & Monitoring
- `GET /health` - Service health check
- `GET /health/queues` - Queue health status
- `GET /metrics` - Service metrics

## Environment Variables

### Server Configuration
```bash
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
```

### Database Configuration
```bash
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=wallet_db
```

### Queue Provider Selection
```bash
QUEUE_PROVIDER=rabbitmq  # or kafka, redis
```

### RabbitMQ Configuration
```bash
RABBITMQ_URI=amqp://guest:guest@localhost:5672/
RABBITMQ_QUEUE_NAME=wallet_queue
RABBITMQ_PREFETCH_COUNT=1
```

### Kafka Configuration
```bash
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=wallet-transactions
KAFKA_CONSUMER_GROUP=wallet-service
```

### Redis Configuration
```bash
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_QUEUE_NAME=wallet:transactions
```

## Installation & Setup

### Prerequisites
- Go 1.21+
- MongoDB
- One of: RabbitMQ, Apache Kafka, or Redis

### Build and Run
```bash
# Install dependencies
go mod download

# Run with RabbitMQ
QUEUE_PROVIDER=rabbitmq go run cmd/main.go

# Run with Kafka
QUEUE_PROVIDER=kafka go run cmd/main.go

# Run with Redis
QUEUE_PROVIDER=redis go run cmd/main.go
```

### Docker Compose (Development)
```yaml
version: '3.8'
services:
  mongodb:
    image: mongo:5.0
    ports:
      - "27017:27017"
  
  rabbitmq:
    image: rabbitmq:3-management
    ports:
      - "5672:5672"
      - "15672:15672"
  
  kafka:
    image: confluentinc/cp-kafka:latest
    ports:
      - "9092:9092"
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
  
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```

## Usage Examples

### Deposit Money
```bash
curl -X POST http://localhost:8080/api/v1/wallet/deposit \
  -H "Content-Type: application/json" \
  -H "Authorization: parent-token-123" \
  -d '{
    "username": "user123",
    "token": "user-token-456",
    "type_name": "DEPOSIT",
    "amount": 100.50,
    "channel": "api",
    "description": "Initial deposit"
  }'
```

### Check Balance
```bash
curl -X POST http://localhost:8080/api/v1/wallet/balance \
  -H "Content-Type: application/json" \
  -H "Authorization: parent-token-123" \
  -d '{
    "username": "user123"
  }'
```

### Test Queue Performance
```bash
curl -X POST http://localhost:8080/api/v1/test/queue \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "rabbitmq",
    "username": "testuser",
    "action": "deposit",
    "amount": 50.0
  }'
```

## Testing Different Queue Providers

The service allows you to test and compare the performance of different queue providers:

1. **Start with RabbitMQ**: `QUEUE_PROVIDER=rabbitmq go run cmd/main.go`
2. **Switch to Kafka**: `QUEUE_PROVIDER=kafka go run cmd/main.go`  
3. **Try Redis**: `QUEUE_PROVIDER=redis go run cmd/main.go`

Each provider implements the same interface, allowing seamless switching without code changes.

## Business Logic (Based on Indo-Wallet-Service)

- **ACID Compliance**: All wallet operations use database transactions
- **Balance Validation**: Ensures balance consistency before and after operations
- **Parent-Child Relationships**: Support for hierarchical user structures
- **Audit Trail**: Complete transaction history with wallet statements
- **Error Handling**: Comprehensive error handling and rollback mechanisms

## Monitoring & Observability

- Health check endpoints for service and queue connectivity
- Metrics endpoint for monitoring key performance indicators
- Structured logging with configurable levels
- Transaction tracing for debugging

## Contributing

1. Follow hexagonal architecture principles
2. Add tests for new features
3. Update documentation
4. Ensure all queue providers are supported

## License

This project is for educational and testing purposes, demonstrating hexagonal architecture with multiple queue providers.
