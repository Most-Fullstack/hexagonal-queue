package web

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"hexagonal-queue/internal/application/ports"
	"hexagonal-queue/internal/application/usecases"
	"hexagonal-queue/internal/domain/models"

	"github.com/gin-gonic/gin"
)

// WalletHandler handles wallet-related HTTP requests
type WalletHandler struct {
	walletUseCase *usecases.WalletUseCase
	queuePort     ports.QueuePort
}

// NewWalletHandler creates a new wallet handler
func NewWalletHandler(walletUseCase *usecases.WalletUseCase, queuePort ports.QueuePort) *WalletHandler {
	return &WalletHandler{
		walletUseCase: walletUseCase,
		queuePort:     queuePort,
	}
}

// DepositHandler handles deposit requests
// POST /api/v1/wallet/deposit
func (h *WalletHandler) DepositHandler(c *gin.Context) {
	var request models.TransactionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  400,
			"message": "Invalid request format",
			"error":   err.Error(),
		})
		return
	}

	// Set action and get authorization header
	request.Action = "DEPOSIT"
	request.ParentToken = c.GetHeader("Authorization")

	if request.ParentToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  401,
			"message": "Authorization header is required",
		})
		return
	}

	response := h.walletUseCase.HandleDeposit(c.Request.Context(), request)

	statusCode := http.StatusOK
	if response.Status >= 400 {
		statusCode = int(response.Status)
	}

	c.JSON(statusCode, response)
}

func (h *WalletHandler) QueueDepositHandler(c *gin.Context) {
	var request models.TransactionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  400,
			"message": "Invalid request format",
			"error":   err.Error(),
		})
		return
	}

	response, err := h.queuePort.PublishTransaction(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  500,
			"message": "Failed to publish transaction",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *WalletHandler) QueueTestRabbitmqDepositHandler(c *gin.Context) {
	const (
		numMessages = 100000 // Total messages to send
		workerPool  = 3      // Increase workers!
		batchSize   = 100    // Messages per batch
	)

	// Get test parameters
	testDuration := 600 * time.Second // Reduce time, increase workers
	messageRate := 100                // Increase rate

	start := time.Now()

	// Use buffered channels for better performance
	jobs := make(chan int, numMessages)
	done := make(chan bool, 1)

	// Metrics collection
	metrics := &RabbitMQLoadMetrics{
		PublishedCount:    0,
		PublishErrors:     0,
		ConsumedCount:     0,
		QueueDepthSamples: make([]int, 0, int(testDuration.Seconds())),
		StartTime:         start,
		PublishLatencies:  make([]time.Duration, 0, numMessages),
	}

	// 1. Start worker pool for publishing
	var wg sync.WaitGroup
	for w := 0; w < workerPool; w++ {
		wg.Add(1)
		go h.publishWorker(c.Request.Context(), jobs, metrics, &wg)
	}

	// 2. Start queue depth monitoring
	go h.monitorQueueDepth(metrics, done)

	// 3. Send jobs in batches with rate limiting
	go h.sendJobsBatched(jobs, numMessages, batchSize, messageRate, testDuration, done)

	// 4. Wait for ALL jobs to be sent
	select {
	case <-done:
		fmt.Printf("📤 All %d messages queued. Waiting for workers to finish...\n", numMessages)
	case <-time.After(testDuration + 10*time.Second):
		fmt.Printf("⏰ Publishing timed out. Waiting for workers to finish...\n")
		close(done)
	}

	// 5. CRITICAL: Wait for ALL workers to complete processing
	close(jobs) // Signal workers to stop
	wg.Wait()   // Wait for all workers to finish

	// 6. Stop monitoring
	close(done)

	// 7. Calculate final metrics
	finalStats := h.calculateRabbitMQStats(metrics)

	// 8. Show COMPLETE results
	fmt.Printf("\n🏁 FINAL RESULTS:\n")
	fmt.Printf("📊 Total Messages: %d\n", numMessages)
	fmt.Printf("✅ Successful: %d (%.1f%%)\n", finalStats.MessagesPublished,
		float64(finalStats.MessagesPublished)/float64(numMessages)*100)
	fmt.Printf("❌ Failed: %d (%.1f%%)\n", finalStats.PublishErrors,
		float64(finalStats.PublishErrors)/float64(numMessages)*100)
	fmt.Printf("⏱️  Duration: %s\n", finalStats.Duration)
	fmt.Printf("🚀 Rate: %.1f msg/sec\n", finalStats.PublishRate)
	fmt.Printf("📈 Avg Latency: %v\n", finalStats.AvgPublishLatency)
	fmt.Printf("📊 Queue Depth: avg=%.1f, max=%d\n",
		finalStats.AvgQueueDepth, finalStats.MaxQueueDepth)

	c.JSON(http.StatusOK, gin.H{
		"message": "RabbitMQ Load Test Results",
		"config": gin.H{
			"total_messages": numMessages,
			"worker_pool":    workerPool,
			"batch_size":     batchSize,
			"target_rate":    messageRate,
			"test_duration":  testDuration.String(),
		},
		"stats": finalStats,
	})
}

