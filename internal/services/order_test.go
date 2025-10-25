package services_test

import (
	"context"
	"orders/internal/models"
	"orders/internal/repositories"
	"orders/internal/services"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockOrderRepository es un mock del repositorio de órdenes
type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) Create(ctx context.Context, order *models.Order) *repositories.RepositoryError {
	args := m.Called(ctx, order)
	if v := args.Get(0); v != nil {
		return v.(*repositories.RepositoryError)
	}
	return nil
}

func (m *MockOrderRepository) FindByID(ctx context.Context, id string) (*models.Order, *repositories.RepositoryError) {
	args := m.Called(ctx, id)
	var order *models.Order
	if v := args.Get(0); v != nil {
		order = v.(*models.Order)
	}

	var repoErr *repositories.RepositoryError
	if v := args.Get(1); v != nil {
		repoErr = v.(*repositories.RepositoryError)
	}

	return order, repoErr
}

func (m *MockOrderRepository) FindWithFilters(ctx context.Context, filters map[string]interface{}, page, limit int) ([]*models.Order, int64, *repositories.RepositoryError) {
	args := m.Called(ctx, filters, page, limit)

	var orders []*models.Order
	if v := args.Get(0); v != nil {
		orders = v.([]*models.Order)
	}

	var total int64
	if v := args.Get(1); v != nil {
		total = v.(int64)
	}

	var repoErr *repositories.RepositoryError
	if v := args.Get(2); v != nil {
		repoErr = v.(*repositories.RepositoryError)
	}

	return orders, total, repoErr
}

func (m *MockOrderRepository) Update(ctx context.Context, order *models.Order) *repositories.RepositoryError {
	args := m.Called(ctx, order)

	if v := args.Get(0); v != nil {
		return v.(*repositories.RepositoryError)
	}
	return nil
}

// MockCacheRepository es un mock del repositorio de caché
type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) GetOrder(ctx context.Context, orderID string) (*models.Order, *repositories.RepositoryError) {
	args := m.Called(ctx, orderID)

	var order *models.Order
	if v := args.Get(0); v != nil {
		order = v.(*models.Order)
	}

	var repoErr *repositories.RepositoryError
	if v := args.Get(1); v != nil {
		repoErr = v.(*repositories.RepositoryError)
	}

	return order, repoErr
}

func (m *MockCacheRepository) SetOrder(ctx context.Context, order *models.Order) *repositories.RepositoryError {
	args := m.Called(ctx, order)

	if v := args.Get(0); v != nil {
		return v.(*repositories.RepositoryError)
	}
	return nil
}

func (m *MockCacheRepository) InvalidateOrder(ctx context.Context, orderID string) *repositories.RepositoryError {
	args := m.Called(ctx, orderID)
	if v := args.Get(0); v != nil {
		return v.(*repositories.RepositoryError)
	}
	return nil
}

// MockEventPublisher es un mock del publicador de eventos
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) PublishOrderEvent(ctx context.Context, event *models.OrderEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func TestOrderService_CreateOrder_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockOrderRepository)
	mockCache := new(MockCacheRepository)
	mockPublisher := new(MockEventPublisher)
	logger, _ := zap.NewDevelopment()

	service := services.NewOrderService(mockRepo, mockCache, mockPublisher, logger)

	customerID := "123e4567-e89b-12d3-a456-426614174000"
	items := []models.OrderItem{
		{SKU: "LAPTOP-001", Quantity: 2, Price: 999.99},
	}

	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Order")).Return(nil)

	// Act
	order, err := service.CreateOrder(context.Background(), customerID, items)

	// Assert
	assert.Nil(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, customerID, order.CustomerID)
	assert.Equal(t, models.StatusNew, order.Status)
	assert.Equal(t, 1999.98, order.TotalAmount)
	mockRepo.AssertExpectations(t)
}

func TestOrderService_CreateOrder_InvalidCustomerID(t *testing.T) {
	// Arrange
	mockRepo := new(MockOrderRepository)
	mockCache := new(MockCacheRepository)
	mockPublisher := new(MockEventPublisher)
	logger, _ := zap.NewDevelopment()

	service := services.NewOrderService(mockRepo, mockCache, mockPublisher, logger)

	items := []models.OrderItem{
		{SKU: "LAPTOP-001", Quantity: 1, Price: 999.99},
	}

	// Act
	order, err := service.CreateOrder(context.Background(), "invalid-uuid", items)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, order)
	assert.Equal(t, 400, err.Status)
}

