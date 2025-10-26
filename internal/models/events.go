package models

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventOrderStatusChanged EventType = "ORDER_STATUS_CHANGED"
)

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

type EventMetadata struct {
	ChangedBy string `json:"changedBy"`
	Reason    string `json:"reason"`
}

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
