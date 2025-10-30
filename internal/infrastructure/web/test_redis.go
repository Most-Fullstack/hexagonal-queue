package web

import (
	"context"
	"fmt"
	"hexagonal-queue/internal/domain/models"
	"hexagonal-queue/internal/infrastructure/adapters/queue/redis"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RedisLoadMetrics struct {
	sync.Mutex
	PublishedCount   int64
	PublishErrors    int64
	StartTime        time.Time
	PublishLatencies []time.Duration
	StreamLengths    []int64
}

func (h *WalletHandler) testRedis(c *gin.Context) {
	const (
		numMessages  = 1000000                // Total messages to send
		workerPool   = 10                     // Number of concurrent workers
		batchSize    = 10000                  // Messages per batch
		testDuration = 600 * time.Second      // Max test duration
		messageRate  = 10000                  // Target messages per second
		batchDelay   = 100 * time.Millisecond // Delay between batches
	)

	redisAdapter := h.queuePort.(*redis.Adapter)
	start := time.Now()

	// Use buffered channels for better performance
	jobs := make(chan int, numMessages)
	done := make(chan bool, 1)

	// Metrics collection
	metrics := &RedisLoadMetrics{
		PublishedCount:   0,
		PublishErrors:    0,
		StartTime:        start,
		PublishLatencies: make([]time.Duration, 0, numMessages),
		StreamLengths:    make([]int64, 0),
	}

	fmt.Printf("🚀 Starting Redis Load Test\n")
	fmt.Printf("   Messages: %d | Workers: %d | Target Rate: %d msg/s\n\n",
		numMessages, workerPool, messageRate)

	// 1. Start worker pool for publishing
	var wg sync.WaitGroup
	for w := 0; w < workerPool; w++ {
		wg.Add(1)
		go h.publishRedisWorker(c.Request.Context(), jobs, metrics, &wg, redisAdapter)
	}

	// 2. Start Redis stream monitoring
	go h.monitorRedisStream(metrics, done, redisAdapter)

	// 3. Send jobs in batches
	go h.sendRedisJobsBatched(jobs, numMessages, batchSize, messageRate, testDuration, batchDelay, done)

	// 4. Wait for all jobs to be sent
	<-done
	fmt.Printf("📤 All %d messages queued. Waiting for workers to finish...\n", numMessages)

	// 5. Close jobs channel and wait for workers
	close(jobs)
	wg.Wait()

	// 6. Calculate and display final stats
	duration := time.Since(start)
	stats := h.calculateRedisStats(metrics, duration)

	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("📊 REDIS LOAD TEST RESULTS\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")
	fmt.Printf("Duration:             %s\n", stats.Duration)
	fmt.Printf("Messages Published:   %d\n", stats.MessagesPublished)
	fmt.Printf("Publish Errors:       %d\n", stats.PublishErrors)
	fmt.Printf("Success Rate:         %.2f%%\n", stats.SuccessRate)
	fmt.Printf("Publish Rate:         %.2f msg/sec\n", stats.PublishRate)
	fmt.Printf("Avg Publish Latency:  %v\n", stats.AvgPublishLatency)
	fmt.Printf("Max Publish Latency:  %v\n", stats.MaxPublishLatency)
	fmt.Printf("Avg Stream Length:    %.0f\n", stats.AvgStreamLength)
	fmt.Printf("Max Stream Length:    %d\n", stats.MaxStreamLength)
	fmt.Printf(strings.Repeat("=", 60) + "\n")

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Redis load test completed",
		"stats":   stats,
	})
}

func (h *WalletHandler) publishRedisWorker(ctx context.Context, jobs <-chan int, metrics *RedisLoadMetrics, wg *sync.WaitGroup, adapter *redis.Adapter) {
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
			Description: fmt.Sprintf("redis-load-test-%d", jobID),
		}

		_, err := adapter.PublishTransaction(ctx, request)
		latency := time.Since(publishStart)

		metrics.Lock()
		if err != nil {
			metrics.PublishErrors++
			// Log errors periodically
			if metrics.PublishErrors%100 == 0 {
				errorType := "Unknown"
				if strings.Contains(err.Error(), "timeout") {
					errorType = "Timeout"
				} else if strings.Contains(err.Error(), "connection") {
					errorType = "Connection"
				}
				fmt.Printf("⚠️  Redis errors: %d (type: %s)\n", metrics.PublishErrors, errorType)
			}
		} else {
			metrics.PublishedCount++
			metrics.PublishLatencies = append(metrics.PublishLatencies, latency)
		}
		metrics.Unlock()
	}
}

