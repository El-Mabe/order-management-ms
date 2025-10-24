package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"orders/internal/models"
	"orders/internal/repositories"

	"github.com/redis/go-redis/v9"
)

const (
	orderKeyPrefix = "order:"
)

type Repository interface {
	GetOrder(ctx context.Context, orderID string) (*models.Order, *repositories.RepositoryError)
	SetOrder(ctx context.Context, order *models.Order) *repositories.RepositoryError
	InvalidateOrder(ctx context.Context, orderID string) *repositories.RepositoryError
	Ping(ctx context.Context) *repositories.RepositoryError
	orderKey(orderID string) string
}

// CacheRepository implementa el repositorio de caché con Redis
type CacheRepository struct {
	client     *redis.Client
	defaultTTL time.Duration
}

// NewCacheRepository crea una nueva instancia del repositorio de caché
func NewCacheRepository(client *redis.Client, defaultTTL time.Duration) *CacheRepository {
	return &CacheRepository{
		client:     client,
		defaultTTL: defaultTTL,
	}
}

// GetOrder obtiene una orden del caché
func (r *CacheRepository) GetOrder(ctx context.Context, orderID string) (*models.Order, *repositories.RepositoryError) {
	key := r.orderKey(orderID)

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // No existe en caché (cache miss)
		}
		return nil, &repositories.RepositoryError{
			StatusCode: http.StatusNotFound,
			Cause:      "order not found",
			Message:    fmt.Sprintf("Order with ID %s not found", orderID),
		}
	}

	var order models.Order
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, &repositories.RepositoryError{
			StatusCode: http.StatusInternalServerError,
			Cause:      "failed to unmarshal order",
			Message:    fmt.Sprintf("Failed to unmarshal order with ID %s", orderID),
		}
	}

	return &order, nil
}

// SetOrder guarda una orden en el caché
func (r *CacheRepository) SetOrder(ctx context.Context, order *models.Order) *repositories.RepositoryError {
	key := r.orderKey(order.ID)

	data, err := json.Marshal(order)
	if err != nil {
		return &repositories.RepositoryError{
			StatusCode: http.StatusInternalServerError,
			Cause:      "failed to marshal order",
			Message:    fmt.Sprintf("Failed to marshal order with ID %s", order.ID),
		}
	}

	status := r.client.Set(ctx, key, data, r.defaultTTL)
	if err := status.Err(); err != nil {
		return &repositories.RepositoryError{
			StatusCode: http.StatusInternalServerError,
			Cause:      "failed to set order in cache",
			Message:    err.Error(),
		}
	}

	// Si todo salió bien, no hay error
	return nil
}

// InvalidateOrder invalida (elimina) una orden del caché
func (r *CacheRepository) InvalidateOrder(ctx context.Context, orderID string) *repositories.RepositoryError {
	key := r.orderKey(orderID)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return &repositories.RepositoryError{
			StatusCode: http.StatusInternalServerError,
			Cause:      "failed to delete order from cache",
			Message:    err.Error(),
		}
	}

	// Si todo salió bien, no hay error
	return nil
}

// Ping verifica la conexión con Redis
func (r *CacheRepository) Ping(ctx context.Context) *repositories.RepositoryError {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return &repositories.RepositoryError{
			StatusCode: http.StatusInternalServerError,
			Cause:      "failed to ping Redis",
			Message:    err.Error(),
		}
	}
	return nil
}

// orderKey genera la key de Redis para una orden
func (r *CacheRepository) orderKey(orderID string) string {
	return fmt.Sprintf("%s%s", orderKeyPrefix, orderID)
}
