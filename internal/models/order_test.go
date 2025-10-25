package models_test

import (
	. "orders/internal/models"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestOrderStatus_IsValid(t *testing.T) {
	tests := []struct {
		status   OrderStatus
		expected bool
	}{
		{StatusNew, true},
		{StatusInProgress, true},
		{StatusDelivered, true},
		{StatusCancelled, true},
		{"INVALID", false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.status.IsValid(), "Status validation failed for %s", tt.status)
	}
}

func TestOrderItem_Subtotal(t *testing.T) {
	item := OrderItem{SKU: "ABC123", Quantity: 2, Price: 10.5}
	assert.Equal(t, 21.0, item.Subtotal())
}

func TestNewOrder_Success(t *testing.T) {
	customerID := uuid.New().String()
	items := []OrderItem{
		{SKU: "SKU123", Quantity: 2, Price: 100},
		{SKU: "SKU456", Quantity: 1, Price: 50},
	}

	order, err := NewOrder(customerID, items)
	assert.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, StatusNew, order.Status)
	assert.Equal(t, 250.0, order.TotalAmount)
	assert.Equal(t, 1, order.Version)
	assert.NotEmpty(t, order.ID)
	assert.WithinDuration(t, time.Now(), order.CreatedAt, time.Second)
}

func TestNewOrder_InvalidData(t *testing.T) {
	invalidUUID := "not-a-uuid"
	validItems := []OrderItem{{SKU: "SKU", Quantity: 1, Price: 10}}
	invalidItems := []OrderItem{}

	tests := []struct {
		name       string
		customerID string
		items      []OrderItem
		wantErr    error
	}{
		{"Empty customerID", "", validItems, ErrInvalidOrderData},
		{"Invalid UUID", invalidUUID, validItems, ErrInvalidOrderData},
		{"Empty items", uuid.New().String(), invalidItems, ErrInvalidOrderData},
		{"Invalid item data", uuid.New().String(), []OrderItem{{SKU: "SKU", Quantity: 0, Price: 10}}, ErrInvalidOrderData},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order, err := NewOrder(tt.customerID, tt.items)
			assert.Nil(t, order)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestOrder_CanTransitionTo(t *testing.T) {
	order := &Order{Status: StatusNew}

	assert.True(t, order.CanTransitionTo(StatusInProgress))
	assert.True(t, order.CanTransitionTo(StatusCancelled))
	assert.False(t, order.CanTransitionTo(StatusDelivered))

	order.Status = StatusInProgress
	assert.True(t, order.CanTransitionTo(StatusDelivered))
	assert.True(t, order.CanTransitionTo(StatusCancelled))
	assert.False(t, order.CanTransitionTo(StatusNew))

	order.Status = StatusDelivered
	assert.False(t, order.CanTransitionTo(StatusCancelled))
}

func TestOrder_UpdateStatus(t *testing.T) {
	order := &Order{
		Status:    StatusNew,
		Version:   1,
		UpdatedAt: time.Now(),
	}

	t.Run("Valid transition", func(t *testing.T) {
		err := order.UpdateStatus(StatusInProgress)
		assert.NoError(t, err)
		assert.Equal(t, StatusInProgress, order.Status)
		assert.Equal(t, 2, order.Version)
	})

	t.Run("Invalid transition", func(t *testing.T) {
		err := order.UpdateStatus(StatusNew)
		assert.ErrorIs(t, err, ErrInvalidStatusTransition)
	})

	t.Run("Invalid status", func(t *testing.T) {
		err := order.UpdateStatus("UNKNOWN")
		assert.ErrorIs(t, err, ErrInvalidOrderData)
	})
}

func TestOrder_CalculateTotalAmount(t *testing.T) {
	order := &Order{
		Items: []OrderItem{
			{SKU: "A", Quantity: 2, Price: 10},
			{SKU: "B", Quantity: 1, Price: 5},
		},
	}

	order.CalculateTotalAmount()
	assert.Equal(t, 25.0, order.TotalAmount)
}
