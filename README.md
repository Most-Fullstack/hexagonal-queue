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
- **Support**: Up to 100,000+ messages
- **Batch Processing**: Configurable batch sizes (25-1000 messages)
- **Rate Limiting**: Prevents connection exhaustion
- **Worker Pools**: Concurrent processing (1-10 workers)
- **Real-time Monitoring**: Queue depth and latency tracking

### Test Endpoints
```bash
# RabbitMQ Load Test (100K messages)
POST /api/v1/test/queue/rabbitmq/deposit

# Kafka Performance Test  
POST /api/v1/test/queue/kafka/deposit

# Redis Speed Test
POST /api/v1/test/queue/redis/deposit

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
// Configurable in handlers.go
const (
    numMessages = 100000  // Total messages to send
    workerPool  = 3       // Concurrent workers
    batchSize   = 100     // Messages per batch
)

// Timing
testDuration := 600 * time.Second  // Max test time
messageRate := 100                 // Target msg/sec
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
  -H "Authorization: admin" \
  -d '{
    "username": "testuser",
    "token": "user-token-123",
    "type_member": "member"
  }'

# Deposit money
curl -X POST http://localhost:8080/api/v1/wallet/deposit \
  -H "Content-Type: application/json" \
  -H "Authorization: admin" \
  -d '{
    "username": "testuser",
    "token": "user-token-123",
    "type_name": "DEPOSIT",
    "amount": 100.50,
    "channel": "api",
    "description": "Load test deposit"
  }'

# Check balance
curl -X POST http://localhost:8080/api/v1/wallet/balance \
  -H "Content-Type: application/json" \
  -H "Authorization: admin" \
  -d '{
    "username": "testuser"
  }'
```

### Load Testing
```bash
# RabbitMQ load test (100K messages)
curl -X POST http://localhost:8080/api/v1/test/queue/rabbitmq/deposit

# Expected output:
# 📤 Queued: 100/100000 messages (0.1%)
# 📤 Queued: 200/100000 messages (0.2%)
# ...
# 🟢 Queue: 0 msgs | Published: 25847 | Errors: 23
# 🏁 FINAL RESULTS:
# ✅ Successful: 99,977 (99.98%)
# ❌ Failed: 23 (0.02%)
# 🚀 Rate: 1,547.2 msg/sec
# ⏱️ Duration: 64.6s
```

### Queue Health Check
```bash
# Check all queue health
curl http://localhost:8080/health/queues

# Response:
{
  "status": "healthy",
  "providers": {
    "rabbitmq": "connected",
    "kafka": "connected", 
    "redis": "connected"
  },
  "active_provider": "rabbitmq"
}
```

## 🔄 Testing Different Queue Providers

The service allows seamless switching between queue providers to compare performance:

### Quick Provider Switch
```bash
# Test RabbitMQ performance
QUEUE_PROVIDER=rabbitmq go run cmd/main.go
curl -X POST http://localhost:8080/api/v1/test/queue/rabbitmq/deposit

# Switch to Kafka
QUEUE_PROVIDER=kafka go run cmd/main.go
curl -X POST http://localhost:8080/api/v1/test/queue/kafka/deposit

# Try Redis
QUEUE_PROVIDER=redis go run cmd/main.go
curl -X POST http://localhost:8080/api/v1/test/queue/redis/deposit
```

### Performance Comparison Results
Based on 100K message load tests:

| Provider | Avg Throughput | Success Rate | Latency | Connection Model |
|----------|----------------|--------------|---------|------------------|
| **RabbitMQ** | 1,200-2,000 msg/sec | 99.8% | 150ms | RPC with temp queues |
| **Kafka** | 8,000-12,000 msg/sec | 99.9% | 25ms | Producer/Consumer |
| **Redis** | 3,000-5,000 msg/sec | 99.7% | 50ms | Streams |

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
GET /metrics              # Prometheus-style metrics
GET /api/v1/wallet/health # Wallet service specific health
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
- **Message Volume**: 1K to 100K+ messages
- **Batch Sizes**: 25, 50, 100, 1000 messages per batch
- **Worker Pools**: 1-10 concurrent workers
- **Rate Limiting**: Prevents connection exhaustion
- **Duration Limits**: Maximum test time controls

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
