package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

func (h *HealthHandler) Check(c *gin.Context) {
	response := HealthResponse{
		Status:    "ok",
		Service:   "hirely-api",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

func (h *HealthHandler) Ping(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}
