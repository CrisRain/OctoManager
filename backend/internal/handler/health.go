package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"octomanger/backend/pkg/response"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Get(c *gin.Context) {
	response.Success(c, gin.H{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}
