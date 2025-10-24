package mongodb

import (
	"context"
	"errors"
	"net/http"
	"orders/internal/models"
	"orders/internal/repositories"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	ordersCollection = "orders"
)

// OrderRepository implementa el repositorio de órdenes para MongoDB
type OrderRepository struct {
	db         *mongo.Database
	collection *mongo.Collection
}

type Repository interface {
	Create(ctx context.Context, order *models.Order) *repositories.RepositoryError
	FindByID(ctx context.Context, id string) (*models.Order, *repositories.RepositoryError)
	FindWithFilters(ctx context.Context, filters map[string]interface{}, page, limit int) ([]*models.Order, int64, *repositories.RepositoryError)
	Update(ctx context.Context, order *models.Order) *repositories.RepositoryError
}

// NewOrderRepository crea una nueva instancia del repositorio
func NewOrderRepository(db *mongo.Database) *OrderRepository {
	return &OrderRepository{
		db:         db,
		collection: db.Collection(ordersCollection),
	}
}

// Create inserta una nueva orden
func (r *OrderRepository) Create(ctx context.Context, order *models.Order) *repositories.RepositoryError {
	_, err := r.collection.InsertOne(ctx, order)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return &repositories.RepositoryError{
				StatusCode: http.StatusConflict,
				Cause:      "duplicate key error",
				Message:    "Order with the same ID already exists",
			}
		}
		return &repositories.RepositoryError{
			StatusCode: http.StatusInternalServerError,
			Cause:      err.Error(),
			Message:    "Failed to create order",
		}
	}
	return nil
}

// FindByID busca una orden por ID
func (r *OrderRepository) FindByID(ctx context.Context, id string) (*models.Order, *repositories.RepositoryError) {
	var order models.Order
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&order)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, &repositories.RepositoryError{
				StatusCode: http.StatusNotFound,
				Cause:      "order not found",
				Message:    "Order not found",
			}
		}
		return nil, &repositories.RepositoryError{
			StatusCode: http.StatusInternalServerError,
			Cause:      err.Error(),
			Message:    "Failed to find order",
		}
	}
	return &order, nil
}

// FindWithFilters busca órdenes con filtros y paginación
func (r *OrderRepository) FindWithFilters(ctx context.Context, filters map[string]interface{}, page, limit int) ([]*models.Order, int64, *repositories.RepositoryError) {
	// Construir filtro
	filter := bson.M{}
	if status, ok := filters["status"].(string); ok && status != "" {
		filter["status"] = status
	}
	if customerID, ok := filters["customerId"].(string); ok && customerID != "" {
		filter["customerId"] = customerID
	}

	// Contar total
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, &repositories.RepositoryError{
			StatusCode: http.StatusInternalServerError,
			Cause:      err.Error(),
			Message:    "Failed to count orders",
		}
	}

	// Calcular skip
	skip := (page - 1) * limit

	// Opciones de búsqueda
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(skip))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, &repositories.RepositoryError{
			StatusCode: http.StatusInternalServerError,
			Cause:      err.Error(),
			Message:    "Failed to find orders",
		}
	}
	defer cursor.Close(ctx)

	var orders []*models.Order
	if err = cursor.All(ctx, &orders); err != nil {
		return nil, 0, &repositories.RepositoryError{
			StatusCode: http.StatusInternalServerError,
			Cause:      err.Error(),
			Message:    "Failed to find orders",
		}
	}

	return orders, total, nil
}

// Update actualiza una orden con control de concurrencia optimista
func (r *OrderRepository) Update(ctx context.Context, order *models.Order) *repositories.RepositoryError {
	filter := bson.M{
		"_id":     order.ID,
		"version": order.Version - 1, // Verificar versión anterior
	}

	update := bson.M{
		"$set": bson.M{
			"status":     order.Status,
			"updated_at": order.UpdatedAt,
			"version":    order.Version,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return &repositories.RepositoryError{
			StatusCode: http.StatusInternalServerError,
			Cause:      err.Error(),
			Message:    "Failed to update order",
		}
	}

	if result.MatchedCount == 0 {
		// Verificar si existe la orden
		_, err := r.FindByID(ctx, order.ID)
		if err != nil {
			return &repositories.RepositoryError{
				StatusCode: http.StatusNotFound,
				Cause:      "order not found",
				Message:    "Order not found",
			}
		}
		// Existe pero versión no coincide
		return &repositories.RepositoryError{
			StatusCode: http.StatusConflict,
			Cause:      "version conflict",
			Message:    "Order was modified by another process",
		}
	}

	return nil
}

// CreateIndexes crea los índices necesarios
func (r *OrderRepository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "customerId", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "customerId", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}
