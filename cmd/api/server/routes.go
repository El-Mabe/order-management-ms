package server

import (
	"orders/cmd/api/config"
	"orders/internal/handlers"
	"orders/internal/middlewares"
	"orders/pkg/logger"

	_ "orders/cmd/api/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRouter initializes the Gin router, applies global middlewares,
// and registers all API routes.
func SetupRouter(deps *Dependencies, cfg *config.Config) *gin.Engine {
	router := gin.New()
	log := logger.Get()

	// Global middlewares
	router.Use(
		gin.Recovery(),
		middlewares.RequestID(),
		middlewares.Security(),
		middlewares.CORS(),
		middlewares.Logger(log),
		middlewares.ErrorHandler(log),
	)

	// Handlers initialization
	orderHandler := handlers.NewOrderHandler(deps.OrderService, log, cfg.App.DefaultPageSize, cfg.App.MaxPageSize)
	healthHandler := handlers.NewHealthHandler(deps.MongoDB, deps.RedisClient)

	// Routes definition
	router.GET("/health", healthHandler.CheckHealth)

	api := router.Group("/api")
	{
		api.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

		api.GET("/orders", orderHandler.ListOrders)
		api.POST("/orders", orderHandler.CreateOrder)
		api.GET("/orders/:id", orderHandler.GetOrder)
		api.PUT("/orders/:id", orderHandler.UpdateOrderStatus)

	}

	return router
}
