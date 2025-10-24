package services

import (
	"context"
	"net/http"
	"orders/internal/models"
	"orders/internal/repositories"
	"orders/internal/repositories/mongodb"
	"orders/internal/repositories/redis"

	// "orders/internal/repositories/redis"

	"go.uber.org/zap"
)

type ServiceError struct {
	Status            int           `json:"status"`
	Message           string        `json:"message"`
	Cause             []interface{} `json:"cause"`
	StatusDescription string        `json:"status_description,omitempty"`
}

type OrderService interface {
	CreateOrder(ctx context.Context, customerID string, items []models.OrderItem) (*models.Order, *ServiceError)
	GetOrderByID(ctx context.Context, orderID string) (*models.Order, *ServiceError)
	UpdateOrderStatus(ctx context.Context, orderID string, newStatus models.OrderStatus) (*models.Order, *ServiceError)
	ListOrders(ctx context.Context, status, customerID string, page, limit int) ([]*models.Order, int64, *ServiceError)
}

// CacheRepository define la interfaz del repositorio de caché
type CacheRepository interface {
	GetOrder(ctx context.Context, orderID string) (*models.Order, *repositories.RepositoryError)
	SetOrder(ctx context.Context, order *models.Order) *repositories.RepositoryError
	InvalidateOrder(ctx context.Context, orderID string) *repositories.RepositoryError
}

// EventPublisher define la interfaz del publicador de eventos
type EventPublisher interface {
	PublishOrderEvent(ctx context.Context, event *models.OrderEvent) error
}

type order struct {
	orderRepo      mongodb.Repository
	cacheRepo      redis.Repository
	eventPublisher EventPublisher
	logger         *zap.Logger
}

