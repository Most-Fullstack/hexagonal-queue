package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"hexagonal-queue/internal/application/ports"
	"hexagonal-queue/internal/domain/models"
	"hexagonal-queue/pkg/utils"

	"github.com/redis/go-redis/v9"
)

// Adapter implements the QueuePort interface for Redis
type Adapter struct {
	client   *redis.Client
	config   Config
	consumer *Consumer
	closed   bool
	mutex    sync.RWMutex
}

// Config holds Redis configuration
type Config struct {
	Addr              string
	Password          string
	DB                int
	QueueName         string
	ResponseQueueName string
	ConsumerGroup     string
	BlockTime         time.Duration
	RequestTimeout    time.Duration
	MaxRetries        int
}

// Consumer handles message consumption from Redis streams
type Consumer struct {
	ctx     context.Context
	cancel  context.CancelFunc
	handler ports.TransactionHandler
	wg      sync.WaitGroup
}

// Message represents a Redis message
type Message struct {
	ID      string                    `json:"id"`
	Type    string                    `json:"type"`
	Data    models.TransactionRequest `json:"data"`
	ReplyTo string                    `json:"reply_to,omitempty"`
	Headers map[string]string         `json:"headers,omitempty"`
}

// NewAdapter creates a new Redis adapter
func NewAdapter(addr, password string, db int) (*Adapter, error) {
	config := Config{
		Addr:              addr,
		Password:          password,
		DB:                db,
		QueueName:         "wallet:transactions",
		ResponseQueueName: "wallet:responses",
		ConsumerGroup:     "wallet-service",
		BlockTime:         5 * time.Second,
		RequestTimeout:    30 * time.Second,
		MaxRetries:        3,
	}

	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	adapter := &Adapter{
		client: client,
		config: config,
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Create consumer group if it doesn't exist
	if err := adapter.ensureConsumerGroup(); err != nil {
		log.Printf("Warning: failed to create consumer group: %v", err)
	}

	return adapter, nil
}

// ensureConsumerGroup creates the consumer group if it doesn't exist
func (a *Adapter) ensureConsumerGroup() error {
	ctx := context.Background()

	// Try to create the stream first
	_, err := a.client.XAdd(ctx, &redis.XAddArgs{
		Stream: a.config.QueueName,
		Values: map[string]interface{}{"init": "init"},
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to initialize stream: %w", err)
	}

	// Create consumer group
	err = a.client.XGroupCreate(ctx, a.config.QueueName, a.config.ConsumerGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	return nil
}

// PublishTransaction publishes a transaction request and waits for response
func (a *Adapter) PublishTransaction(ctx context.Context, request models.TransactionRequest) (*models.TransactionResponse, error) {
	if a.closed {
		return nil, fmt.Errorf("connection is closed")
	}

	// Generate correlation ID
	correlationID := utils.GenerateTransactionID(request.Username)
	responseKey := fmt.Sprintf("%s:%s", a.config.ResponseQueueName, correlationID)

	// Create message
	message := Message{
		ID:      correlationID,
		Type:    strings.ToUpper(request.Action),
		Data:    request,
		ReplyTo: responseKey,
		Headers: map[string]string{
			"correlation_id": correlationID,
			"timestamp":      fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	// Marshal message
	messageData, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	// Publish to stream
	_, err = a.client.XAdd(ctx, &redis.XAddArgs{
		Stream: a.config.QueueName,
		Values: map[string]interface{}{
			"message": string(messageData),
		},
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to publish message: %w", err)
	}

	// Wait for response using blocking list operations
	responseCtx, cancel := context.WithTimeout(ctx, a.config.RequestTimeout)
	defer cancel()

	result, err := a.client.BLPop(responseCtx, a.config.RequestTimeout, responseKey).Result()
	if err != nil {
		if err == redis.Nil || err == context.DeadlineExceeded {
			return nil, fmt.Errorf("request timeout")
		}
		return nil, fmt.Errorf("failed to receive response: %w", err)
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("invalid response format")
	}

	var response models.TransactionResponse
	if err := json.Unmarshal([]byte(result[1]), &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

func (a *Adapter) PublishTransactionFireAndForget(ctx context.Context, request models.TransactionRequest) error {
	return nil
}

// PublishMessage publishes a message to a specific Redis list/stream
func (a *Adapter) PublishMessage(ctx context.Context, queueName string, message interface{}) error {
	if a.closed {
		return fmt.Errorf("connection is closed")
	}

	messageData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Use list for simple pub/sub
	return a.client.LPush(ctx, queueName, string(messageData)).Err()
}

// StartConsumer starts consuming messages from Redis streams
func (a *Adapter) StartConsumer(handler ports.TransactionHandler) error {
	if a.closed {
		return fmt.Errorf("connection is closed")
	}

	ctx, cancel := context.WithCancel(context.Background())

	a.consumer = &Consumer{
		ctx:     ctx,
		cancel:  cancel,
		handler: handler,
	}

	log.Printf("Redis consumer started, consuming from stream: %s", a.config.QueueName)

	a.consumer.wg.Add(1)
	go func() {
		defer a.consumer.wg.Done()
		a.consumeMessages()
	}()

	return nil
}

// consumeMessages consumes messages from Redis streams
func (a *Adapter) consumeMessages() {
	consumerName := fmt.Sprintf("consumer-%d", time.Now().Unix())

	for {
		select {
		case <-a.consumer.ctx.Done():
			return
		default:
			// Read from stream with consumer group
			streams, err := a.client.XReadGroup(a.consumer.ctx, &redis.XReadGroupArgs{
				Group:    a.config.ConsumerGroup,
				Consumer: consumerName,
				Streams:  []string{a.config.QueueName, ">"},
				Count:    1,
				Block:    a.config.BlockTime,
			}).Result()

			if err != nil {
				if err == redis.Nil || err == context.Canceled {
					continue
				}
				log.Printf("Failed to read from stream: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			for _, stream := range streams {
				for _, message := range stream.Messages {
					a.handleMessage(message, a.consumer.handler)

					// Acknowledge the message
					a.client.XAck(context.Background(), a.config.QueueName, a.config.ConsumerGroup, message.ID)
				}
			}
		}
	}
}

// handleMessage processes individual messages
func (a *Adapter) handleMessage(msg redis.XMessage, handler ports.TransactionHandler) {
	ctx := context.Background()

	messageData, ok := msg.Values["message"].(string)
	if !ok {
		log.Printf("Invalid message format: missing message field")
		return
	}

	var message Message
	if err := json.Unmarshal([]byte(messageData), &message); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		a.sendErrorResponse(message.ReplyTo, "Invalid message format")
		return
	}

	var response models.TransactionResponse

	// Handle based on message type
	switch strings.ToUpper(message.Type) {
	case "DEPOSIT":
		response = handler.HandleDeposit(ctx, message.Data)
	case "WITHDRAW":
		response = handler.HandleWithdraw(ctx, message.Data)
	case "ADD_MEMBER":
		response = handler.HandleCreateMember(ctx, message.Data)
	default:
		response = models.TransactionResponse{
			Status:  400,
			Message: "Unknown operation type",
		}
	}

	// Send response if ReplyTo is set
	if message.ReplyTo != "" {
		a.sendResponse(message.ReplyTo, response)
	}
}

// sendResponse sends response back to the requester
func (a *Adapter) sendResponse(replyTo string, response models.TransactionResponse) {
	responseData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	// Use RPUSH to add response to the reply queue
	if err := a.client.RPush(context.Background(), replyTo, string(responseData)).Err(); err != nil {
		log.Printf("Failed to send response: %v", err)
	}

	// Set expiration for the response queue
	a.client.Expire(context.Background(), replyTo, 5*time.Minute)
}

// sendErrorResponse sends error response
func (a *Adapter) sendErrorResponse(replyTo, errorMsg string) {
	if replyTo == "" {
		return
	}

	response := models.TransactionResponse{
		Status:  400,
		Message: errorMsg,
	}
	a.sendResponse(replyTo, response)
}

// StopConsumer stops the consumer
func (a *Adapter) StopConsumer() error {
	if a.consumer != nil {
		a.consumer.cancel()
		a.consumer.wg.Wait()
	}
	return nil
}

// IsConnected checks if the Redis connection is active
func (a *Adapter) IsConnected() bool {
	if a.closed {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return a.client.Ping(ctx).Err() == nil
}

// Close closes the Redis connection
func (a *Adapter) Close() error {
	a.closed = true

	if err := a.StopConsumer(); err != nil {
		log.Printf("Failed to stop consumer: %v", err)
	}

	if a.client != nil {
		return a.client.Close()
	}

	return nil
}

// GetStreamInfo returns information about the Redis stream (useful for monitoring)
func (a *Adapter) GetStreamInfo() (*redis.XInfoStream, error) {
	if a.closed {
		return nil, fmt.Errorf("connection is closed")
	}

	return a.client.XInfoStream(context.Background(), a.config.QueueName).Result()
}

// GetConsumerGroupInfo returns information about the consumer group
func (a *Adapter) GetConsumerGroupInfo() ([]redis.XInfoGroup, error) {
	if a.closed {
		return nil, fmt.Errorf("connection is closed")
	}

	return a.client.XInfoGroups(context.Background(), a.config.QueueName).Result()
}
