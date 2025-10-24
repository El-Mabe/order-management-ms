package models

import (
	"time"

	"github.com/google/uuid"
)

// EventType representa el tipo de evento
type EventType string

const (
	EventOrderStatusChanged EventType = "ORDER_STATUS_CHANGED"
)

// OrderEvent representa un evento relacionado con una orden
type OrderEvent struct {
	EventID    string        `json:"eventId"`
	EventType  EventType     `json:"eventType"`
	OrderID    string        `json:"orderId"`
	CustomerID string        `json:"customerId"`
	OldStatus  OrderStatus   `json:"oldStatus"`
	NewStatus  OrderStatus   `json:"newStatus"`
	Timestamp  time.Time     `json:"timestamp"`
	Metadata   EventMetadata `json:"metadata"`
}

// EventMetadata contiene metadatos adicionales del evento
type EventMetadata struct {
	ChangedBy string `json:"changedBy"`
	Reason    string `json:"reason"`
}

// NewOrderStatusChangedEvent crea un nuevo evento de cambio de estado
func NewOrderStatusChangedEvent(orderID, customerID string, oldStatus, newStatus OrderStatus) *OrderEvent {
	return &OrderEvent{
		EventID:    uuid.New().String(),
		EventType:  EventOrderStatusChanged,
		OrderID:    orderID,
		CustomerID: customerID,
		OldStatus:  oldStatus,
		NewStatus:  newStatus,
		Timestamp:  time.Now(),
		Metadata: EventMetadata{
			ChangedBy: "system",
			Reason:    "status_update",
		},
	}
}
