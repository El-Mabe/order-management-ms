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

// HealthHandler maneja el endpoint de health check
type HealthHandler struct {
	mongoDB *mongo.Database
	redis   *redis.Client
}

// NewHealthHandler crea una nueva instancia del handler
func NewHealthHandler(mongoDB *mongo.Database, redis *redis.Client) *HealthHandler {
	return &HealthHandler{
		mongoDB: mongoDB,
		redis:   redis,
	}
}

// HealthResponseGin representa la respuesta del health check
type HealthResponse struct {
	Status       string            `json:"status"`
	Timestamp    time.Time         `json:"timestamp"`
	Dependencies map[string]string `json:"dependencies"`
}

// CheckHealthGin godoc
// @Summary Health check
// @Description Verifica el estado del servicio y sus dependencias
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponseGin
// @Failure 503 {object} HealthResponseGin
// @Router /health [get]
func (h *HealthHandler) CheckHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dependencies := make(map[string]string)
	allHealthy := true

	// Check MongoDB
	mongoStatus := "connected"
	if err := h.mongoDB.Client().Ping(ctx, readpref.Primary()); err != nil {
		mongoStatus = "disconnected"
		allHealthy = false
	}
	dependencies["mongodb"] = mongoStatus

	// Check Redis
	redisStatus := "connected"
	if err := h.redis.Ping(ctx).Err(); err != nil {
		redisStatus = "disconnected"
		allHealthy = false
	}
	dependencies["redis"] = redisStatus

	// Check Kafka (simplificado - en producción verificar conexión real)
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