// WithdrawHandler handles withdraw requests
// POST /api/v1/wallet/withdraw
func (h *WalletHandler) WithdrawHandler(c *gin.Context) {
	var request models.TransactionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  400,
			"message": "Invalid request format",
			"error":   err.Error(),
		})
		return
	}

	// Set action and get authorization header
	request.Action = "WITHDRAW"
	request.ParentToken = c.GetHeader("Authorization")

	if request.ParentToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  401,
			"message": "Authorization header is required",
		})
		return
	}

	response := h.walletUseCase.HandleWithdraw(c.Request.Context(), request)

	statusCode := http.StatusOK
	if response.Status >= 400 {
		statusCode = int(response.Status)
	}

	c.JSON(statusCode, response)
}

// CheckBalanceHandler handles balance check requests
// POST /api/v1/wallet/balance
func (h *WalletHandler) CheckBalanceHandler(c *gin.Context) {
	type BalanceRequest struct {
		Username string `json:"username" binding:"required"`
	}

	var request BalanceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  400,
			"message": "Invalid request format",
			"error":   err.Error(),
		})
		return
	}

	parentToken := c.GetHeader("Authorization")
	if parentToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  401,
			"message": "Authorization header is required",
		})
		return
	}

	response := h.walletUseCase.HandleBalanceCheck(c.Request.Context(), request.Username, parentToken)

	statusCode := http.StatusOK
	if response.Status >= 400 {
		statusCode = int(response.Status)
	}

	c.JSON(statusCode, response)
}

// CheckMultipleBalancesHandler handles multiple balance check requests
// POST /api/v1/wallet/balances
func (h *WalletHandler) CheckMultipleBalancesHandler(c *gin.Context) {
	type MultipleBalanceRequest struct {
		Usernames []string `json:"usernames" binding:"required"`
	}

	var request MultipleBalanceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  400,
			"message": "Invalid request format",
			"error":   err.Error(),
		})
		return
	}

	parentToken := c.GetHeader("Authorization")
	if parentToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  401,
			"message": "Authorization header is required",
		})
		return
	}

	response := h.walletUseCase.GetMultipleBalances(c.Request.Context(), request.Usernames, parentToken)

	statusCode := http.StatusOK
	if response.Status >= 400 {
		statusCode = int(response.Status)
	}

	c.JSON(statusCode, response)
}