func TestOrderService_GetOrderByID_FromCache(t *testing.T) {
	// Arrange
	mockRepo := new(MockOrderRepository)
	mockCache := new(MockCacheRepository)
	mockPublisher := new(MockEventPublisher)
	logger, _ := zap.NewDevelopment()

	service := services.NewOrderService(mockRepo, mockCache, mockPublisher, logger)

	expectedOrder := &models.Order{
		ID:         "order-123",
		CustomerID: "customer-456",
		Status:     models.StatusNew,
	}

	mockCache.On("GetOrder", mock.Anything, "order-123").Return(expectedOrder, nil)

	// Act
	order, err := service.GetOrderByID(context.Background(), "order-123")

	// Assert
	assert.Nil(t, err)
	assert.Equal(t, expectedOrder, order)
	mockCache.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "FindByID")
}

func TestOrderService_GetOrderByID_FromDatabase(t *testing.T) {
	// Arrange
	mockRepo := new(MockOrderRepository)
	mockCache := new(MockCacheRepository)
	mockPublisher := new(MockEventPublisher)
	logger, _ := zap.NewDevelopment()

	service := services.NewOrderService(mockRepo, mockCache, mockPublisher, logger)

	expectedOrder := &models.Order{
		ID:         "order-123",
		CustomerID: "customer-456",
		Status:     models.StatusNew,
	}

	mockCache.On("GetOrder", mock.Anything, "order-123").Return(nil, nil)
	mockRepo.On("FindByID", mock.Anything, "order-123").Return(expectedOrder, nil)
	mockCache.On("SetOrder", mock.Anything, expectedOrder).Return(nil)

	// Act
	order, err := service.GetOrderByID(context.Background(), "order-123")

	// Assert
	assert.Nil(t, err)
	assert.Equal(t, expectedOrder, order)
	mockCache.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestOrderService_GetOrderByID_NotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockOrderRepository)
	mockCache := new(MockCacheRepository)
	mockPublisher := new(MockEventPublisher)
	logger, _ := zap.NewDevelopment()

	service := services.NewOrderService(mockRepo, mockCache, mockPublisher, logger)

	mockCache.On("GetOrder", mock.Anything, "order-999").Return(nil, nil)
	notFoundErr := &repositories.RepositoryError{
		StatusCode: 404,
		Message:    "Order not found",
	}
	mockRepo.On("FindByID", mock.Anything, "order-999").Return(nil, notFoundErr)

	// Act
	order, err := service.GetOrderByID(context.Background(), "order-999")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, order)
	assert.Equal(t, 404, err.Status)
}

func TestOrderService_UpdateOrderStatus_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockOrderRepository)
	mockCache := new(MockCacheRepository)
	mockPublisher := new(MockEventPublisher)
	logger, _ := zap.NewDevelopment()

	service := services.NewOrderService(mockRepo, mockCache, mockPublisher, logger)

	existingOrder := &models.Order{
		ID:         "order-123",
		CustomerID: "customer-456",
		Status:     models.StatusNew,
		Version:    1,
	}

	mockRepo.On("FindByID", mock.Anything, "order-123").Return(existingOrder, nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Order")).Return(nil)
	mockCache.On("InvalidateOrder", mock.Anything, "order-123").Return(nil)
	mockPublisher.On("PublishOrderEvent", mock.Anything, mock.AnythingOfType("*models.OrderEvent")).Return(nil)

	// Act
	order, err := service.UpdateOrderStatus(context.Background(), "order-123", models.StatusInProgress)

	// Assert
	assert.Nil(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, models.StatusInProgress, order.Status)
	assert.Equal(t, 2, order.Version)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

func TestOrderService_UpdateOrderStatus_InvalidTransition(t *testing.T) {
	// Arrange
	mockRepo := new(MockOrderRepository)
	mockCache := new(MockCacheRepository)
	mockPublisher := new(MockEventPublisher)
	logger, _ := zap.NewDevelopment()

	service := services.NewOrderService(mockRepo, mockCache, mockPublisher, logger)

	existingOrder := &models.Order{
		ID:         "order-123",
		CustomerID: "customer-456",
		Status:     models.StatusDelivered,
		Version:    1,
	}

	mockRepo.On("FindByID", mock.Anything, "order-123").Return(existingOrder, nil)

	// Act
	order, err := service.UpdateOrderStatus(context.Background(), "order-123", models.StatusInProgress)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, order)
	assert.Equal(t, 400, err.Status)
	mockRepo.AssertNotCalled(t, "Update")
	mockPublisher.AssertNotCalled(t, "PublishOrderEvent")
}

