package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Queue    QueueConfig    `json:"queue"`
	Cache    CacheConfig    `json:"cache,omitempty"`
	Logging  LoggingConfig  `json:"logging"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         string        `json:"port"`
	Host         string        `json:"host"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	MongoDB MongoDBConfig `json:"mongodb"`
}

// MongoDBConfig holds MongoDB-specific configuration
type MongoDBConfig struct {
	URI      string `json:"uri"`
	Database string `json:"database"`
}

// QueueConfig holds queue configuration
type QueueConfig struct {
	Provider string         `json:"provider"` // "rabbitmq", "kafka", "redis"
	RabbitMQ RabbitMQConfig `json:"rabbitmq,omitempty"`
	Kafka    KafkaConfig    `json:"kafka,omitempty"`
	Redis    RedisConfig    `json:"redis,omitempty"`
}

// RabbitMQConfig holds RabbitMQ configuration
type RabbitMQConfig struct {
	URI            string        `json:"uri"`
	QueueName      string        `json:"queue_name"`
	Exchange       string        `json:"exchange"`
	RoutingKey     string        `json:"routing_key"`
	ConsumerTag    string        `json:"consumer_tag"`
	PrefetchCount  int           `json:"prefetch_count"`
	ReconnectDelay time.Duration `json:"reconnect_delay"`
	RequestTimeout time.Duration `json:"request_timeout"`
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers        []string      `json:"brokers"`
	Topic          string        `json:"topic"`
	ResponseTopic  string        `json:"response_topic"`
	ConsumerGroup  string        `json:"consumer_group"`
	BatchSize      int           `json:"batch_size"`
	BatchTimeout   time.Duration `json:"batch_timeout"`
	RequestTimeout time.Duration `json:"request_timeout"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Addr              string        `json:"addr"`
	Password          string        `json:"password"`
	DB                int           `json:"db"`
	QueueName         string        `json:"queue_name"`
	ResponseQueueName string        `json:"response_queue_name"`
	ConsumerGroup     string        `json:"consumer_group"`
	BlockTime         time.Duration `json:"block_time"`
	RequestTimeout    time.Duration `json:"request_timeout"`
	MaxRetries        int           `json:"max_retries"`
}

// CacheConfig holds cache configuration (optional Redis cache)
type CacheConfig struct {
	Enabled  bool   `json:"enabled"`
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	TTL      int    `json:"ttl"` // seconds
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `json:"level"`  // debug, info, warn, error
	Format string `json:"format"` // json, text
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			ReadTimeout:  parseDuration(getEnv("SERVER_READ_TIMEOUT", "10s")),
			WriteTimeout: parseDuration(getEnv("SERVER_WRITE_TIMEOUT", "10s")),
			IdleTimeout:  parseDuration(getEnv("SERVER_IDLE_TIMEOUT", "60s")),
		},
		Database: DatabaseConfig{
			MongoDB: MongoDBConfig{
				URI:      getEnv("MONGODB_URI", "mongodb://localhost:27017"),
				Database: getEnv("MONGODB_DATABASE", "wallet_db"),
			},
		},
		Queue: QueueConfig{
			Provider: strings.ToLower(getEnv("QUEUE_PROVIDER", "rabbitmq")),
		},
		Cache: CacheConfig{
			Enabled:  parseBool(getEnv("CACHE_ENABLED", "false")),
			Addr:     getEnv("CACHE_REDIS_ADDR", "localhost:6379"),
			Password: getEnv("CACHE_REDIS_PASSWORD", ""),
			DB:       parseInt(getEnv("CACHE_REDIS_DB", "1")),
			TTL:      parseInt(getEnv("CACHE_TTL", "300")),
		},
		Logging: LoggingConfig{
			Level:  strings.ToLower(getEnv("LOG_LEVEL", "info")),
			Format: strings.ToLower(getEnv("LOG_FORMAT", "text")),
		},
	}

	// Load queue-specific configuration based on provider
	switch config.Queue.Provider {
	case "rabbitmq":
		config.Queue.RabbitMQ = RabbitMQConfig{
			URI:            getEnv("RABBITMQ_URI", "amqp://guest:guest@localhost:5672/"),
			QueueName:      getEnv("RABBITMQ_QUEUE_NAME", "wallet_queue"),
			Exchange:       getEnv("RABBITMQ_EXCHANGE", ""),
			RoutingKey:     getEnv("RABBITMQ_ROUTING_KEY", "wallet_queue"),
			ConsumerTag:    getEnv("RABBITMQ_CONSUMER_TAG", "wallet_consumer"),
			PrefetchCount:  parseInt(getEnv("RABBITMQ_PREFETCH_COUNT", "1")),
			ReconnectDelay: parseDuration(getEnv("RABBITMQ_RECONNECT_DELAY", "5s")),
			RequestTimeout: parseDuration(getEnv("RABBITMQ_REQUEST_TIMEOUT", "60s")),
		}
	case "kafka":
		brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
		config.Queue.Kafka = KafkaConfig{
			Brokers:        brokers,
			Topic:          getEnv("KAFKA_TOPIC", "wallet-transactions"),
			ResponseTopic:  getEnv("KAFKA_RESPONSE_TOPIC", "wallet-responses"),
			ConsumerGroup:  getEnv("KAFKA_CONSUMER_GROUP", "wallet-service"),
			BatchSize:      parseInt(getEnv("KAFKA_BATCH_SIZE", "100")),
			BatchTimeout:   parseDuration(getEnv("KAFKA_BATCH_TIMEOUT", "1s")),
			RequestTimeout: parseDuration(getEnv("KAFKA_REQUEST_TIMEOUT", "30s")),
		}
	case "redis":
		config.Queue.Redis = RedisConfig{
			Addr:              getEnv("REDIS_ADDR", "localhost:6379"),
			Password:          getEnv("REDIS_PASSWORD", ""),
			DB:                parseInt(getEnv("REDIS_DB", "0")),
			QueueName:         getEnv("REDIS_QUEUE_NAME", "wallet:transactions"),
			ResponseQueueName: getEnv("REDIS_RESPONSE_QUEUE_NAME", "wallet:responses"),
			ConsumerGroup:     getEnv("REDIS_CONSUMER_GROUP", "wallet-service"),
			BlockTime:         parseDuration(getEnv("REDIS_BLOCK_TIME", "5s")),
			RequestTimeout:    parseDuration(getEnv("REDIS_REQUEST_TIMEOUT", "30s")),
			MaxRetries:        parseInt(getEnv("REDIS_MAX_RETRIES", "3")),
		}
	default:
		return nil, fmt.Errorf("unsupported queue provider: %s", config.Queue.Provider)
	}

	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// validate validates the configuration
func (c *Config) validate() error {
	// Validate queue provider
	validProviders := []string{"rabbitmq", "kafka", "redis"}
	if !contains(validProviders, c.Queue.Provider) {
		return fmt.Errorf("invalid queue provider: %s, must be one of: %s",
			c.Queue.Provider, strings.Join(validProviders, ", "))
	}

	// Validate MongoDB URI
	if c.Database.MongoDB.URI == "" {
		return fmt.Errorf("MongoDB URI is required")
	}

	if c.Database.MongoDB.Database == "" {
		return fmt.Errorf("MongoDB database name is required")
	}

	// Provider-specific validation
	switch c.Queue.Provider {
	case "rabbitmq":
		if c.Queue.RabbitMQ.URI == "" {
			return fmt.Errorf("RabbitMQ URI is required")
		}
	case "kafka":
		if len(c.Queue.Kafka.Brokers) == 0 {
			return fmt.Errorf("Kafka brokers are required")
		}
	case "redis":
		if c.Queue.Redis.Addr == "" {
			return fmt.Errorf("Redis address is required")
		}
	}

	// Validate logging level
	validLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLevels, c.Logging.Level) {
		return fmt.Errorf("invalid log level: %s, must be one of: %s",
			c.Logging.Level, strings.Join(validLevels, ", "))
	}

	return nil
}

// GetDSN returns a data source name for the current queue provider
func (c *Config) GetDSN() string {
	switch c.Queue.Provider {
	case "rabbitmq":
		return c.Queue.RabbitMQ.URI
	case "kafka":
		return strings.Join(c.Queue.Kafka.Brokers, ",")
	case "redis":
		return c.Queue.Redis.Addr
	default:
		return ""
	}
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return strings.ToLower(getEnv("GO_ENV", "development")) == "production"
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return !c.IsProduction()
}

// Helper functions

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// parseInt parses string to int with error handling
func parseInt(s string) int {
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return 0
}

// parseBool parses string to bool with error handling
func parseBool(s string) bool {
	if val, err := strconv.ParseBool(s); err == nil {
		return val
	}
	return false
}

// parseDuration parses string to time.Duration with error handling
func parseDuration(s string) time.Duration {
	if duration, err := time.ParseDuration(s); err == nil {
		return duration
	}
	return 0
}

// contains checks if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// PrintConfig prints configuration for debugging (without sensitive data)
func (c *Config) PrintConfig() {
	fmt.Printf("=== Configuration ===\n")
	fmt.Printf("Server: %s:%s\n", c.Server.Host, c.Server.Port)
	fmt.Printf("Database: %s\n", c.Database.MongoDB.Database)
	fmt.Printf("Queue Provider: %s\n", c.Queue.Provider)
	fmt.Printf("Cache Enabled: %t\n", c.Cache.Enabled)
	fmt.Printf("Log Level: %s\n", c.Logging.Level)
	fmt.Printf("Environment: %s\n", getEnv("GO_ENV", "development"))
	fmt.Printf("====================\n")
}








