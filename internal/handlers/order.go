package handlers

import (
	"math"
	"net/http"
	"orders/internal/models"
	"orders/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type ErrorResponse struct {
	Code    int    `json:"code"`    // Código HTTP o interno
	Message string `json:"message"` // Mensaje de error
}

type OrderHandler struct {
	service         services.OrderService
	validator       *validator.Validate
	logger          *zap.Logger
	maxPageSize     int
	defaultPageSize int
}

func NewOrderHandler(service services.OrderService, logger *zap.Logger, defaultPageSize, maxPageSize int) *OrderHandler {
	return &OrderHandler{
		service:         service,
		validator:       validator.New(),
		logger:          logger,
		maxPageSize:     maxPageSize,
		defaultPageSize: defaultPageSize,
	}
}

type CreateOrderRequest struct {
	CustomerID string             `json:"customerId" binding:"required,uuid"`
	Items      []models.OrderItem `json:"items" binding:"required,min=1,max=100,dive"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=NEW IN_PROGRESS DELIVERED CANCELLED"`
}

type PaginationResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

type ListOrdersResponse struct {
	Orders     []*models.Order    `json:"orders"`
	Pagination PaginationResponse `json:"pagination"`
}

// CreateOrder godoc
// @Summary Create a new order
// @Description Creates a new delivery order
// @Tags orders
// @Accept json
// @Produce json
// @Param order body CreateOrderRequest true "Order data"
// @Success 201 {object} models.Order
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	requestID := getRequestID(c)
	ctx := c.Request.Context()

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", zap.Error(err), zap.String("requestId", requestID))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	order, err := h.service.CreateOrder(ctx, req.CustomerID, req.Items)
	if err != nil {
		h.logger.Error("Failed to create order", zap.String("requestId", requestID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, order)
}

// GetOrder godoc
// @Summary Get order by ID
// @Description Retrieves a specific order by its ID
// @Tags orders
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} models.Order
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/orders/{id} [get]
func (h *OrderHandler) GetOrder(c *gin.Context) {
	requestID := getRequestID(c)
	ctx := c.Request.Context()
	orderID := c.Param("id")

	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	order, err := h.service.GetOrderByID(ctx, orderID)
	if err != nil {
		h.logger.Error("Failed to get order", zap.Error(err), zap.String("orderId", orderID), zap.String("requestId", requestID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error - Failed to get order"})
		return
	}

	c.JSON(http.StatusOK, order)
}

// ListOrders godoc
// @Summary List orders
// @Description Lists orders with optional filters and pagination
// @Tags orders
// @Produce json
// @Param status query string false "Filter by status"
// @Param customerId query string false "Filter by customer ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Results per page" default(10)
// @Success 200 {object} ListOrdersResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/orders [get]
func (h *OrderHandler) ListOrders(c *gin.Context) {
	requestID := getRequestID(c)
	ctx := c.Request.Context()

	status := c.Query("status")
	customerID := c.Query("customerId")

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(h.defaultPageSize)))
	if err != nil || limit < 1 {
		limit = h.defaultPageSize
	}
	if limit > h.maxPageSize {
		limit = h.maxPageSize
	}

	if status != "" {
		statusEnum := models.OrderStatus(status)
		if !statusEnum.IsValid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status value"})
			return
		}
	}

	orders, total, err := h.service.ListOrders(ctx, status, customerID, page, limit)
	if err != nil {
		h.logger.Error("Failed to list orders", zap.String("requestId", requestID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error - Failed to list orders"})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	response := ListOrdersResponse{
		Orders: orders,
		Pagination: PaginationResponse{
			Page:       page,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	c.JSON(http.StatusOK, response)
}

// UpdateOrderStatus godoc
// @Summary Update order status
// @Description Changes the status of an order and publishes an event
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param status body UpdateStatusRequest true "New status"
// @Success 200 {object} models.Order
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/orders/{id}/status [patch]
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	requestID := getRequestID(c)
	ctx := c.Request.Context()
	orderID := c.Param("id")

	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format or missing required fields"})
		return
	}

	newStatus := models.OrderStatus(req.Status)
	order, err := h.service.UpdateOrderStatus(ctx, orderID, newStatus)
	if err != nil {
		h.logger.Error("Failed to update order status", zap.String("orderId", orderID), zap.String("requestId", requestID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error - Failed to update order status"})
		return
	}

	c.JSON(http.StatusOK, order)
}

// Helper function to retrieve request ID from headers or context
func getRequestID(c *gin.Context) string {
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		if id, exists := c.Get("requestId"); exists {
			requestID = id.(string)
		}
	}
	return requestID
}
