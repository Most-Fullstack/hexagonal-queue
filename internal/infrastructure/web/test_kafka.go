package web

import (
	"context"
	"encoding/json"
	"fmt"
	"hexagonal-queue/internal/domain/models"
	"hexagonal-queue/pkg/utils"
	"net/http"
	"strings"
	"sync"
	"time"
)

type KafkaLoadMetrics struct {
	sync.Mutex
	PublishedCount   int64
	PublishErrors    int64
	ConsumedCount    int64
	LagSamples       []int64
	StartTime        time.Time
	PublishLatencies []time.Duration
}

// Start Kafka publisher with specified rate
func (h *WalletHandler) startKafkaPublisher(ctx context.Context, rate int, duration time.Duration, metrics *KafkaLoadMetrics, done chan bool) {
	ticker := time.NewTicker(time.Second / time.Duration(rate))
	defer ticker.Stop()

	timeout := time.After(duration)
	messageID := 0

	for {
		select {
		case <-timeout:
			done <- true
			return

		case <-ticker.C:
			go func(id int) {
				publishStart := time.Now()

				request := models.TransactionRequest{
					ParentToken: "admin",
					Token:       "1",
					Username:    "admin",
					Action:      "DEPOSIT",
					TypeName:    "DEPOSIT",
					Amount:      float64(id % 1000),
					Channel:     "test",
					Description: fmt.Sprintf("kafka-test-%d", id),
				}

				_, err := h.queuePort.PublishTransaction(ctx, request)

				metrics.Lock()
				if err != nil {
					metrics.PublishErrors++
					if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "connection") {
						fmt.Printf("⚠️ Kafka connection issue (error #%d)\n", metrics.PublishErrors)
					}
				} else {
					metrics.PublishedCount++
					metrics.PublishLatencies = append(metrics.PublishLatencies, time.Since(publishStart))
				}
				metrics.Unlock()
			}(messageID)

			messageID++
		}
	}
}

// Monitor Kafka consumer lag as a proxy for queue depth
func (h *WalletHandler) monitorKafkaLag(brokers []string, topic string, groupID string, metrics *KafkaLoadMetrics, done chan bool) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastLag := int64(-1)
	lastLogTime := time.Now()
	maxObservedLag := int64(0)

	for {
		select {
		case <-done:
			return

		case <-ticker.C:
			lag := h.getKafkaConsumerLag(brokers, topic, groupID)

			if lag > maxObservedLag {
				maxObservedLag = lag
			}

			metrics.Lock()
			publishedCount := int(metrics.PublishedCount)
			errorCount := int(metrics.PublishErrors)
			metrics.LagSamples = append(metrics.LagSamples, lag)
			metrics.Unlock()

			// Log if lag changed significantly or every 30 seconds
			shouldLog := lastLag == -1 ||
				utils.Abs(int(lag-lastLag)) > 100 ||
				time.Since(lastLogTime) > 30*time.Second

			if shouldLog {
				status := h.getKafkaHealthStatus(lag, maxObservedLag)

				fmt.Printf("%s Lag: %d msgs | Published: %d | Errors: %d | Peak Lag: %d\n",
					status, lag, publishedCount, errorCount, maxObservedLag)

				lastLag = lag
				lastLogTime = time.Now()
			}
		}
	}
}

// Get Kafka consumer lag - simplified version
func (h *WalletHandler) getKafkaConsumerLag(brokers []string, topic string, groupID string) int64 {
	// For simplified monitoring, return 0
	// Real lag monitoring requires Kafka Admin API or external tools
	return 0
}

// Get Kafka health status based on lag
func (h *WalletHandler) getKafkaHealthStatus(currentLag, maxLag int64) string {
	var status string
	switch {
	case currentLag >= 100000:
		status = "🚨" // Critical
	case currentLag >= 50000:
		status = "🔴" // High
	case currentLag >= 10000:
		status = "🟡" // Warning
	case currentLag >= 5000:
		status = "🟠" // Caution
	case currentLag >= 1000:
		status = "🔵" // Moderate
	default:
		status = "🟢" // Healthy
	}

	return status
}

