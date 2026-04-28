package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/service"
)

// ClientHandler handles client CRUD endpoints.
type ClientHandler struct {
	clientService *service.ClientService
	logger        *zap.Logger
}

// NewClientHandler creates a new ClientHandler.
func NewClientHandler(cs *service.ClientService, l *zap.Logger) *ClientHandler {
	return &ClientHandler{
		clientService: cs,
		logger:        l,
	}
}

// List returns all clients.
func (h *ClientHandler) List(c *gin.Context) {
	clients, err := h.clientService.List(c.Request.Context())
	if err != nil {
		h.logger.Error("list clients failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list clients"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": clients})
}

// CreateClientRequest represents the body for creating a client.
type CreateClientRequest struct {
	Name        string  `json:"name" binding:"required"`
	APIEndpoint string  `json:"api_endpoint" binding:"required"`
	APIKey      string  `json:"api_key" binding:"required"`
	HMACSecret  string  `json:"hmac_secret" binding:"required"`
	CallbackURL *string `json:"callback_url"`
	Config      string  `json:"config"`
	Status      string  `json:"status"`
}

// Create inserts a new client.
func (h *ClientHandler) Create(c *gin.Context) {
	var req CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, api_endpoint, api_key, and hmac_secret are required"})
		return
	}

	client := &model.Client{
		Name:        req.Name,
		APIEndpoint: req.APIEndpoint,
		APIKey:      req.APIKey,
		HMACSecret:  req.HMACSecret,
		CallbackURL: req.CallbackURL,
		Status:      req.Status,
	}
	if client.Status == "" {
		client.Status = "active"
	}
	if req.Config != "" {
		client.Config = model.JSON(req.Config)
	}

	id, err := h.clientService.Create(c.Request.Context(), client)
	if err != nil {
		h.logger.Error("create client failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create client"})
		return
	}
	client.ID = id
	c.JSON(http.StatusCreated, client)
}

// Update updates an existing client.
func (h *ClientHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	client := &model.Client{
		ID:          id,
		Name:        req.Name,
		APIEndpoint: req.APIEndpoint,
		APIKey:      req.APIKey,
		HMACSecret:  req.HMACSecret,
		CallbackURL: req.CallbackURL,
		Status:      req.Status,
	}
	if req.Config != "" {
		client.Config = model.JSON(req.Config)
	}

	if err := h.clientService.Update(c.Request.Context(), client); err != nil {
		h.logger.Error("update client failed", zap.Int64("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update client"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// Delete removes a client by ID.
func (h *ClientHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.clientService.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("delete client failed", zap.Int64("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete client"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
