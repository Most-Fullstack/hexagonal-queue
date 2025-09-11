package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"hexagonal-queue/internal/application/ports"
	"hexagonal-queue/internal/domain/models"
	"hexagonal-queue/pkg/utils"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Adapter implements the QueuePort interface for RabbitMQ
type Adapter struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	config  Config
	closed  bool
}

// Config holds RabbitMQ configuration
type Config struct {
	URI            string
	QueueName      string
	Exchange       string
	RoutingKey     string
	ConsumerTag    string
	PrefetchCount  int
	ReconnectDelay time.Duration
	RequestTimeout time.Duration
}

// NewAdapter creates a new RabbitMQ adapter
func NewAdapter(uri string) (*Adapter, error) {
	config := Config{
		URI:            uri,
		QueueName:      "wallet_queue",
		Exchange:       "",
		RoutingKey:     "wallet_queue",
		ConsumerTag:    "wallet_consumer",
		PrefetchCount:  1,
		ReconnectDelay: 5 * time.Second,
		RequestTimeout: 60 * time.Second,
	}

	adapter := &Adapter{
		config: config,
	}

	if err := adapter.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	if err := adapter.setupQueue(); err != nil {
		return nil, fmt.Errorf("failed to setup queue: %w", err)
	}

	return adapter, nil
}