func (h *WalletHandler) sendRedisJobsBatched(jobs chan<- int, numMessages, batchSize, messageRate int, duration time.Duration, batchDelay time.Duration, done chan bool) {
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

			// Progress update every second
			if time.Since(lastUpdate) > 1*time.Second {
				fmt.Printf("📤 Queued: %d/%d messages (%.1f%%)\n",
					messagesSent, numMessages, float64(messagesSent)/float64(numMessages)*100)
				lastUpdate = time.Now()
			}

			time.Sleep(batchDelay)
		}
	}

	fmt.Printf("✅ All %d messages queued for publishing\n", numMessages)
	done <- true
}

func (h *WalletHandler) monitorRedisStream(metrics *RedisLoadMetrics, done chan bool, adapter *redis.Adapter) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastLength := int64(-1)
	lastLogTime := time.Now()
	maxObservedLength := int64(0)

	for {
		select {
		case <-done:
			return

		case <-ticker.C:
			streamInfo, err := adapter.GetStreamInfo()
			if err != nil {
				continue
			}

			currentLength := streamInfo.Length

			if currentLength > maxObservedLength {
				maxObservedLength = currentLength
			}

			metrics.Lock()
			publishedCount := metrics.PublishedCount
			errorCount := metrics.PublishErrors
			metrics.StreamLengths = append(metrics.StreamLengths, currentLength)
			metrics.Unlock()

			// Only log if stream length changed significantly or every 30 seconds
			shouldLog := lastLength == -1 ||
				abs64(currentLength-lastLength) > 100 ||
				time.Since(lastLogTime) > 30*time.Second

			if shouldLog {
				status := h.getRedisStreamStatus(currentLength, maxObservedLength)

				fmt.Printf("%s Stream: %d msgs | Published: %d | Errors: %d | Peak: %d\n",
					status, currentLength, publishedCount, errorCount, maxObservedLength)

				lastLength = currentLength
				lastLogTime = time.Now()
			}
		}
	}
}

func (h *WalletHandler) getRedisStreamStatus(currentLength, maxLength int64) string {
	// Redis can handle millions of messages in streams
	// These thresholds are conservative
	switch {
	case currentLength >= 1000000:
		return "🚨" // Very high
	case currentLength >= 500000:
		return "🔴" // High
	case currentLength >= 100000:
		return "🟡" // Warning
	case currentLength >= 50000:
		return "🟠" // Moderate
	case currentLength >= 10000:
		return "🔵" // Light load
	default:
		return "🟢" // Healthy
	}
}

type RedisLoadStats struct {
	Duration          string        `json:"duration"`
	MessagesPublished int64         `json:"messages_published"`
	PublishErrors     int64         `json:"publish_errors"`
	PublishRate       float64       `json:"publish_rate_per_sec"`
	AvgPublishLatency time.Duration `json:"avg_publish_latency"`
	MaxPublishLatency time.Duration `json:"max_publish_latency"`
	AvgStreamLength   float64       `json:"avg_stream_length"`
	MaxStreamLength   int64         `json:"max_stream_length"`
	SuccessRate       float64       `json:"success_rate_percent"`
}

func (h *WalletHandler) calculateRedisStats(metrics *RedisLoadMetrics, duration time.Duration) RedisLoadStats {
	metrics.Lock()
	defer metrics.Unlock()

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

	// Calculate stream length stats
	var avgStreamLength float64
	var maxStreamLength int64
	if len(metrics.StreamLengths) > 0 {
		var totalLength int64
		for _, length := range metrics.StreamLengths {
			totalLength += length
			if length > maxStreamLength {
				maxStreamLength = length
			}
		}
		avgStreamLength = float64(totalLength) / float64(len(metrics.StreamLengths))
	}

	totalMessages := metrics.PublishedCount + metrics.PublishErrors
	successRate := 0.0
	if totalMessages > 0 {
		successRate = float64(metrics.PublishedCount) / float64(totalMessages) * 100
	}

	publishRate := float64(metrics.PublishedCount) / duration.Seconds()

	return RedisLoadStats{
		Duration:          duration.String(),
		MessagesPublished: metrics.PublishedCount,
		PublishErrors:     metrics.PublishErrors,
		PublishRate:       publishRate,
		AvgPublishLatency: avgLatency,
		MaxPublishLatency: maxLatency,
		AvgStreamLength:   avgStreamLength,
		MaxStreamLength:   maxStreamLength,
		SuccessRate:       successRate,
	}
}

// Helper function for int64 absolute difference
func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
