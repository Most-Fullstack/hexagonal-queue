package web

import (
	"context"
	"encoding/json"
	"fmt"
	"hexagonal-queue/internal/domain/models"
	"net/http"
	"strings"
	"sync"
	"time"
)

type RabbitMQLoadMetrics struct {
	sync.Mutex
	PublishedCount    int64
	PublishErrors     int64
	ConsumedCount     int64 // You'll need to track this from consumer
	QueueDepthSamples []int
	StartTime         time.Time
	PublishLatencies  []time.Duration
}

func (h *WalletHandler) startPublisher(ctx context.Context, rate int, duration time.Duration, metrics *RabbitMQLoadMetrics, done chan bool) {
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
					Amount:      float64(id % 1000), // Vary amounts,
					Channel:     "test",
					Description: fmt.Sprintf("test-%d", id),
				}

				_, err := h.queuePort.PublishTransaction(ctx, request)

				metrics.Lock()
				if err != nil {
					metrics.PublishErrors++
					// Only log different types of errors
					if strings.Contains(err.Error(), "503") || strings.Contains(err.Error(), "connection") {
						fmt.Printf("⚠️ RabbitMQ connection issue (error #%d)\n", metrics.PublishErrors)
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

func (h *WalletHandler) monitorQueueDepth(metrics *RabbitMQLoadMetrics, done chan bool) {
	ticker := time.NewTicker(5 * time.Second) // Changed from 1s to 5s
	defer ticker.Stop()

	// consecutive_errors := 0
	lastDepth := -1 // Track last depth to avoid spam
	lastLogTime := time.Now()

	for {
		select {
		case <-done:
			return

		case <-ticker.C:
			depth := h.getRabbitMQQueueDepth("wallet_queue")

			// if depth == -1 {
			// 	consecutive_errors++
			// 	if consecutive_errors == 1 { // Only log first error
			// 		fmt.Printf("⚠️  Cannot get queue depth from RabbitMQ Management API\n")
			// 	}
			// 	continue
			// }

			// consecutive_errors = 0

			metrics.Lock()
			metrics.QueueDepthSamples = append(metrics.QueueDepthSamples, depth)
			metrics.Unlock()

			// Only log if depth changed significantly
			if lastDepth == -1 || time.Since(lastLogTime) > 1*time.Second {
				status := "🟢" // Green - healthy
				if depth > 100 {
					status = "🔴" // Red - backing up
				} else if depth > 50 {
					status = "🟡" // Yellow - building up
				}

				fmt.Printf("%s Queue: %d msgs | Published: %d | Errors: %d\n",
					status, depth, metrics.PublishedCount, metrics.PublishErrors)
				lastDepth = depth
				lastLogTime = time.Now()
			}
		}
	}
}

// This requires RabbitMQ Management API
func (h *WalletHandler) getRabbitMQQueueDepth(queueName string) int {
	// Call RabbitMQ Management API
	// GET http://localhost:15672/api/queues/%2F/wallet_queue

	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest("GET",
		fmt.Sprintf("http://localhost:15672/api/queues/%%2F/%s", queueName), nil)
	req.SetBasicAuth("admin", "password123")

	resp, err := client.Do(req)
	if err != nil {
		return -1 // Error getting queue depth
	}
	defer resp.Body.Close()

	var queueInfo struct {
		Messages int `json:"messages"`
	}

	json.NewDecoder(resp.Body).Decode(&queueInfo)
	return queueInfo.Messages
}

type RabbitMQLoadStats struct {
	Duration          string        `json:"duration"`
	MessagesPublished int64         `json:"messages_published"`
	PublishErrors     int64         `json:"publish_errors"`
	PublishRate       float64       `json:"publish_rate_per_sec"`
	AvgPublishLatency time.Duration `json:"avg_publish_latency"`
	MaxPublishLatency time.Duration `json:"max_publish_latency"`
	AvgQueueDepth     float64       `json:"avg_queue_depth"`
	MaxQueueDepth     int           `json:"max_queue_depth"`
	SuccessRate       float64       `json:"success_rate_percent"`
}

func (h *WalletHandler) calculateRabbitMQStats(metrics *RabbitMQLoadMetrics) RabbitMQLoadStats {
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

	// Calculate queue depth stats
	var avgQueueDepth float64
	var maxQueueDepth int
	if len(metrics.QueueDepthSamples) > 0 {
		totalDepth := 0
		for _, depth := range metrics.QueueDepthSamples {
			totalDepth += depth
			if depth > maxQueueDepth {
				maxQueueDepth = depth
			}
		}
		avgQueueDepth = float64(totalDepth) / float64(len(metrics.QueueDepthSamples))
	}

	totalMessages := metrics.PublishedCount + metrics.PublishErrors
	successRate := float64(metrics.PublishedCount) / float64(totalMessages) * 100

	return RabbitMQLoadStats{
		Duration:          duration.String(),
		MessagesPublished: metrics.PublishedCount,
		PublishErrors:     metrics.PublishErrors,
		PublishRate:       float64(metrics.PublishedCount) / duration.Seconds(),
		AvgPublishLatency: avgLatency,
		MaxPublishLatency: maxLatency,
		AvgQueueDepth:     avgQueueDepth,
		MaxQueueDepth:     maxQueueDepth,
		SuccessRate:       successRate,
	}
}

// Optimized worker that processes jobs from channel
func (h *WalletHandler) publishWorker(ctx context.Context, jobs <-chan int, metrics *RabbitMQLoadMetrics, wg *sync.WaitGroup) {
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
			Description: fmt.Sprintf("load-test-%d", jobID),
		}

		_, err := h.queuePort.PublishTransaction(ctx, request)
		latency := time.Since(publishStart)

		metrics.Lock()
		if err != nil {
			metrics.PublishErrors++
			// Only log every 50 errors, and show different error types
			if metrics.PublishErrors%50 == 0 || (metrics.PublishErrors <= 10 && metrics.PublishErrors%5 == 0) {
				errorType := "Unknown"
				if strings.Contains(err.Error(), "504") {
					errorType = "Connection Limit"
				} else if strings.Contains(err.Error(), "timeout") {
					errorType = "Timeout"
				} else if strings.Contains(err.Error(), "no response") {
					errorType = "No Response"
				}
				fmt.Printf("🚨 %s errors: %d (type: %s)\n",
					errorType, metrics.PublishErrors, errorType)
			}
		} else {
			metrics.PublishedCount++
			metrics.PublishLatencies = append(metrics.PublishLatencies, latency)
		}
		metrics.Unlock()
	}
}

func (h *WalletHandler) sendJobsBatched(jobs chan<- int, numMessages, batchSize, messageRate int, duration time.Duration, done chan bool) {
	// batchDelay := time.Duration(float64(batchSize)/float64(messageRate)) * time.Second
	batchDelay := 2 * time.Second
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

			// Progress update every 1 seconds
			if time.Since(lastUpdate) > 1*time.Second {
				fmt.Printf("📤 Queued: %d/%d messages (%.1f%%)\n",
					messagesSent, numMessages, float64(messagesSent)/float64(numMessages)*100)
				lastUpdate = time.Now()
			}

			// fmt.Printf("📤 Queued: %d/%d messages (%.1f%%)\n",
			// 	messagesSent, numMessages, float64(messagesSent)/float64(numMessages)*100)

			time.Sleep(batchDelay)
		}
	}

	fmt.Printf("✅ All %d messages queued for publishing\n", numMessages)
	done <- true // Only signal when ALL messages queued
}
