# Hexagonal Queue - Wallet Service

A microservice implementing hexagonal architecture (ports and adapters pattern) with advanced queue provider testing and comparison capabilities for wallet operations.

## 🚀 Features

- **Hexagonal Architecture**: Clean separation between business logic and infrastructure
- **Multiple Queue Providers**: Support for RabbitMQ, Apache Kafka, and Redis with seamless switching
- **Advanced Load Testing**: Comprehensive queue performance testing with 100K+ message support
- **ACID Transactions**: MongoDB replica set with transaction support for data consistency
- **Wallet Operations**: Deposit, withdraw, balance check, member creation with full audit trail
- **REST API**: HTTP endpoints for wallet operations and performance testing
- **Connection Pooling**: Optimized connection management to prevent resource exhaustion
- **Real-time Monitoring**: Queue depth monitoring and performance metrics
- **Batch Processing**: Configurable batch processing with rate limiting

## 🏗️ Architecture

```
├── cmd/                    # Application entry point
├── internal/
│   ├── domain/            # Business logic (entities, services)
│   │   ├── models/        # Domain models (User, Wallet, Transaction)
│   │   └── services/      # Domain services (WalletService)
│   ├── application/       # Application layer
│   │   ├── ports/         # Interfaces (repository, queue)
│   │   └── usecases/      # Use cases (WalletUseCase)
│   └── infrastructure/    # Infrastructure layer
│       ├── adapters/      # External service adapters
│       │   ├── db/        # MongoDB adapter with replica set support
│       │   └── queue/     # Queue provider adapters
│       │       ├── factory/      # Queue provider factory
│       │       ├── rabbitmq/     # RabbitMQ with RPC support
│       │       ├── kafka/        # Kafka with topic-based messaging
│       │       └── redis/        # Redis Streams implementation
│       ├── config/        # Environment-based configuration
│       └── web/           # HTTP handlers, server, and load testing
└── pkg/                   # Shared utilities
    └── utils/             # Utility functions (UUID, decimal, time)
```

## 🔄 Queue Provider Comparison

| Provider | Pattern | Throughput | Use Case | Connection Model |
|----------|---------|------------|----------|------------------|
| **RabbitMQ** | RPC (Request-Reply) | ~1,000 msg/sec | Reliable messaging | Connection pooling |
| **Apache Kafka** | Topic-based | ~10,000 msg/sec | Event streaming | Producer/Consumer groups |
| **Redis** | Streams/Lists | ~5,000 msg/sec | Fast caching | Single connection |

## 📊 Load Testing Capabilities

### High-Volume Testing
- **Support**: Up to 1,000 messages (configurable)
- **Batch Processing**: Configurable batch sizes (default: 10,000 messages per batch)
- **Rate Limiting**: Prevents connection exhaustion with batch delays
- **Worker Pools**: Concurrent processing (default: 2 workers)
- **Real-time Monitoring**: Queue depth and latency tracking

### Test Endpoints
```bash
# RabbitMQ Load Test (1K messages with 2 workers)
POST /api/v1/test/queue/rabbitmq/deposit

# Generic Queue Test (configurable provider)
POST /api/v1/test/queue
{
  "provider": "rabbitmq",
  "username": "testuser",
  "action": "deposit",
  "amount": 100.0
}

# Queue Health Monitoring
GET /health/queues
```

### Performance Metrics
- **Throughput**: Messages per second
- **Latency**: Average/max publish latency  
- **Success Rate**: Percentage of successful operations
- **Queue Depth**: Real-time queue monitoring
- **Connection Efficiency**: Resource utilization tracking

## API Endpoints

### Wallet Operations
- `POST /api/v1/wallet/deposit` - Deposit money
- `POST /api/v1/wallet/withdraw` - Withdraw money
- `POST /api/v1/wallet/balance` - Check balance
- `POST /api/v1/wallet/balances` - Check multiple balances
- `POST /api/v1/wallet/member` - Create member
- `GET /api/v1/wallet/history/:username` - Get transaction history
- `POST /api/v1/wallet/queue/deposit` - Queue-based deposit (fire-and-forget)

