package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"orders/internal/handlers"
	"orders/internal/models"
	"orders/internal/services"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// Mock del servicio
type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) CreateOrder(ctx context.Context, customerID string, items []models.OrderItem) (*models.Order, *services.ServiceError) {
	args := m.Called(ctx, customerID, items)
	return args.Get(0).(*models.Order), args.Error(1).(*services.ServiceError)
}

func (m *MockOrderService) GetOrderByID(ctx context.Context, orderID string) (*models.Order, *services.ServiceError) {
	args := m.Called(ctx, orderID)
	return args.Get(0).(*models.Order), args.Error(1).(*services.ServiceError)
}

func (m *MockOrderService) ListOrders(ctx context.Context, status, customerID string, page, limit int) ([]*models.Order, int64, *services.ServiceError) {
	args := m.Called(ctx, status, customerID, page, limit)
	return args.Get(0).([]*models.Order), args.Get(1).(int64), args.Error(2).(*services.ServiceError)
}

func (m *MockOrderService) UpdateOrderStatus(ctx context.Context, orderID string, newStatus models.OrderStatus) (*models.Order, *services.ServiceError) {
	args := m.Called(ctx, orderID, newStatus)
	return args.Get(0).(*models.Order), args.Error(1).(*services.ServiceError)
}

func TestOrderHandler_CreateOrder_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockOrderService)
	logger, _ := zap.NewDevelopment()
	handler := handlers.NewOrderHandler(mockService, logger, 10, 100)

	order := &models.Order{
		ID:          "order-123",
		CustomerID:  "123e4567-e89b-12d3-a456-426614174000",
		Status:      models.StatusNew,
		TotalAmount: 100,
	}

	mockService.On("CreateOrder", mock.Anything, order.CustomerID, mock.Anything).
		Return(order, (*services.ServiceError)(nil))

	body := `{"customerId":"123e4567-e89b-12d3-a456-426614174000","items":[{"sku":"ITEM-1","quantity":1,"price":100}]}`
	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.CreateOrder(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp models.Order
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, order.ID, resp.ID)
}

func TestOrderHandler_CreateOrder_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewOrderHandler(new(MockOrderService), zap.NewNop(), 10, 100)

	body := `{"customerId":"not-uuid"}`
	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.CreateOrder(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOrderHandler_GetOrder_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockOrderService)
	logger, _ := zap.NewDevelopment()
	handler := handlers.NewOrderHandler(mockService, logger, 10, 100)

	order := &models.Order{ID: "order-123"}
	mockService.On("GetOrderByID", mock.Anything, "order-123").Return(order, (*services.ServiceError)(nil))

	req := httptest.NewRequest(http.MethodGet, "/orders/order-123", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "order-123"}}

	handler.GetOrder(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOrderHandler_ListOrders_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockOrderService)
	logger, _ := zap.NewDevelopment()
	handler := handlers.NewOrderHandler(mockService, logger, 10, 100)

	orders := []*models.Order{
		{ID: "order-1"},
		{ID: "order-2"},
	}
	mockService.On("ListOrders", mock.Anything, "", "", 1, 10).Return(orders, int64(2), (*services.ServiceError)(nil))

	req := httptest.NewRequest(http.MethodGet, "/orders?page=1&limit=10", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.ListOrders(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOrderHandler_UpdateOrderStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockOrderService)
	logger, _ := zap.NewDevelopment()
	handler := handlers.NewOrderHandler(mockService, logger, 10, 100)

	order := &models.Order{ID: "order-123", Status: models.StatusInProgress}
	mockService.On("UpdateOrderStatus", mock.Anything, "order-123", models.StatusInProgress).Return(order, (*services.ServiceError)(nil))

	body := `{"status":"IN_PROGRESS"}`
	req := httptest.NewRequest(http.MethodPatch, "/orders/order-123/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "order-123"}}

	handler.UpdateOrderStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOrderHandler_GetOrder_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockOrderService)
	logger, _ := zap.NewDevelopment()
	handler := handlers.NewOrderHandler(mockService, logger, 10, 100)

	req := httptest.NewRequest(http.MethodGet, "/orders/", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: ""}} // ID vacío

	handler.GetOrder(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Order ID is required", resp["error"])
}

func TestOrderHandler_GetOrder_NonExistentID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockOrderService)
	logger, _ := zap.NewDevelopment()
	handler := handlers.NewOrderHandler(mockService, logger, 10, 100)

	// Simulamos que el servicio devuelve error (orden no encontrada)
	mockService.On("GetOrderByID", mock.Anything, "nonexistent-id").
		Return((*models.Order)(nil), &services.ServiceError{Message: "order not found"})

	req := httptest.NewRequest(http.MethodGet, "/orders/nonexistent-id", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "nonexistent-id"}}

	handler.GetOrder(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["error"], "Internal server error")
}

func TestOrderHandler_ListOrders_InvalidStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockOrderService)
	logger, _ := zap.NewDevelopment()
	handler := handlers.NewOrderHandler(mockService, logger, 10, 100)

	// status inválido que no existe en OrderStatus
	req := httptest.NewRequest(http.MethodGet, "/orders?status=INVALID_STATUS", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.ListOrders(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid status value", resp["error"])
}

func TestOrderHandler_UpdateOrderStatus_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockOrderService)
	logger, _ := zap.NewDevelopment()
	handler := handlers.NewOrderHandler(mockService, logger, 10, 100)

	// JSON inválido (missing "status")
	body := `{"wrongField":"IN_PROGRESS"}`
	req := httptest.NewRequest(http.MethodPatch, "/orders/order-123/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "order-123"}}

	handler.UpdateOrderStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid JSON format or missing required fields", resp["error"])
}

func TestOrderHandler_UpdateOrderStatus_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockOrderService)
	logger, _ := zap.NewDevelopment()
	handler := handlers.NewOrderHandler(mockService, logger, 10, 100)

	body := `{"status":"IN_PROGRESS"}`
	req := httptest.NewRequest(http.MethodPatch, "/orders//status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: ""}} // ID vacío

	handler.UpdateOrderStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Order ID is required", resp["error"])
}
