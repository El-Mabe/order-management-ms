package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"orders/cmd/api/config"
	"orders/cmd/api/server"
	"orders/pkg/logger"

	"go.uber.org/zap"
)

// @title Orders Service API
// @version 1.0
// @description Microservice for delivery order management
// @host localhost:3000
// @BasePath /api
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(cfg.Logging.Level, cfg.Logging.Format); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	log := logger.Get()
	log.Info("Starting Orders Service",
		zap.String("environment", cfg.Server.Environment),
		zap.String("port", cfg.Server.Port),
	)

	// Initialize dependencies (MongoDB, Redis, Kafka, repositories, services, handlers)
	deps, err := server.Initialize(cfg, log)
	if err != nil {
		log.Fatal("Failed to initialize dependencies", zap.Error(err))
	}
	defer deps.Close()

	// Setup routes and middlewares
	router := server.SetupRouter(deps, cfg)

	// Configure HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a separate goroutine
	go func() {
		log.Info("Server starting", zap.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Error("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server stopped")
}
