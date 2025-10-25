package server

import (
	"orders/cmd/api/config"
	"orders/internal/handlers"
	"orders/internal/middlewares"
	"orders/pkg/logger"

	"github.com/gin-gonic/gin"
)

// SetupRouter configura middlewares y rutas
func SetupRouter(deps *Dependencies, cfg *config.Config) *gin.Engine {
	router := gin.New()

	log := logger.Get()

	// Middlewares globales
	router.Use(
		gin.Recovery(),
		middlewares.RequestID(),
		middlewares.Security(),
		middlewares.CORS(),
		middlewares.Logger(log),
		middlewares.ErrorHandler(log),
	)

	// Handlers
	orderHandler := handlers.NewOrderHandler(deps.OrderService, log, cfg.App.DefaultPageSize, cfg.App.MaxPageSize)
	healthHandler := handlers.NewHealthHandler(deps.MongoDB, deps.RedisClient)

	// Rutas
	router.GET("/health", healthHandler.CheckHealth)
	api := router.Group("/api")
	{
		api.GET("/orders", orderHandler.ListOrders)
		api.POST("/orders", orderHandler.CreateOrder)
		api.GET("/orders/:id", orderHandler.GetOrder)
		api.PUT("/orders/:id", orderHandler.UpdateOrderStatus)
	}

	return router
}