func TestOrderService_UpdateOrderStatus_VersionConflict(t *testing.T) {
	// Arrange
	mockRepo := new(MockOrderRepository)
	mockCache := new(MockCacheRepository)
	mockPublisher := new(MockEventPublisher)
	logger, _ := zap.NewDevelopment()

	service := services.NewOrderService(mockRepo, mockCache, mockPublisher, logger)

	existingOrder := &models.Order{
		ID:         "order-123",
		CustomerID: "customer-456",
		Status:     models.StatusNew,
		Version:    1,
	}

	mockRepo.On("FindByID", mock.Anything, "order-123").Return(existingOrder, nil)
	conflictErr := &repositories.RepositoryError{
		StatusCode: 409,
		Message:    "Version conflict",
	}
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Order")).Return(conflictErr)

	// Act
	order, err := service.UpdateOrderStatus(context.Background(), "order-123", models.StatusInProgress)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, order)
	assert.Equal(t, 409, err.Status)

}

func TestOrderService_ListOrders_Success_NoFilters(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	mockRepo := new(MockOrderRepository)
	mockCache := new(MockCacheRepository)
	mockPublisher := new(MockEventPublisher)
	service := services.NewOrderService(mockRepo, mockCache, mockPublisher, logger)

	ordersMock := []*models.Order{
		{ID: "1", CustomerID: "customer-1", Status: models.StatusNew},
		{ID: "2", CustomerID: "customer-1", Status: models.StatusInProgress},
	}
	totalMock := int64(2)

	mockRepo.On("FindWithFilters", ctx, map[string]interface{}{}, 1, 10).
		Return(ordersMock, totalMock, nil).Once()

	orders, total, err := service.ListOrders(ctx, "", "", 1, 10)
	assert.Nil(t, err)
	assert.Len(t, orders, 2)
	assert.Equal(t, int64(2), total)
	mockRepo.AssertExpectations(t)
}

func TestOrderService_ListOrders_Success_WithFilters(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	mockRepo := new(MockOrderRepository)
	mockCache := new(MockCacheRepository)
	mockPublisher := new(MockEventPublisher)
	service := services.NewOrderService(mockRepo, mockCache, mockPublisher, logger)

	ordersMock := []*models.Order{
		{ID: "1", CustomerID: "customer-1", Status: models.StatusNew},
	}
	totalMock := int64(1)

	filters := map[string]interface{}{
		"status":     string(models.StatusNew),
		"customerId": "customer-1",
	}

	mockRepo.On("FindWithFilters", ctx, filters, 1, 5).
		Return(ordersMock, totalMock, nil).Once()

	orders, total, err := service.ListOrders(ctx, string(models.StatusNew), "customer-1", 1, 5)
	assert.Nil(t, err)
	assert.Len(t, orders, 1)
	assert.Equal(t, int64(1), total)
	mockRepo.AssertExpectations(t)
}

func TestOrderService_ListOrders_RepoError(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	mockRepo := new(MockOrderRepository)
	mockCache := new(MockCacheRepository)
	mockPublisher := new(MockEventPublisher)
	service := services.NewOrderService(mockRepo, mockCache, mockPublisher, logger)

	repoErr := &repositories.RepositoryError{
		StatusCode: 500,
		Message:    "DB error",
		Cause:      "connection failed",
	}

	mockRepo.On("FindWithFilters", ctx, map[string]interface{}{}, 1, 10).
		Return(nil, int64(0), repoErr).Once()

	orders, total, err := service.ListOrders(ctx, "", "", 1, 10)
	assert.Nil(t, orders)
	assert.Equal(t, int64(0), total)
	assert.NotNil(t, err)
	assert.Equal(t, 500, err.Status)
	assert.Equal(t, "DB error", err.Message)
	assert.Equal(t, []interface{}{"connection failed"}, err.Cause)
	mockRepo.AssertExpectations(t)
}

func TestOrderService_ListOrders_Pagination(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	mockRepo := new(MockOrderRepository)
	mockCache := new(MockCacheRepository)
	mockPublisher := new(MockEventPublisher)
	service := services.NewOrderService(mockRepo, mockCache, mockPublisher, logger)

	ordersMock := []*models.Order{
		{ID: "1", CustomerID: "customer-1", Status: models.StatusNew},
		{ID: "2", CustomerID: "customer-1", Status: models.StatusInProgress},
	}
	totalMock := int64(2)

	mockRepo.On("FindWithFilters", ctx, map[string]interface{}{}, 2, 3).
		Return(ordersMock, totalMock, nil).Once()

	orders, total, err := service.ListOrders(ctx, "", "", 2, 3)
	assert.Nil(t, err)
	assert.Len(t, orders, 2)
	assert.Equal(t, int64(2), total)
	mockRepo.AssertExpectations(t)
}
