package handlers

import (
	"errors"
	"hirely-api/internal/adapters/http/dto"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIKeyHandler struct {
	service *services.APIKeyService
}

func NewAPIKeyHandler(service *services.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{
		service: service,
	}
}

type CreateAPIKeyRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *APIKeyHandler) Create(c *gin.Context) {
	userID := c.GetString("userID")
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.HandleValidationError(c, err)
		return
	}

	apiKey, rawKey, err := h.service.CreateAPIKey(c.Request.Context(), userID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao criar chave"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"apiKey": apiKey,
		"key":    rawKey,
	})
}

func (h *APIKeyHandler) List(c *gin.Context) {
	userID := c.GetString("userID")

	keys, err := h.service.ListUserAPIKeys(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao listar chaves"})
		return
	}

	if keys == nil {
		keys = []*domain.APIKey{}
	}

	c.JSON(http.StatusOK, gin.H{"apiKeys": keys})
}

func (h *APIKeyHandler) Revoke(c *gin.Context) {
	userID := c.GetString("userID")
	keyID := c.Param("key_id")

	if err := h.service.RevokeAPIKey(c.Request.Context(), keyID, userID); err != nil {
		if errors.Is(err, domain.ErrAPIKeyNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chave não encontrada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao revogar chave"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chave revogada com sucesso"})
}
