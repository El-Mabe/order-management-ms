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

// OrderHandlerGin maneja los endpoints relacionados con órdenes usando Gin
type OrderHandlerGin struct {
	service         services.OrderService
	validator       *validator.Validate
	logger          *zap.Logger
	maxPageSize     int
	defaultPageSize int
}

// NewOrderHandlerGin crea una nueva instancia del handler con Gin
// func NewOrderHandler(service services.OrderService, logger *zap.Logger) *OrderHandlerGin {
func NewOrderHandler(service services.OrderService, logger *zap.Logger, defaultPageSize, maxPageSize int) *OrderHandlerGin {
	return &OrderHandlerGin{
		service:         service,
		validator:       validator.New(),
		logger:          logger,
		maxPageSize:     maxPageSize,
		defaultPageSize: defaultPageSize,
	}
}

// CreateOrderRequest representa la solicitud de creación de orden
type CreateOrderRequest struct {
	CustomerID string             `json:"customerId" binding:"required,uuid"`
	Items      []models.OrderItem `json:"items" binding:"required,min=1,max=100,dive"`
}

// UpdateStatusRequest representa la solicitud de actualización de estado
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=NEW IN_PROGRESS DELIVERED CANCELLED"`
}

// PaginationResponse representa la información de paginación
type PaginationResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

// ListOrdersResponse representa la respuesta de listado de órdenes
type ListOrdersResponse struct {
	Orders     []*models.Order    `json:"orders"`
	Pagination PaginationResponse `json:"pagination"`
}

// CreateOrder godoc
// @Summary Crear nueva orden
// @Description Crea una nueva orden de entrega
// @Tags orders
// @Accept json
// @Produce json
// @Param order body CreateOrderRequest true "Datos de la orden"
// @Success 201 {object} models.Order
// @Failure 400 {object} apierrors.ErrorResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /api/v1/orders [post]
func (h *OrderHandlerGin) CreateOrder(c *gin.Context) {
	requestID := getRequestID(c)
	ctx := c.Request.Context()

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body",
			zap.Error(err),
			zap.String("requestId", requestID),
		)

		// details := h.extractValidationErrors(err)
		// apiErr := apierrors.NewValidationError(requestID, details)
		// c.JSON(http.StatusBadRequest, gin.H{"error": apiErr})
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Crear orden
	order, err := h.service.CreateOrder(ctx, req.CustomerID, req.Items)
	if err != nil {
		h.logger.Error("Failed to create order",
			zap.String("requestId", requestID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, order)
}

// GetOrder godoc
// @Summary Obtener orden por ID
// @Description Obtiene una orden específica por su ID
// @Tags orders
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} models.Order
// @Failure 404 {object} apierrors.ErrorResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /api/v1/orders/{id} [get]
func (h *OrderHandlerGin) GetOrder(c *gin.Context) {
	requestID := getRequestID(c)
	ctx := c.Request.Context()
	orderID := c.Param("id")

	if orderID == "" {

		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	order, err := h.service.GetOrderByID(ctx, orderID)
	if err != nil {
		h.logger.Error("Failed to get order",
			// zap.Error(err),
			zap.String("orderId", orderID),
			zap.String("requestId", requestID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error - Failed to get order "})
		return
	}

	c.JSON(http.StatusOK, order)
}

// ListOrders godoc
// @Summary Listar órdenes
// @Description Lista órdenes con filtros opcionales y paginación
// @Tags orders
// @Produce json
// @Param status query string false "Filtrar por estado"
// @Param customerId query string false "Filtrar por ID de cliente"
// @Param page query int false "Número de página" default(1)
// @Param limit query int false "Resultados por página" default(10)
// @Success 200 {object} ListOrdersResponse
// @Failure 400 {object} apierrors.ErrorResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /api/v1/orders [get]
func (h *OrderHandlerGin) ListOrders(c *gin.Context) {
	requestID := getRequestID(c)
	ctx := c.Request.Context()

	// Obtener parámetros de query
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

	// Validar estado si se proporciona
	if status != "" {
		statusEnum := models.OrderStatus(status)
		if !statusEnum.IsValid() {

			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status value"})
			return
		}
	}

	// Listar órdenes
	orders, total, error := h.service.ListOrders(ctx, status, customerID, page, limit)
	if error != nil {
		h.logger.Error("Failed to list orders",
			// zap.Error(err),
			zap.String("requestId", requestID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error - Failed to list orders"})
		return
	}

	// Calcular total de páginas
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	response := ListOrdersResponse{
		Orders: orders,
		Pagination: PaginationResponse{
			Page: page,
			// Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	c.JSON(http.StatusOK, response)
}

// UpdateOrderStatus godoc
// @Summary Actualizar estado de orden
// @Description Cambia el estado de una orden y publica un evento
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param status body UpdateStatusRequest true "Nuevo estado"
// @Success 200 {object} models.Order
// @Failure 400 {object} apierrors.ErrorResponse
// @Failure 404 {object} apierrors.ErrorResponse
// @Failure 409 {object} apierrors.ErrorResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /api/v1/orders/{id}/status [patch]
func (h *OrderHandlerGin) UpdateOrderStatus(c *gin.Context) {
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

	// Actualizar estado
	newStatus := models.OrderStatus(req.Status)
	order, err := h.service.UpdateOrderStatus(ctx, orderID, newStatus)
	if err != nil {
		h.logger.Error("Failed to update order status",
			// zap.Error(err),
			zap.String("orderId", orderID),
			zap.String("requestId", requestID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error - Failed to update order status"})
		return
	}

	c.JSON(http.StatusOK, order)
}

// Funciones auxiliares

func getRequestID(c *gin.Context) string {
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		if id, exists := c.Get("requestId"); exists {
			requestID = id.(string)
		}
	}
	return requestID
}