### Testing & Load Testing
- `POST /api/v1/test/queue` - Test queue operations
- `POST /api/v1/test/queue/rabbitmq/deposit` - RabbitMQ load test (1K messages)

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

## 🛠️ Quick Start

### Using Makefile (Recommended)
```bash
# See all available commands
make help

# Start infrastructure services  
make dev-setup

# Run with specific queue provider
make run-rabbitmq
make run-kafka  
make run-redis

# Run load tests
make test-load-rabbitmq
make test-load-kafka
make test-load-redis

# Docker operations
make docker-up           # Start all services
make docker-up-minimal   # Start only MongoDB + RabbitMQ
make docker-down         # Stop all services
make docker-logs         # View logs
```

### Manual Setup
```bash
# Install dependencies
go mod tidy

# Start infrastructure (MongoDB + RabbitMQ)
docker-compose up -d mongodb rabbitmq

# Set environment variables
cp env.template .env

# Run the service
QUEUE_PROVIDER=rabbitmq go run cmd/main.go
```

## 🔧 Configuration

### Environment Variables (.env file)
```bash
# Server Configuration
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
GO_ENV=development

# Database Configuration (Replica Set)
MONGODB_URI=mongodb://admin:password123@localhost:27017/wallet_db?authSource=admin
MONGODB_DATABASE=wallet_db

# Queue Provider Selection
QUEUE_PROVIDER=rabbitmq  # or kafka, redis

# RabbitMQ Configuration
RABBITMQ_URI=amqp://admin:password123@localhost:5672/
RABBITMQ_QUEUE_NAME=wallet_queue
RABBITMQ_PREFETCH_COUNT=1

# Kafka Configuration  
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=wallet-transactions
KAFKA_CONSUMER_GROUP=wallet-service

# Redis Configuration
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=password123
REDIS_DB=0
```

### Load Test Configuration
```go
// Current configuration in handlers.go
const (
    numMessages  = 1000              // Total messages to send
    workerPool   = 2                 // Concurrent workers
    batchSize    = 10000             // Messages per batch
    testDuration = 600 * time.Second // Max test time (10 minutes)
    messageRate  = 100               // Target msg/sec
    batchDelay   = 1 * time.Second   // Delay between batches
)
```

## 🐳 Docker Setup

### Full Environment
```bash
# Start complete environment
docker-compose up -d

# Services included:
# - MongoDB (replica set with auth)
# - RabbitMQ (with management UI)
# - Apache Kafka + Zookeeper
# - Redis (with persistence)
# - Hexagonal Queue Service
```

### Minimal Setup (MongoDB + RabbitMQ only)
```bash
docker-compose up -d mongodb rabbitmq
```

### Service URLs
- **Wallet Service**: http://localhost:8080
- **RabbitMQ Management**: http://localhost:15672 (admin/password123)
- **MongoDB**: mongodb://admin:password123@localhost:27017
- **Redis**: redis://localhost:6379 (password: password123)

## 📊 API Usage Examples