type KafkaLoadStats struct {
	Duration          string        `json:"duration"`
	MessagesPublished int64         `json:"messages_published"`
	PublishErrors     int64         `json:"publish_errors"`
	PublishRate       float64       `json:"publish_rate_per_sec"`
	AvgPublishLatency time.Duration `json:"avg_publish_latency"`
	MaxPublishLatency time.Duration `json:"max_publish_latency"`
	AvgLag            float64       `json:"avg_lag"`
	MaxLag            int64         `json:"max_lag"`
	SuccessRate       float64       `json:"success_rate_percent"`
}

func (h *WalletHandler) calculateKafkaStats(metrics *KafkaLoadMetrics) KafkaLoadStats {
	metrics.Lock()
	defer metrics.Unlock()

	duration := time.Since(metrics.StartTime)

	// Calculate latency stats
	var avgLatency, maxLatency time.Duration
	if len(metrics.PublishLatencies) > 0 {
		var totalLatency time.Duration
		for _, lat := range metrics.PublishLatencies {
			totalLatency += lat
			if lat > maxLatency {
				maxLatency = lat
			}
		}
		avgLatency = totalLatency / time.Duration(len(metrics.PublishLatencies))
	}

	// Calculate lag stats
	var avgLag float64
	var maxLag int64
	if len(metrics.LagSamples) > 0 {
		var totalLag int64
		for _, lag := range metrics.LagSamples {
			totalLag += lag
			if lag > maxLag {
				maxLag = lag
			}
		}
		avgLag = float64(totalLag) / float64(len(metrics.LagSamples))
	}

	totalMessages := metrics.PublishedCount + metrics.PublishErrors
	successRate := float64(0)
	if totalMessages > 0 {
		successRate = float64(metrics.PublishedCount) / float64(totalMessages) * 100
	}

	return KafkaLoadStats{
		Duration:          duration.String(),
		MessagesPublished: metrics.PublishedCount,
		PublishErrors:     metrics.PublishErrors,
		PublishRate:       float64(metrics.PublishedCount) / duration.Seconds(),
		AvgPublishLatency: avgLatency,
		MaxPublishLatency: maxLatency,
		AvgLag:            avgLag,
		MaxLag:            maxLag,
		SuccessRate:       successRate,
	}
}

// Kafka worker that processes jobs from channel
func (h *WalletHandler) kafkaPublishWorker(ctx context.Context, jobs <-chan int, metrics *KafkaLoadMetrics, wg *sync.WaitGroup) {
	defer wg.Done()

	for jobID := range jobs {
		publishStart := time.Now()

		request := models.TransactionRequest{
			ParentToken: "admin",
			Token:       "1",
			Username:    "admin",
			Action:      "DEPOSIT",
			TypeName:    "DEPOSIT",
			Amount:      1.00,
			Channel:     "test",
			Description: fmt.Sprintf("kafka-load-test-%d", jobID),
		}

		err := h.queuePort.PublishTransactionFireAndForget(ctx, request)
		latency := time.Since(publishStart)

		metrics.Lock()
		if err != nil {
			metrics.PublishErrors++
			if metrics.PublishErrors%50 == 0 || (metrics.PublishErrors <= 10 && metrics.PublishErrors%5 == 0) {
				errorType := "Unknown"
				if strings.Contains(err.Error(), "timeout") {
					errorType = "Timeout"
				} else if strings.Contains(err.Error(), "connection") {
					errorType = "Connection"
				} else if strings.Contains(err.Error(), "leader") {
					errorType = "Leader Not Available"
				}

				fmt.Printf("🚨 Kafka %s errors: %d\n", errorType, metrics.PublishErrors)
			}
		} else {
			metrics.PublishedCount++
			metrics.PublishLatencies = append(metrics.PublishLatencies, latency)
		}
		metrics.Unlock()
	}
}

