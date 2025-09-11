package kafka

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

	"github.com/segmentio/kafka-go"
)

// Adapter implements the QueuePort interface for Apache Kafka
type Adapter struct {
	config   Config
	writer   *kafka.Writer
	readers  map[string]*kafka.Reader
	mutex    sync.RWMutex
	closed   bool
	consumer *Consumer
}

// Config holds Kafka configuration
type Config struct {
	Brokers        []string
	Topic          string
	ResponseTopic  string
	ConsumerGroup  string
	BatchSize      int
	BatchTimeout   time.Duration
	RequestTimeout time.Duration
}

// Consumer handles message consumption
type Consumer struct {
	reader  *kafka.Reader
	handler ports.TransactionHandler
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewAdapter creates a new Kafka adapter
func NewAdapter(brokers []string) (*Adapter, error) {
	config := Config{
		Brokers:        brokers,
		Topic:          "wallet-transactions",
		ResponseTopic:  "wallet-responses",
		ConsumerGroup:  "wallet-service",
		BatchSize:      100,
		BatchTimeout:   1 * time.Second,
		RequestTimeout: 30 * time.Second,
	}

	adapter := &Adapter{
		config:  config,
		readers: make(map[string]*kafka.Reader),
	}

	if err := adapter.setupWriter(); err != nil {
		return nil, fmt.Errorf("failed to setup Kafka writer: %w", err)
	}

	return adapter, nil
}

// setupWriter initializes the Kafka writer
func (a *Adapter) setupWriter() error {
	a.writer = &kafka.Writer{
		Addr:         kafka.TCP(a.config.Brokers...),
		Topic:        a.config.Topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    a.config.BatchSize,
		BatchTimeout: a.config.BatchTimeout,
		RequiredAcks: kafka.RequireOne,
		Compression:  kafka.Snappy,
	}

	return nil
}

// setupReader creates a Kafka reader for a specific topic
func (a *Adapter) setupReader(topic string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:     a.config.Brokers,
		Topic:       topic,
		GroupID:     a.config.ConsumerGroup,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		MaxWait:     1 * time.Second,
		StartOffset: kafka.LastOffset,
	})
}

