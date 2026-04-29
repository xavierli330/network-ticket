package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/service"
)

// TicketTypeHandler handles ticket type CRUD endpoints (admin-only).
type TicketTypeHandler struct {
	ticketTypeService *service.TicketTypeService
	logger            *zap.Logger
}

// NewTicketTypeHandler creates a new TicketTypeHandler.
func NewTicketTypeHandler(ts *service.TicketTypeService, l *zap.Logger) *TicketTypeHandler {
	return &TicketTypeHandler{
		ticketTypeService: ts,
		logger:            l,
	}
}

// List returns all ticket types.
func (h *TicketTypeHandler) List(c *gin.Context) {
	types, err := h.ticketTypeService.List(c.Request.Context())
	if err != nil {
		h.logger.Error("list ticket types failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list ticket types"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": types})
}

type CreateTicketTypeRequest struct {
	Code        string  `json:"code" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	Color       string  `json:"color"`
	Status      string  `json:"status"`
}

// Create creates a new ticket type.
func (h *TicketTypeHandler) Create(c *gin.Context) {
	var req CreateTicketTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code and name are required"})
		return
	}

	tt, err := h.ticketTypeService.Create(c.Request.Context(), req.Code, req.Name, req.Description, req.Color, req.Status)
	if err != nil {
		h.logger.Error("create ticket type failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, tt)
}

type UpdateTicketTypeRequest struct {
	Code        string  `json:"code" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	Color       string  `json:"color"`
	Status      string  `json:"status"`
}

// Update updates an existing ticket type.
func (h *TicketTypeHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req UpdateTicketTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.ticketTypeService.Update(c.Request.Context(), id, req.Code, req.Name, req.Description, req.Color, req.Status); err != nil {
		h.logger.Error("update ticket type failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// Delete removes a ticket type.
func (h *TicketTypeHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.ticketTypeService.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("delete ticket type failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