// Send jobs in batches for Kafka load testing
func (h *WalletHandler) sendKafkaJobsBatched(jobs chan<- int, numMessages, batchSize int, duration time.Duration, batchDelay time.Duration, done chan bool) {
	timeout := time.After(duration)

	messagesSent := 0
	lastUpdate := time.Now()

	for messagesSent < numMessages {
		select {
		case <-timeout:
			fmt.Printf("⏰ Time limit reached. Sent %d/%d messages\n", messagesSent, numMessages)
			done <- true
			return

		default:
			batchEnd := messagesSent + batchSize
			if batchEnd > numMessages {
				batchEnd = numMessages
			}

			for i := messagesSent; i < batchEnd; i++ {
				select {
				case jobs <- i:
					messagesSent++
				case <-timeout:
					done <- true
					return
				}
			}

			// Progress update every 1 second
			if time.Since(lastUpdate) > 1*time.Second {
				fmt.Printf("📤 Queued: %d/%d messages (%.1f%%)\n",
					messagesSent, numMessages, float64(messagesSent)/float64(numMessages)*100)
				lastUpdate = time.Now()
			}

			time.Sleep(batchDelay)
		}
	}

	fmt.Printf("✅ All %d messages queued for Kafka publishing\n", numMessages)
	done <- true
}

// HTTP handler for Kafka load test
func (h *WalletHandler) KafkaLoadTest(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	rate := 10_000 // messages per second
	duration := 10 * time.Second
	numWorkers := 3
	numMessages := 1_000_000
	batchSize := 10000

	if rateParam := r.URL.Query().Get("rate"); rateParam != "" {
		fmt.Sscanf(rateParam, "%d", &rate)
	}
	if durationParam := r.URL.Query().Get("duration"); durationParam != "" {
		var durationSec int
		fmt.Sscanf(durationParam, "%d", &durationSec)
		duration = time.Duration(durationSec) * time.Second
	}
	if workersParam := r.URL.Query().Get("workers"); workersParam != "" {
		fmt.Sscanf(workersParam, "%d", &numWorkers)
	}
	if messagesParam := r.URL.Query().Get("messages"); messagesParam != "" {
		fmt.Sscanf(messagesParam, "%d", &numMessages)
	}

	ctx := r.Context()
	metrics := &KafkaLoadMetrics{
		StartTime: time.Now(),
	}

	fmt.Printf("\n🚀 Starting Kafka Load Test\n")
	fmt.Printf("   Rate: %d msg/sec\n", rate)
	fmt.Printf("   Duration: %v\n", duration)
	fmt.Printf("   Workers: %d\n", numWorkers)
	fmt.Printf("   Total Messages: %d\n", numMessages)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Create channels
	jobs := make(chan int, batchSize*2)
	done := make(chan bool)
	monitorDone := make(chan bool)

	// Start monitor
	go h.monitorKafkaLag([]string{"localhost:9092"}, "wallet-transactions", "wallet-service", metrics, monitorDone)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go h.kafkaPublishWorker(ctx, jobs, metrics, &wg)
	}

	// Send jobs
	batchDelay := time.Duration(float64(batchSize)/float64(rate)) * time.Second
	go h.sendKafkaJobsBatched(jobs, numMessages, batchSize, duration, batchDelay, done)

	// Wait for completion
	<-done
	close(jobs)
	wg.Wait()
	monitorDone <- true

	// Calculate and return stats
	stats := h.calculateKafkaStats(metrics)

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 Kafka Load Test Results:")
	fmt.Printf("   Duration: %s\n", stats.Duration)
	fmt.Printf("   Messages Published: %d\n", stats.MessagesPublished)
	fmt.Printf("   Publish Errors: %d\n", stats.PublishErrors)
	fmt.Printf("   Success Rate: %.2f%%\n", stats.SuccessRate)
	fmt.Printf("   Publish Rate: %.2f msg/sec\n", stats.PublishRate)
	fmt.Printf("   Avg Latency: %v\n", stats.AvgPublishLatency)
	fmt.Printf("   Max Latency: %v\n", stats.MaxPublishLatency)
	fmt.Printf("   Avg Lag: %.0f messages\n", stats.AvgLag)
	fmt.Printf("   Max Lag: %d messages\n", stats.MaxLag)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