// PublishTransaction publishes a transaction request and waits for response
func (a *Adapter) PublishTransaction(ctx context.Context, request models.TransactionRequest) (*models.TransactionResponse, error) {
	if a.closed {
		return nil, fmt.Errorf("connection is closed")
	}

	// Generate correlation ID and response topic
	correlationID := utils.GenerateTransactionID(request.Username)
	responseTopic := fmt.Sprintf("%s-%s", a.config.ResponseTopic, correlationID)

	// Create message headers
	headers := []kafka.Header{
		{Key: "correlation_id", Value: []byte(correlationID)},
		{Key: "reply_to", Value: []byte(responseTopic)},
		{Key: "type", Value: []byte(strings.ToUpper(request.Action))},
		{Key: "timestamp", Value: []byte(fmt.Sprintf("%d", time.Now().Unix()))},
	}

	// Marshal request
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create response reader
	responseReader := a.setupReader(responseTopic)
	defer responseReader.Close()

	// Start response listener
	responseCtx, responseCancel := context.WithTimeout(ctx, a.config.RequestTimeout)
	defer responseCancel()

	responseChan := make(chan models.TransactionResponse, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		for {
			select {
			case <-responseCtx.Done():
				errorChan <- responseCtx.Err()
				return
			default:
				msg, err := responseReader.FetchMessage(responseCtx)
				if err != nil {
					if err == context.DeadlineExceeded || err == context.Canceled {
						return
					}
					log.Printf("Failed to fetch response message: %v", err)
					continue
				}

				// Check correlation ID
				var msgCorrelationID string
				for _, header := range msg.Headers {
					if header.Key == "correlation_id" {
						msgCorrelationID = string(header.Value)
						break
					}
				}

				if msgCorrelationID == correlationID {
					var response models.TransactionResponse
					if err := json.Unmarshal(msg.Value, &response); err != nil {
						log.Printf("Failed to unmarshal response: %v", err)
						responseReader.CommitMessages(responseCtx, msg)
						continue
					}

					responseChan <- response
					responseReader.CommitMessages(responseCtx, msg)
					return
				}

				responseReader.CommitMessages(responseCtx, msg)
			}
		}
	}()

	// Publish request message
	message := kafka.Message{
		Key:     []byte(request.Username),
		Value:   body,
		Headers: headers,
		Time:    time.Now(),
	}

	if err := a.writer.WriteMessages(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to write message: %w", err)
	}

	// Wait for response
	select {
	case response := <-responseChan:
		return &response, nil
	case err := <-errorChan:
		return nil, fmt.Errorf("response error: %w", err)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (a *Adapter) PublishTransactionFireAndForget(ctx context.Context, request models.TransactionRequest) error {
	return nil
}

// PublishMessage publishes a message to a specific topic
func (a *Adapter) PublishMessage(ctx context.Context, topic string, message interface{}) error {
	if a.closed {
		return fmt.Errorf("connection is closed")
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create a temporary writer for the specific topic
	writer := &kafka.Writer{
		Addr:         kafka.TCP(a.config.Brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    1,
		RequiredAcks: kafka.RequireOne,
	}
	defer writer.Close()

	msg := kafka.Message{
		Value: body,
		Time:  time.Now(),
	}

	return writer.WriteMessages(ctx, msg)
}

// StartConsumer starts consuming messages from the topic
func (a *Adapter) StartConsumer(handler ports.TransactionHandler) error {
	if a.closed {
		return fmt.Errorf("connection is closed")
	}

	reader := a.setupReader(a.config.Topic)

	ctx, cancel := context.WithCancel(context.Background())

	a.consumer = &Consumer{
		reader:  reader,
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
	}

	log.Printf("Kafka consumer started, consuming from topic: %s", a.config.Topic)

	a.consumer.wg.Add(1)
	go func() {
		defer a.consumer.wg.Done()
		a.consumeMessages()
	}()

	return nil
}

// consumeMessages consumes messages from Kafka
func (a *Adapter) consumeMessages() {
	for {
		select {
		case <-a.consumer.ctx.Done():
			return
		default:
			msg, err := a.consumer.reader.FetchMessage(a.consumer.ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				log.Printf("Failed to fetch message: %v", err)
				continue
			}

			a.handleMessage(msg, a.consumer.handler)

			if err := a.consumer.reader.CommitMessages(a.consumer.ctx, msg); err != nil {
				log.Printf("Failed to commit message: %v", err)
			}
		}
	}
}

// handleMessage processes individual messages
func (a *Adapter) handleMessage(msg kafka.Message, handler ports.TransactionHandler) {
	ctx := context.Background()

	var request models.TransactionRequest
	if err := json.Unmarshal(msg.Value, &request); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		a.sendErrorResponse(msg, "Invalid message format")
		return
	}

	var response models.TransactionResponse
	var messageType string

	// Get message type from headers
	for _, header := range msg.Headers {
		if header.Key == "type" {
			messageType = string(header.Value)
			break
		}
	}

	// Handle based on message type
	switch strings.ToUpper(messageType) {
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

	// Send response if reply_to header is present
	var replyTo string
	for _, header := range msg.Headers {
		if header.Key == "reply_to" {
			replyTo = string(header.Value)
			break
		}
	}

	if replyTo != "" {
		a.sendResponse(msg, replyTo, response)
	}
}

// sendResponse sends response back to the requester
func (a *Adapter) sendResponse(originalMsg kafka.Message, replyTopic string, response models.TransactionResponse) {
	body, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	// Get correlation ID from original message
	var correlationID string
	for _, header := range originalMsg.Headers {
		if header.Key == "correlation_id" {
			correlationID = string(header.Value)
			break
		}
	}

	headers := []kafka.Header{
		{Key: "correlation_id", Value: []byte(correlationID)},
	}

	// Create temporary writer for response
	responseWriter := &kafka.Writer{
		Addr:         kafka.TCP(a.config.Brokers...),
		Topic:        replyTopic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    1,
		RequiredAcks: kafka.RequireOne,
	}
	defer responseWriter.Close()

	responseMsg := kafka.Message{
		Value:   body,
		Headers: headers,
		Time:    time.Now(),
	}

	if err := responseWriter.WriteMessages(context.Background(), responseMsg); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

// sendErrorResponse sends error response
func (a *Adapter) sendErrorResponse(originalMsg kafka.Message, errorMsg string) {
	response := models.TransactionResponse{
		Status:  400,
		Message: errorMsg,
	}

	var replyTo string
	for _, header := range originalMsg.Headers {
		if header.Key == "reply_to" {
			replyTo = string(header.Value)
			break
		}
	}

	if replyTo != "" {
		a.sendResponse(originalMsg, replyTo, response)
	}
}

// StopConsumer stops the consumer
func (a *Adapter) StopConsumer() error {
	if a.consumer != nil {
		a.consumer.cancel()
		a.consumer.wg.Wait()
		if err := a.consumer.reader.Close(); err != nil {
			log.Printf("Failed to close consumer reader: %v", err)
		}
	}
	return nil
}

// IsConnected checks if the connection is active (always true for Kafka)
func (a *Adapter) IsConnected() bool {
	return !a.closed
}

// Close closes all connections and resources
func (a *Adapter) Close() error {
	a.closed = true

	if err := a.StopConsumer(); err != nil {
		log.Printf("Failed to stop consumer: %v", err)
	}

	if a.writer != nil {
		if err := a.writer.Close(); err != nil {
			log.Printf("Failed to close writer: %v", err)
		}
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	for topic, reader := range a.readers {
		if err := reader.Close(); err != nil {
			log.Printf("Failed to close reader for topic %s: %v", topic, err)
		}
	}

	return nil
}
