package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	StatusNew        OrderStatus = "NEW"
	StatusInProgress OrderStatus = "IN_PROGRESS"
	StatusDelivered  OrderStatus = "DELIVERED"
	StatusCancelled  OrderStatus = "CANCELLED"
)

var (
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrOrderNotFound           = errors.New("order not found")
	ErrInvalidOrderData        = errors.New("invalid order data")
	ErrVersionConflict         = errors.New("version conflict - order was modified")
)

type OrderStatus string

type Order struct {
	ID          string      `json:"orderId" bson:"_id"`
	CustomerID  string      `json:"customerId" bson:"customerId" validate:"required,uuid"`
	Status      OrderStatus `json:"status" bson:"status"`
	Items       []OrderItem `json:"items" bson:"items" validate:"required,min=1,max=100,dive"`
	TotalAmount float64     `json:"totalAmount" bson:"totalAmount"`
	Version     int         `json:"version" bson:"version"`
	CreatedAt   time.Time   `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt" bson:"updatedAt"`
}

// type OrderItem struct {
// 	ID        string
// 	ProductID string
// 	Quantity  int
// }

type OrderItem struct {
	SKU      string  `json:"sku" bson:"sku" validate:"required,min=3,max=50"`
	Quantity int     `json:"quantity" bson:"quantity" validate:"required,min=1,max=10000"`
	Price    float64 `json:"price" bson:"price" validate:"required,gt=0"`
}

// IsValid verifica si el estado es válido
func (s OrderStatus) IsValid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDelivered, StatusCancelled:
		return true
	}
	return false
}

// Subtotal calcula el subtotal del ítem
func (i OrderItem) Subtotal() float64 {
	return float64(i.Quantity) * i.Price
}

// NewOrder crea una nueva orden
func NewOrder(customerID string, items []OrderItem) (*Order, error) {
	if customerID == "" {
		return nil, ErrInvalidOrderData
	}

	if len(items) == 0 {
		return nil, ErrInvalidOrderData
	}

	// Validar UUID del cliente
	if _, err := uuid.Parse(customerID); err != nil {
		return nil, ErrInvalidOrderData
	}

	totalAmount := 0.0
	for _, item := range items {
		if item.Quantity <= 0 || item.Price <= 0 {
			return nil, ErrInvalidOrderData
		}
		totalAmount += item.Subtotal()
	}

	now := time.Now()
	return &Order{
		ID:          uuid.New().String(),
		CustomerID:  customerID,
		Status:      StatusNew,
		Items:       items,
		TotalAmount: totalAmount,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (o *Order) CanTransitionTo(newStatus OrderStatus) bool {
	switch o.Status {
	case StatusNew:
		return newStatus == StatusInProgress || newStatus == StatusCancelled
	case StatusInProgress:
		return newStatus == StatusDelivered || newStatus == StatusCancelled
	case StatusDelivered, StatusCancelled:
		return false // Estados finales
	}
	return false
}

// UpdateStatus actualiza el estado de la orden si la transición es válida
func (o *Order) UpdateStatus(newStatus OrderStatus) error {
	if !newStatus.IsValid() {
		return ErrInvalidOrderData
	}

	if !o.CanTransitionTo(newStatus) {
		return ErrInvalidStatusTransition
	}

	o.Status = newStatus
	o.UpdatedAt = time.Now()
	o.Version++

	return nil
}

// CalculateTotalAmount recalcula el monto total
func (o *Order) CalculateTotalAmount() {
	total := 0.0
	for _, item := range o.Items {
		total += item.Subtotal()
	}
	o.TotalAmount = total
}