### Wallet Operations
```bash
# Create a member
curl -X POST http://localhost:8080/api/v1/wallet/member \
  -H "Content-Type: application/json" \
  -H "Authorization: admin-parent-token" \
  -d '{
    "parent_token": "admin-parent-token",
    "token": "user-token-123",
    "username": "testuser",
    "action": "ADD_MEMBER",
    "type_name": "ADD_MEMBER",
    "channel": "web-api",
    "description": "New member registration"
  }'

# Response:
{
  "status": 201,
  "message": "Member created successfully",
  "data": {
    "username": "testuser",
    "parent_token": "admin-parent-token",
    "token": "user-token-123"
  }
}

# Deposit money
curl -X POST http://localhost:8080/api/v1/wallet/deposit \
  -H "Content-Type: application/json" \
  -H "Authorization: admin-parent-token" \
  -d '{
    "parent_token": "admin-parent-token",
    "token": "user-token-123",
    "username": "testuser",
    "action": "DEPOSIT",
    "type_name": "DEPOSIT",
    "amount": 100.50,
    "channel": "web-api",
    "description": "Initial deposit"
  }'

# Success Response:
{
  "status": 200,
  "message": "Deposit successful",
  "data": {
    "username": "testuser",
    "current_balance": 100.50,
    "before_balance": 0.00,
    "after_balance": 100.50
  }
}

# Withdraw money
curl -X POST http://localhost:8080/api/v1/wallet/withdraw \
  -H "Content-Type: application/json" \
  -H "Authorization: admin-parent-token" \
  -d '{
    "parent_token": "admin-parent-token",
    "token": "user-token-123",
    "username": "testuser",
    "action": "WITHDRAW",
    "type_name": "WITHDRAW",
    "amount": 25.00,
    "channel": "web-api",
    "description": "ATM withdrawal"
  }'

# Success Response:
{
  "status": 200,
  "message": "Withdrawal successful",
  "data": {
    "username": "testuser",
    "current_balance": 75.50,
    "before_balance": 100.50,
    "after_balance": 75.50
  }
}

# Check balance
curl -X POST http://localhost:8080/api/v1/wallet/balance \
  -H "Content-Type: application/json" \
  -H "Authorization: admin-parent-token" \
  -d '{
    "username": "testuser"
  }'

# Response:
{
  "status": 200,
  "message": "Balance retrieved successfully",
  "data": {
    "username": "testuser",
    "current_balance": 75.50
  }
}

# Check multiple balances
curl -X POST http://localhost:8080/api/v1/wallet/balances \
  -H "Content-Type: application/json" \
  -H "Authorization: admin-parent-token" \
  -d '{
    "usernames": ["testuser", "user2", "user3"]
  }'

# Get transaction history
curl -X GET "http://localhost:8080/api/v1/wallet/history/testuser?limit=5" \
  -H "Authorization: admin-parent-token"

# Response:
{
  "status": 200,
  "message": "Success",
  "data": {
    "username": "testuser",
    "limit": 5,
    "transactions": []
  }
}
```

### Load Testing
```bash
# RabbitMQ load test (1K messages with 2 workers)
curl -X POST http://localhost:8080/api/v1/test/queue/rabbitmq/deposit

# Expected Console Output:
# 📤 Queued: 100/1000 messages (10.0%)
# 📤 Queued: 200/1000 messages (20.0%)
# ...
# 🟢 Queue: 2 msgs (0.000%) | Published: 847 | Errors: 3 | Peak: 15
# 📤 All 1000 messages queued. Waiting for workers to finish...
# 🏁 FINAL RESULTS:
# 📊 Total Messages: 1000
# ✅ Successful: 997 (99.7%)
# ❌ Failed: 3 (0.3%)
# ⏱️ Duration: 15.2s
# 🚀 Rate: 65.6 msg/sec
# 📈 Avg Latency: 23ms
# 📊 Queue Depth: avg=1.2, max=15

# JSON Response:
{
  "message": "RabbitMQ Load Test Results",
  "config": {
    "total_messages": 1000,
    "worker_pool": 2,
    "batch_size": 10000,
    "target_rate": 100,
    "test_duration": "10m0s"
  },
  "stats": {
    "duration": "15.234s",
    "messages_published": 997,
    "publish_errors": 3,
    "publish_rate_per_sec": 65.6,
    "avg_publish_latency": "23ms",
    "max_publish_latency": "145ms",
    "avg_queue_depth": 1.2,
    "max_queue_depth": 15,
    "success_rate_percent": 99.7
  }
}

# Generic queue test with configurable provider
curl -X POST http://localhost:8080/api/v1/test/queue \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "rabbitmq",
    "username": "testuser",
    "action": "deposit",
    "amount": 50.0
  }'

# Response:
{
  "test_provider": "rabbitmq",
  "test_action": "deposit",
  "queue_response": {
    "status": 200,
    "message": "Deposit successful",
    "data": {
      "username": "testuser",
      "current_balance": 125.50,
      "before_balance": 75.50,
      "after_balance": 125.50
    }
  }
}
```

