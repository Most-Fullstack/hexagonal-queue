package ports

import (
	"context"
	"hexagonal-queue/internal/domain/models"
)

// QueuePort defines the interface for message queue operations
type QueuePort interface {
	// Producer operations
	PublishTransaction(ctx context.Context, request models.TransactionRequest) (*models.TransactionResponse, error)
	PublishMessage(ctx context.Context, queueName string, message interface{}) error

	// Consumer operations
	StartConsumer(handler TransactionHandler) error
	StopConsumer() error

	// Connection management
	Close() error
	IsConnected() bool
}

// TransactionHandler defines the interface for handling queue messages
type TransactionHandler interface {
	HandleDeposit(ctx context.Context, request models.TransactionRequest) models.TransactionResponse
	HandleWithdraw(ctx context.Context, request models.TransactionRequest) models.TransactionResponse
	HandleCreateMember(ctx context.Context, request models.TransactionRequest) models.TransactionResponse
	HandleBalanceCheck(ctx context.Context, username, parentToken string) models.TransactionResponse
}

// QueueConfig defines queue configuration
type QueueConfig struct {
	QueueName   string                 `json:"queue_name"`
	Exchange    string                 `json:"exchange"`
	RoutingKey  string                 `json:"routing_key"`
	Durable     bool                   `json:"durable"`
	AutoDelete  bool                   `json:"auto_delete"`
	Exclusive   bool                   `json:"exclusive"`
	NoWait      bool                   `json:"no_wait"`
	Arguments   map[string]interface{} `json:"arguments"`
	ConsumerTag string                 `json:"consumer_tag"`
	AutoAck     bool                   `json:"auto_ack"`
}

// MessageMetadata contains message metadata
type MessageMetadata struct {
	MessageID     string            `json:"message_id"`
	CorrelationID string            `json:"correlation_id"`
	ReplyTo       string            `json:"reply_to"`
	Timestamp     int64             `json:"timestamp"`
	Type          string            `json:"type"`
	Headers       map[string]string `json:"headers"`
}
