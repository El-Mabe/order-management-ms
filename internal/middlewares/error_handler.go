package middlewares

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func ErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		err := c.Errors.ByType(gin.ErrorTypeAny).Last()
		if err == nil {
			return
		}

		code := http.StatusInternalServerError
		if c.Writer.Status() != http.StatusOK {
			code = c.Writer.Status()
		}

		requestID, exists := c.Get("requestId")
		if !exists {
			requestID = "unknown"
		}

		logger.Error("Request error",
			zap.Error(err.Err),
			zap.String("requestId", requestID.(string)),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", code),
		)

		c.JSON(code, gin.H{
			"error": gin.H{
				"code":      "INTERNAL_ERROR",
				"message":   "Internal server error",
				"requestId": requestID,
				"timestamp": time.Now(),
			},
		})
	}
}