### Error Response Examples
```bash
# Missing Authorization Header (401)
{
  "status": 401,
  "message": "Authorization header is required"
}

# Invalid Request Format (400)
{
  "status": 400,
  "message": "Invalid request format",
  "error": "Key: 'TransactionRequest.Username' Error:Field validation for 'Username' failed on the 'required' tag"
}

# Insufficient Balance (400)
{
  "status": 400,
  "message": "Insufficient balance for withdrawal",
  "data": {
    "username": "testuser",
    "current_balance": 25.00,
    "requested_amount": 100.00
  }
}

# User Not Found (404)
{
  "status": 404,
  "message": "User not found",
  "data": {
    "username": "nonexistentuser"
  }
}

# Queue Publishing Error (500)
{
  "status": 500,
  "message": "Failed to publish transaction",
  "error": "connection refused"
}
```

### Queue Health Check
```bash
# Check all queue health
curl http://localhost:8080/health/queues

# Response:
{
  "status": "healthy",
  "queues": {
    "rabbitmq": {"status": "available"},
    "kafka": {"status": "available"},
    "redis": {"status": "available"}
  }
}

# Service health check
curl http://localhost:8080/health

# Response:
{
  "status": "healthy",
  "service": "hexagonal-queue-wallet",
  "version": "1.0.0"
}

# Metrics endpoint
curl http://localhost:8080/metrics

# Response:
{
  "service": "hexagonal-queue-wallet",
  "metrics": {
    "total_transactions": 0,
    "active_connections": 0,
    "queue_health": {
      "rabbitmq": "connected",
      "kafka": "connected",
      "redis": "connected"
    }
  }
}
```

## 🔄 Testing Different Queue Providers

The service allows seamless switching between queue providers to compare performance:

### Quick Provider Switch
```bash
# Test RabbitMQ performance with load test
QUEUE_PROVIDER=rabbitmq go run cmd/main.go
curl -X POST http://localhost:8080/api/v1/test/queue/rabbitmq/deposit

# Test different providers with generic endpoint
curl -X POST http://localhost:8080/api/v1/test/queue \
  -H "Content-Type: application/json" \
  -d '{"provider": "rabbitmq", "username": "testuser", "action": "deposit", "amount": 100.0}'

curl -X POST http://localhost:8080/api/v1/test/queue \
  -H "Content-Type: application/json" \
  -d '{"provider": "kafka", "username": "testuser", "action": "deposit", "amount": 100.0}'

curl -X POST http://localhost:8080/api/v1/test/queue \
  -H "Content-Type: application/json" \
  -d '{"provider": "redis", "username": "testuser", "action": "deposit", "amount": 100.0}'
```

### Queue-based Deposit (Fire-and-Forget)
```bash
# Direct queue deposit without waiting for response
curl -X POST http://localhost:8080/api/v1/wallet/queue/deposit \
  -H "Content-Type: application/json" \
  -H "Authorization: admin-parent-token" \
  -d '{
    "parent_token": "admin-parent-token",
    "token": "user-token-123",
    "username": "testuser",
    "action": "DEPOSIT",
    "type_name": "DEPOSIT",
    "amount": 50.0,
    "channel": "async-api",
    "description": "Async deposit via queue"
  }'

# Response (immediate):
{
  "message": "Transaction queued successfully",
  "queue_id": "queue-tx-12345"
}
```

### Performance Comparison Results
Based on current 1K message load tests with 2 workers:

| Provider | Avg Throughput | Success Rate | Latency | Connection Model | Current Status |
|----------|----------------|--------------|---------|------------------|----------------|
| **RabbitMQ** | 50-100 msg/sec | 99.7% | 23ms | Fire-and-forget | ✅ Implemented |
| **Kafka** | TBD | TBD | TBD | Producer/Consumer | 🚧 Available |
| **Redis** | TBD | TBD | TBD | Streams | 🚧 Available |

