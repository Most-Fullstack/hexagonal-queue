package queue

import (
	"fmt"

	"hexagonal-queue/internal/application/ports"
	"hexagonal-queue/internal/infrastructure/adapters/queue/kafka"
	"hexagonal-queue/internal/infrastructure/adapters/queue/rabbitmq"
	"hexagonal-queue/internal/infrastructure/adapters/queue/redis"
	"hexagonal-queue/internal/infrastructure/config"
)

// QueueFactory creates queue adapters based on configuration
type QueueFactory struct {
	config *config.Config
}

// NewQueueFactory creates a new queue factory
func NewQueueFactory(cfg *config.Config) *QueueFactory {
	return &QueueFactory{config: cfg}
}

// CreateQueueAdapter creates a queue adapter based on the configured provider
func (f *QueueFactory) CreateQueueAdapter(provider string) (ports.QueuePort, error) {
	if provider == "" {
		provider = f.config.Queue.Provider
	}

	switch provider {
	case "rabbitmq":
		return rabbitmq.NewAdapter(f.config.Queue.RabbitMQ.URI)
	case "kafka":
		return kafka.NewAdapter(f.config.Queue.Kafka.Brokers)
	case "redis":
		return redis.NewAdapter(
			f.config.Queue.Redis.Addr,
			f.config.Queue.Redis.Password,
			f.config.Queue.Redis.DB,
		)
	default:
		return nil, fmt.Errorf("unsupported queue provider: %s", provider)
	}
}

// GetSupportedProviders returns a list of supported queue providers
func (f *QueueFactory) GetSupportedProviders() []string {
	return []string{"rabbitmq", "kafka", "redis"}
}

// CreateAllAdapters creates adapters for all supported providers for testing
func (f *QueueFactory) CreateAllAdapters() (map[string]ports.QueuePort, error) {
	adapters := make(map[string]ports.QueuePort)

	for _, provider := range f.GetSupportedProviders() {
		adapter, err := f.CreateQueueAdapter(provider)
		if err != nil {
			return nil, fmt.Errorf("failed to create %s adapter: %w", provider, err)
		}
		adapters[provider] = adapter
	}

	return adapters, nil
}