// connect establishes connection to RabbitMQ
func (a *Adapter) connect() error {
	var err error

	a.conn, err = amqp.Dial(a.config.URI)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	a.channel, err = a.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Set QoS
	if err := a.channel.Qos(a.config.PrefetchCount, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	return nil
}

// setupQueue declares the queue and exchange
func (a *Adapter) setupQueue() error {
	// Declare queue
	_, err := a.channel.QueueDeclare(
		a.config.QueueName, // name
		false,              // durable
		true,               // delete when unused
		false,              // exclusive
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	return nil
}

// PublishTransaction publishes a transaction request and waits for response (RPC pattern)
func (a *Adapter) PublishTransaction(ctx context.Context, request models.TransactionRequest) (*models.TransactionResponse, error) {
	if a.closed {
		return nil, fmt.Errorf("connection is closed")
	}

	// Create temporary queue for response
	responseQueue, err := a.channel.QueueDeclare(
		"",    // name (auto-generated)
		false, // durable
		true,  // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare response queue: %w", err)
	}

	consumerTag := fmt.Sprintf("temp-consumer-%s-%d",
		utils.GenerateTransactionID(request.Username),
		time.Now().UnixNano())

	msgs, err := a.channel.Consume(
		responseQueue.Name, // queue name
		consumerTag,        // consumer tag
		true,               // auto-ack
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register consumer: %w", err)
	}

	defer func() {
		if cancelErr := a.channel.Cancel(consumerTag, false); cancelErr != nil {
			log.Printf("Warning: Failed to cancel consumer %s: %v", consumerTag, cancelErr)
		}
	}()

	// Generate correlation ID
	corrID := utils.GenerateTransactionID(request.Username)

	// Marshal request
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Publish message
	err = a.channel.PublishWithContext(
		ctx,
		a.config.Exchange,   // exchange
		a.config.RoutingKey, // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			Type:          strings.ToUpper(request.Action),
			ContentType:   "application/json",
			CorrelationId: corrID,
			ReplyTo:       responseQueue.Name,
			Body:          body,
			Expiration:    "60000", // 60 seconds
			Timestamp:     time.Now(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to publish message: %w", err)
	}

	// Wait for response
	select {
	case msg := <-msgs:
		if msg.CorrelationId == corrID {
			var response models.TransactionResponse
			if err := json.Unmarshal(msg.Body, &response); err != nil {
				return nil, fmt.Errorf("failed to unmarshal response: %w", err)
			}
			return &response, nil
		}
	case <-time.After(a.config.RequestTimeout):
		return nil, fmt.Errorf("request timeout")
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return nil, fmt.Errorf("no response received")
}

// Add this to adapter.go for high-volume testing
func (a *Adapter) PublishTransactionFireAndForget(ctx context.Context, request models.TransactionRequest) error {
	if a.closed {
		return fmt.Errorf("connection is closed")
	}

	// Marshal request
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Simple publish - NO RPC overhead
	err = a.channel.PublishWithContext(
		ctx,
		a.config.Exchange,   // exchange
		a.config.RoutingKey, // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			Type:        strings.ToUpper(request.Action),
			ContentType: "application/json",
			Body:        body,
			Timestamp:   time.Now(),
		},
	)

	return err
}

// PublishMessage publishes a message to a specific queue
func (a *Adapter) PublishMessage(ctx context.Context, queueName string, message interface{}) error {
	if a.closed {
		return fmt.Errorf("connection is closed")
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return a.channel.PublishWithContext(
		ctx,
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Timestamp:   time.Now(),
		},
	)
}

// StartConsumer starts consuming messages from the queue
func (a *Adapter) StartConsumer(handler ports.TransactionHandler) error {
	if a.closed {
		return fmt.Errorf("connection is closed")
	}

	msgs, err := a.channel.Consume(
		a.config.QueueName,   // queue
		a.config.ConsumerTag, // consumer
		false,                // auto-ack
		false,                // exclusive
		false,                // no-local
		false,                // no-wait
		nil,                  // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("RabbitMQ consumer started, waiting for messages on queue: %s", a.config.QueueName)

	go func() {
		for msg := range msgs {
			a.handleMessage(msg, handler)

			fmt.Printf("Received message: %s\n", msg.ConsumerTag)
		}
	}()

	return nil
}

// handleMessage processes individual messages
func (a *Adapter) handleMessage(msg amqp.Delivery, handler ports.TransactionHandler) {
	ctx := context.Background()

	var request models.TransactionRequest
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		a.sendErrorResponse(msg, "Invalid message format")
		msg.Ack(false)
		return
	}

	var response models.TransactionResponse

	// Handle based on message type
	switch strings.ToUpper(msg.Type) {
	case "DEPOSIT":
		response = handler.HandleDeposit(ctx, request)
	case "WITHDRAW":
		response = handler.HandleWithdraw(ctx, request)
	case "ADD_MEMBER":
		response = handler.HandleCreateMember(ctx, request)
	default:
		response = models.TransactionResponse{
			Status:  400,
			Message: "Unknown operation type",
		}
	}

	// Send response if ReplyTo is set
	if msg.ReplyTo != "" {
		a.sendResponse(msg, response)
	}

	msg.Ack(false)
}

// sendResponse sends response back to the requester
func (a *Adapter) sendResponse(msg amqp.Delivery, response models.TransactionResponse) {
	body, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	err = a.channel.Publish(
		"",          // exchange
		msg.ReplyTo, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: msg.CorrelationId,
			Body:          body,
		},
	)
	if err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

// sendErrorResponse sends error response
func (a *Adapter) sendErrorResponse(msg amqp.Delivery, errorMsg string) {
	response := models.TransactionResponse{
		Status:  400,
		Message: errorMsg,
	}
	a.sendResponse(msg, response)
}

// StopConsumer stops the consumer
func (a *Adapter) StopConsumer() error {
	if a.channel != nil {
		return a.channel.Cancel(a.config.ConsumerTag, false)
	}
	return nil
}

// IsConnected checks if the connection is active
func (a *Adapter) IsConnected() bool {
	return !a.closed && a.conn != nil && !a.conn.IsClosed()
}

// Close closes the connection
func (a *Adapter) Close() error {
	a.closed = true

	if a.channel != nil {
		if err := a.channel.Close(); err != nil {
			log.Printf("Failed to close channel: %v", err)
		}
	}

	if a.conn != nil {
		if err := a.conn.Close(); err != nil {
			log.Printf("Failed to close connection: %v", err)
		}
	}

	return nil
}
