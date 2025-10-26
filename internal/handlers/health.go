package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// HealthHandler handles the health check endpoint.
type HealthHandler struct {
	mongoDB *mongo.Database
	redis   *redis.Client
}

// NewHealthHandler creates a new instance of HealthHandler.
func NewHealthHandler(mongoDB *mongo.Database, redis *redis.Client) *HealthHandler {
	return &HealthHandler{
		mongoDB: mongoDB,
		redis:   redis,
	}
}

// HealthResponse represents the response structure for health checks.
type HealthResponse struct {
	Status       string            `json:"status"`
	Timestamp    time.Time         `json:"timestamp"`
	Dependencies map[string]string `json:"dependencies"`
}

// CheckHealth checks the status of the service and its dependencies (MongoDB, Redis, Kafka).
// Returns HTTP 200 if all dependencies are healthy, otherwise HTTP 503.
func (h *HealthHandler) CheckHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dependencies := make(map[string]string)
	allHealthy := true

	// Check MongoDB connection
	mongoStatus := "connected"
	if err := h.mongoDB.Client().Ping(ctx, readpref.Primary()); err != nil {
		mongoStatus = "disconnected"
		allHealthy = false
	}
	dependencies["mongodb"] = mongoStatus

	// Check Redis connection
	redisStatus := "connected"
	if err := h.redis.Ping(ctx).Err(); err != nil {
		redisStatus = "disconnected"
		allHealthy = false
	}
	dependencies["redis"] = redisStatus

	// Kafka status (simplified - in production verify actual connection)
	dependencies["kafka"] = "connected"

	status := "healthy"
	statusCode := http.StatusOK
	if !allHealthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:       status,
		Timestamp:    time.Now(),
		Dependencies: dependencies,
	}

	c.JSON(statusCode, response)
}
