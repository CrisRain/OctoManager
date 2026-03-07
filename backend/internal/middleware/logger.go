package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	applogger "octomanger/backend/pkg/logger"
)

const requestIDKey = "request_id"

func Logger(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set(requestIDKey, requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Request = c.Request.WithContext(applogger.WithTraceID(c.Request.Context(), requestID))

		start := time.Now()
		c.Next()

		logger.Info(
			"http request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.RequestURI()),
			zap.Int("status", c.Writer.Status()),
			zap.Int("bytes", c.Writer.Size()),
			zap.String("request_id", requestID),
			zap.String("trace_id", requestID),
			zap.String("remote_addr", c.Request.RemoteAddr),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Duration("duration", time.Since(start)),
		)
	}
}

func RequestID(c *gin.Context) string {
	if c == nil {
		return "-"
	}
	if value, exists := c.Get(requestIDKey); exists {
		if requestID, ok := value.(string); ok && strings.TrimSpace(requestID) != "" {
			return requestID
		}
	}
	if requestID := strings.TrimSpace(c.GetHeader("X-Request-ID")); requestID != "" {
		return requestID
	}
	return "-"
}