func NewOrderService(orderRepo mongodb.Repository, cacheRepo redis.Repository, eventPublisher EventPublisher, logger *zap.Logger) OrderService {
	return &order{
		orderRepo:      orderRepo,
		cacheRepo:      cacheRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// CreateOrder crea una nueva orden
func (s *order) CreateOrder(ctx context.Context, customerID string, items []models.OrderItem) (*models.Order, *ServiceError) {
	s.logger.Debug("Creating order",
		zap.String("customerId", customerID),
		zap.Int("itemsCount", len(items)),
	)

	// Crear orden en dominio
	order, err := models.NewOrder(customerID, items)
	if err != nil {
		s.logger.Error("Failed to create order entity",
			zap.Error(err),
			zap.String("customerId", customerID),
		)
		return nil, &ServiceError{
			Status:  http.StatusBadRequest,
			Message: "Invalid order data",
			Cause:   []interface{}{err.Error()},
		}
	}

	// Persistir en MongoDB
	if err := s.orderRepo.Create(ctx, order); err != nil {
		s.logger.Error("Failed to persist order",
			// zap.Error(err),
			zap.String("orderId", order.ID),
		)
		return nil, &ServiceError{
			Status:  err.StatusCode,
			Message: err.Message,
			Cause:   []interface{}{err.Cause},
		}
	}

	s.logger.Info("Order created successfully",
		zap.String("orderId", order.ID),
		zap.String("customerId", order.CustomerID),
		// zap.Float64("totalAmount", order.TotalAmount),
	)

	return order, nil
}

func (s *order) GetOrderByID(ctx context.Context, orderID string) (*models.Order, *ServiceError) {
	s.logger.Debug("Getting order by ID",
		zap.String("orderId", orderID),
	)

	// Intentar obtener del caché
	order, err := s.cacheRepo.GetOrder(ctx, orderID)
	if err != nil {
		s.logger.Warn("Cache error, falling back to database",
			// zap.Error(err),
			zap.String("orderId", orderID),
		)
	} else if order != nil {
		s.logger.Debug("Order found in cache",
			zap.String("orderId", orderID),
		)
		return order, nil
	}

	// Si no está en caché, buscar en MongoDB
	order, err = s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		s.logger.Error("Failed to get order from database",
			zap.String("Message", err.Message),
			zap.Int("StatusCode", err.StatusCode),
		)
		return nil, &ServiceError{
			Status:  err.StatusCode,
			Message: err.Message,
			Cause:   []interface{}{err.Cause},
		}
	}

	// Guardar en caché para futuras consultas
	if err := s.cacheRepo.SetOrder(ctx, order); err != nil {
		s.logger.Warn("Failed to cache order",
			zap.String("orderId", orderID),
		)
		// No retornar error, el caché es secundario
	}

	s.logger.Debug("Order retrieved from database",
		zap.String("orderId", orderID),
	)

	return order, nil

}

func (s *order) ListOrders(ctx context.Context, status, customerID string, page, limit int) ([]*models.Order, int64, *ServiceError) {
	s.logger.Debug("Listing orders",
		zap.String("status", status),
		zap.String("customerId", customerID),
		zap.Int("page", page),
		zap.Int("limit", limit),
	)

	filters := make(map[string]interface{})
	if status != "" {
		filters["status"] = status
	}
	if customerID != "" {
		filters["customerId"] = customerID
	}

	orders, total, err := s.orderRepo.FindWithFilters(ctx, filters, page, limit)
	if err != nil {
		s.logger.Error("Failed to list orders",
			zap.String("Message", err.Message),
			zap.Int("StatusCode", err.StatusCode),
			zap.String("Cause", err.Cause),
		)
		return nil, 0, &ServiceError{
			Status:  err.StatusCode,
			Message: err.Message,
			Cause:   []interface{}{err.Cause},
		}
	}

	s.logger.Debug("Orders listed successfully",
		zap.Int("count", len(orders)),
		zap.Int64("total", total),
	)

	return orders, total, nil
}

// UpdateOrderStatus actualiza el estado de una orden
func (s *order) UpdateOrderStatus(ctx context.Context, orderID string, newStatus models.OrderStatus) (*models.Order, *ServiceError) {
	s.logger.Debug("Updating order status",
		zap.String("orderId", orderID),
		zap.String("newStatus", string(newStatus)),
	)

	// Obtener orden actual
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, &ServiceError{
			Status:  err.StatusCode,
			Message: err.Message,
			Cause:   []interface{}{err.Cause},
		}
	}

	oldStatus := order.Status

	// Actualizar estado en dominio (con validación de transición)
	if err := order.UpdateStatus(newStatus); err != nil {
		s.logger.Warn("Invalid status transition",
			zap.Error(err),
			zap.String("orderId", orderID),
			zap.String("oldStatus", string(oldStatus)),
			zap.String("newStatus", string(newStatus)),
		)
		return nil, &ServiceError{
			Status:  http.StatusBadRequest,
			Message: "Invalid status transition",
			Cause:   []interface{}{err.Error()},
		}
	}

	// Persistir cambios en MongoDB
	if err := s.orderRepo.Update(ctx, order); err != nil {
		s.logger.Error("Failed to update order",
			zap.String("orderId", orderID),
		)
		return nil, &ServiceError{
			Status:  err.StatusCode,
			Message: err.Message,
			Cause:   []interface{}{err.Cause},
		}
	}

	// Invalidar caché
	if err := s.cacheRepo.InvalidateOrder(ctx, orderID); err != nil {
		s.logger.Warn("Failed to invalidate cache",
			zap.String("orderId", orderID),
		)
		// No retornar error, continuar con el flujo
	}

	// Publicar evento en Kafka
	event := models.NewOrderStatusChangedEvent(order.ID, order.CustomerID, oldStatus, newStatus)
	if err := s.eventPublisher.PublishOrderEvent(ctx, event); err != nil {
		s.logger.Error("Failed to publish event",
			zap.Error(err),
			zap.String("orderId", orderID),
			zap.String("eventId", event.EventID),
		)
		// No retornar error - el cambio ya se persistió
		// En producción, esto debería ir a un sistema de retry/DLQ
	}

	s.logger.Info("Order status updated successfully",
		zap.String("orderId", orderID),
		zap.String("oldStatus", string(oldStatus)),
		zap.String("newStatus", string(newStatus)),
	)

	return order, nil
}
