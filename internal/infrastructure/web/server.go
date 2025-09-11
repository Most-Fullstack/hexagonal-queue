package web

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"hexagonal-queue/internal/application/ports"
	"hexagonal-queue/internal/application/usecases"

	"github.com/gin-gonic/gin"
)

// Server represents the HTTP server
type Server struct {
	port          string
	walletHandler *WalletHandler
	httpServer    *http.Server
}

// NewServer creates a new HTTP server
func NewServer(port string, walletUseCase *usecases.WalletUseCase, queuePort ports.QueuePort) *Server {
	walletHandler := NewWalletHandler(walletUseCase, queuePort)

	return &Server{
		port:          port,
		walletHandler: walletHandler,
	}
}

// setupRoutes configures all the routes
func (s *Server) setupRoutes() *gin.Engine {
	// Set Gin mode based on environment
	gin.SetMode(gin.ReleaseMode) // Can be changed based on config

	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(s.corsMiddleware())

	// Health check routes
	router.GET("/health", s.walletHandler.HealthHandler)
	router.GET("/health/queues", s.walletHandler.QueuesHealthHandler)
	router.GET("/metrics", s.walletHandler.GetMetricsHandler)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		wallet := v1.Group("/wallet")
		{
			wallet.POST("/deposit", s.walletHandler.DepositHandler)
			wallet.POST("/withdraw", s.walletHandler.WithdrawHandler)
			wallet.POST("/balance", s.walletHandler.CheckBalanceHandler)
			wallet.POST("/balances", s.walletHandler.CheckMultipleBalancesHandler)
			wallet.POST("/member", s.walletHandler.CreateMemberHandler)
			wallet.GET("/history/:username", s.walletHandler.GetTransactionHistoryHandler)

			queue := wallet.Group("/queue")
			{
				queue.POST("/deposit", s.walletHandler.QueueDepositHandler)
			}
		}

		test := v1.Group("/test")
		{
			test.POST("/queue", s.walletHandler.QueueTestHandler)
		}
	}

	return router
}

// corsMiddleware sets up CORS headers
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	router := s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:         ":" + s.port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting HTTP server on port %s", s.port)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down HTTP server...")
	return s.httpServer.Shutdown(ctx)
}
