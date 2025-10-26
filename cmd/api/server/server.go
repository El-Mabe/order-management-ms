package server

import (
	"context"
	"time"

	"orders/cmd/api/config"
	"orders/internal/messages/kafka"
	"orders/internal/repositories/mongodb"
	redisrepo "orders/internal/repositories/redis"
	"orders/internal/services"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// Dependencies holds all shared resources used by the application,
// including database connections, external clients, and services.
type Dependencies struct {
	MongoClient   *mongo.Client
	MongoDB       *mongo.Database
	RedisClient   *redis.Client
	OrderService  services.OrderService
	KafkaProducer *kafka.Producer
}

// Initialize sets up and returns all core dependencies such as
// MongoDB, Redis, Kafka, and application services.
func Initialize(cfg *config.Config, log *zap.Logger) (*Dependencies, error) {
	// MongoDB setup
	mongoClient, err := ConnectMongoDB(cfg.MongoDB)
	if err != nil {
		return nil, err
	}
	mongoDB := mongoClient.Database(cfg.MongoDB.Database)

	orderRepo := mongodb.NewOrderRepository(mongoDB)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = orderRepo.CreateIndexes(ctx) // Ignore index creation errors during initialization

	// Redis setup
	redisClient := ConnectRedis(cfg.Redis)
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	// Kafka Producer setup (optional)
	var kafkaProducer *kafka.Producer
	if cfg.Kafka.EnableProducer {
		kafkaProducer = kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.TopicOrders, log)
	}

	// Repositories and services initialization
	cacheRepo := redisrepo.NewCacheRepository(redisClient, cfg.Redis.DefaultTTL)
	orderService := services.NewOrderService(orderRepo, cacheRepo, kafkaProducer, log)

	return &Dependencies{
		MongoClient:   mongoClient,
		MongoDB:       mongoDB,
		RedisClient:   redisClient,
		OrderService:  orderService,
		KafkaProducer: kafkaProducer,
	}, nil
}

// Close gracefully shuts down all active connections and releases resources.
func (d *Dependencies) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if d.MongoClient != nil {
		_ = d.MongoClient.Disconnect(ctx)
	}

	if d.RedisClient != nil {
		_ = d.RedisClient.Close()
	}

	if d.KafkaProducer != nil {
		_ = d.KafkaProducer.Close()
	}
}
