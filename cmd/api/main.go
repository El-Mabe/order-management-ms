package main

import (
	"context"
	"fmt"
	"net/http"
	"orders/cmd/api/config"
	"orders/internal/handlers"
	"orders/internal/messages/kafka"
	"orders/internal/middlewares"

	"orders/internal/repositories/mongodb"
	redisrepo "orders/internal/repositories/redis"
	"orders/internal/services"
	"orders/pkg/logger"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// @title Orders Service API
// @version 1.0
// @description Microservicio de gestión de órdenes de entrega
// @host localhost:3000
// @BasePath /api/v1
func main() {
	// Cargar configuración
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Inicializar logger
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

	// Conectar MongoDB
	mongoClient, err := connectMongoDB(cfg.MongoDB)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Error("Failed to disconnect MongoDB", zap.Error(err))
		}
	}()

	mongoDB := mongoClient.Database(cfg.MongoDB.Database)
	log.Info("Connected to MongoDB")

	// Crear índices
	orderRepo := mongodb.NewOrderRepository(mongoDB)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := orderRepo.CreateIndexes(ctx); err != nil {
		log.Warn("Failed to create indexes", zap.Error(err))
	}
	cancel()
	log.Info("MongoDB indexes created")

	// Conectar Redis
	redisClient := connectRedis(cfg.Redis)
	defer redisClient.Close()

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("Failed to connect Redis", zap.Error(err))
	}
	cancel()
	log.Info("Connected to Redis")

	// Kafka Producer
	var kafkaProducer *kafka.Producer
	if cfg.Kafka.EnableProducer {
		kafkaProducer = kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.TopicOrders, log)
		defer kafkaProducer.Close()
		log.Info("Kafka producer initialized", zap.Strings("brokers", cfg.Kafka.Brokers))
	}

	// Repositorios
	cacheRepo := redisrepo.NewCacheRepository(redisClient, cfg.Redis.DefaultTTL)

	// Servicios
	orderService := services.NewOrderService(orderRepo, cacheRepo, kafkaProducer, log)

	// Handlers
	orderHandler := handlers.NewOrderHandler(orderService, log, cfg.App.DefaultPageSize, cfg.App.MaxPageSize)
	healthHandler := handlers.NewHealthHandler(mongoDB, redisClient)

	// Crear servidor
	router := gin.New()
	// Middlewares globales
	router.Use(gin.Recovery())
	router.Use(middlewares.RequestID())
	router.Use(middlewares.Security())
	router.Use(middlewares.CORS())
	router.Use(middlewares.Logger(log))
	router.Use(middlewares.ErrorHandler(log))

	router.GET("/health", healthHandler.CheckHealth)

	// Rutas
	// customhttp.SetupRoutesGin(router, orderHandler, healthHandler)
	api := router.Group("/api")
	{
		api.GET("/orders", orderHandler.ListOrders)
		api.POST("/orders", orderHandler.CreateOrder)
		api.GET("/orders/:id", orderHandler.GetOrder)
		api.PUT("/orders/:id", orderHandler.UpdateOrderStatus)
	}

	// Configurar servidor HTTP
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Iniciar servidor
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

// ---- Conexiones ----

func connectMongoDB(cfg config.MongoDBConfig) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectionTimeout)
	defer cancel()

	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetConnectTimeout(cfg.ConnectionTimeout)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}

func connectRedis(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.URL,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
}