// CreateMemberHandler handles member creation requests
// POST /api/v1/wallet/member
func (h *WalletHandler) CreateMemberHandler(c *gin.Context) {
	var request models.TransactionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  400,
			"message": "Invalid request format",
			"error":   err.Error(),
		})
		return
	}

	// Set action and get authorization header
	request.Action = "ADD_MEMBER"
	request.ParentToken = c.GetHeader("Authorization")

	if request.ParentToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  401,
			"message": "Authorization header is required",
		})
		return
	}

	response := h.walletUseCase.HandleCreateMember(c.Request.Context(), request)

	statusCode := http.StatusCreated
	if response.Status >= 400 {
		statusCode = int(response.Status)
	}

	c.JSON(statusCode, response)
}

// HealthHandler handles health check requests
// GET /health
func (h *WalletHandler) HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "hexagonal-queue-wallet",
		"version": "1.0.0",
	})
}

// QueuesHealthHandler handles queue health check requests
// GET /health/queues
func (h *WalletHandler) QueuesHealthHandler(c *gin.Context) {
	// This could be extended to actually check queue connectivity
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"queues": map[string]interface{}{
			"rabbitmq": gin.H{"status": "available"},
			"kafka":    gin.H{"status": "available"},
			"redis":    gin.H{"status": "available"},
		},
	})
}

// QueueTestHandler handles queue testing requests - sends a test message
// POST /api/v1/test/queue
func (h *WalletHandler) QueueTestHandler(c *gin.Context) {
	type QueueTestRequest struct {
		Provider string  `json:"provider" binding:"required"` // rabbitmq, kafka, redis
		Username string  `json:"username" binding:"required"`
		Action   string  `json:"action" binding:"required"` // deposit, withdraw, add_member
		Amount   float64 `json:"amount"`
	}

	var request QueueTestRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  400,
			"message": "Invalid request format",
			"error":   err.Error(),
		})
		return
	}

	// Create transaction request
	txRequest := models.TransactionRequest{
		ParentToken: "test-parent-token",
		Token:       "test-user-token",
		Username:    request.Username,
		Action:      request.Action,
		TypeName:    request.Action,
		Amount:      request.Amount,
		Channel:     "test-api",
		Description: "Queue test transaction",
		Lang:        "en",
	}

	var response models.TransactionResponse

	// Handle based on action
	switch request.Action {
	case "deposit":
		response = h.walletUseCase.HandleDeposit(c.Request.Context(), txRequest)
	case "withdraw":
		response = h.walletUseCase.HandleWithdraw(c.Request.Context(), txRequest)
	case "add_member":
		response = h.walletUseCase.HandleCreateMember(c.Request.Context(), txRequest)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  400,
			"message": "Invalid action. Must be: deposit, withdraw, add_member",
		})
		return
	}

	statusCode := http.StatusOK
	if response.Status >= 400 {
		statusCode = int(response.Status)
	}

	result := gin.H{
		"test_provider":  request.Provider,
		"test_action":    request.Action,
		"queue_response": response,
	}

	c.JSON(statusCode, result)
}

// GetTransactionHistoryHandler handles transaction history requests
// GET /api/v1/wallet/history/:username?limit=10
func (h *WalletHandler) GetTransactionHistoryHandler(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  400,
			"message": "Username is required",
		})
		return
	}

	parentToken := c.GetHeader("Authorization")
	if parentToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  401,
			"message": "Authorization header is required",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100 // Maximum limit
	}

	// This would require additional method in use case to get transaction history
	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"status":  200,
		"message": "Success",
		"data": gin.H{
			"username":     username,
			"limit":        limit,
			"transactions": []gin.H{
				// Placeholder - would be actual transaction history
			},
		},
	})
}

// GetMetricsHandler handles metrics requests for monitoring
// GET /metrics
func (h *WalletHandler) GetMetricsHandler(c *gin.Context) {
	// This could be extended to return actual metrics
	c.JSON(http.StatusOK, gin.H{
		"service": "hexagonal-queue-wallet",
		"metrics": gin.H{
			"total_transactions": 0,
			"active_connections": 0,
			"queue_health": gin.H{
				"rabbitmq": "connected",
				"kafka":    "connected",
				"redis":    "connected",
			},
		},
	})
}