*Note: Current load testing is configured for 1,000 messages with 2 concurrent workers. Performance can be scaled by adjusting these parameters in the handlers configuration.*

## 💼 Business Logic (Based on Indo-Wallet-Service)

### Core Features
- **ACID Compliance**: MongoDB replica set transactions for data consistency
- **Balance Validation**: Ensures sufficient funds before operations
- **Parent-Child Relationships**: Hierarchical user structure support
- **Audit Trail**: Complete transaction history and wallet statements
- **Error Handling**: Comprehensive error handling with automatic rollbacks

### Transaction Flow
```
1. Request Validation → 2. User Authentication → 3. Balance Check
         ↓                        ↓                       ↓
4. Database Transaction → 5. Queue Publication → 6. Consumer Processing
         ↓                        ↓                       ↓
7. Wallet Update → 8. Transaction Log → 9. Response to Client
```

### Data Models
- **Users**: Username, tokens, member types, status
- **Wallets**: Balance tracking, parent-child relationships
- **Transactions**: Complete audit trail with amounts, descriptions

## 📊 Monitoring & Observability

### Real-time Metrics
- **Queue Depth Monitoring**: Live tracking of message backlog
- **Throughput Metrics**: Messages per second, success rates
- **Latency Tracking**: Average and maximum processing times
- **Connection Health**: Active connections and error rates

### Health Endpoints
```bash
GET /health                 # Overall service health
GET /health/queues         # Queue provider connectivity  
GET /metrics              # Service metrics
```

### Log Levels
```bash
LOG_LEVEL=debug    # Detailed debugging info
LOG_LEVEL=info     # General operational info  
LOG_LEVEL=warn     # Warning messages
LOG_LEVEL=error    # Error messages only
```

## 🧪 Load Testing Features

### Configurable Test Parameters
- **Message Volume**: Currently 1K messages (configurable in code)
- **Batch Sizes**: 10,000 messages per batch (configurable)
- **Worker Pools**: 2 concurrent workers (configurable 1-10)
- **Rate Limiting**: 100 msg/sec target rate with 1s batch delays
- **Duration Limits**: 10 minutes maximum test time

### Test Outputs
```bash
📤 Progress: Real-time queuing progress
🟢 Queue Monitoring: Live queue depth tracking  
🏁 Final Results: Comprehensive performance summary
📊 Detailed Metrics: JSON response with full statistics
```

## 🚀 Production Considerations

### Performance Tuning
- **Connection Pooling**: Optimized for high-throughput scenarios
- **Batch Processing**: Configurable batch sizes for efficiency
- **Rate Limiting**: Prevents system overload
- **Circuit Breakers**: Automatic failover capabilities

### Security Features
- **Authentication**: Token-based authentication
- **Authorization**: Parent-child permission validation
- **Input Validation**: Comprehensive request validation
- **SQL Injection Protection**: MongoDB parameterized queries

## 🤝 Contributing

### Development Guidelines
1. **Hexagonal Architecture**: Maintain clean separation of concerns
2. **Interface Compliance**: All queue providers must implement common interfaces
3. **Test Coverage**: Add comprehensive tests for new features
4. **Documentation**: Update README and inline documentation
5. **Load Testing**: Ensure new providers support high-volume testing

### Adding New Queue Providers
1. Implement `ports.QueuePort` interface
2. Add factory registration in `queue/factory.go`
3. Add configuration in `config/config.go`
4. Create load test endpoint in `web/handlers.go`
5. Update documentation

## 📄 License

This project is for educational and testing purposes, demonstrating:
- Hexagonal architecture implementation
- Multiple queue provider integration
- High-performance load testing capabilities
- Production-ready wallet service patterns

Perfect for learning about microservices, queue systems, and performance testing methodologies.








