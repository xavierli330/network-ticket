package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/alert/parser"
	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/repository"
	"github.com/xavierli/network-ticket/internal/service"
)

// TicketHandler handles ticket CRUD and status transition endpoints.
type TicketHandler struct {
	ticketService  *service.TicketService
	clientRepo     *repository.ClientRepo
	ticketTypeRepo *repository.TicketTypeRepo
	ticketRepo     *repository.TicketRepo
	logger         *zap.Logger
}

// NewTicketHandler creates a new TicketHandler.
func NewTicketHandler(ts *service.TicketService, clientRepo *repository.ClientRepo, ticketTypeRepo *repository.TicketTypeRepo, ticketRepo *repository.TicketRepo, l *zap.Logger) *TicketHandler {
	return &TicketHandler{
		ticketService:  ts,
		clientRepo:     clientRepo,
		ticketTypeRepo: ticketTypeRepo,
		ticketRepo:     ticketRepo,
		logger:         l,
	}
}

// List returns a paginated list of tickets matching query filters.
func (h *TicketHandler) List(c *gin.Context) {
	var filter model.TicketFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters"})
		return
	}

	// Apply defaults.
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}

	tickets, total, err := h.ticketService.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("list tickets failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tickets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     tickets,
		"total":     total,
		"page":      filter.Page,
		"page_size": filter.PageSize,
	})
}

// Get returns a single ticket with its workflow states.
func (h *TicketHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ticket, err := h.ticketService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("get ticket failed", zap.Int64("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}

	states, err := h.ticketService.GetWorkflowStates(c.Request.Context(), id)
	if err != nil {
		h.logger.Warn("get workflow states failed", zap.Int64("ticket_id", id), zap.Error(err))
		states = []model.WorkflowState{}
	}

	c.JSON(http.StatusOK, gin.H{
		"ticket":         ticket,
		"workflow_states": states,
	})
}

// UpdateTicketRequest represents the body for manually updating a ticket's status.
type UpdateTicketRequest struct {
	Status   string `json:"status" binding:"required"`
	Operator string `json:"operator"`
}

// Update performs a manual status update on a ticket.
func (h *TicketHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req UpdateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}

	operator := req.Operator
	if operator == "" {
		operator = "manual"
	}

	if err := h.ticketService.TransitionStatus(c.Request.Context(), id, req.Status, operator); err != nil {
		h.logger.Error("update ticket status failed", zap.Int64("id", id), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

// Retry transitions a failed ticket back to pending for retry.
func (h *TicketHandler) Retry(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.ticketService.TransitionStatus(c.Request.Context(), id, model.TicketStatusPending, "retry"); err != nil {
		h.logger.Error("retry ticket failed", zap.Int64("id", id), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket queued for retry"})
}

// Cancel transitions a ticket to cancelled status.
func (h *TicketHandler) Cancel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.ticketService.TransitionStatus(c.Request.Context(), id, model.TicketStatusCancelled, "cancel"); err != nil {
		h.logger.Error("cancel ticket failed", zap.Int64("id", id), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket cancelled"})
}

// CreateManualRequest represents a manual ticket creation request.
type CreateManualRequest struct {
	TicketTypeID int64   `json:"ticket_type_id" binding:"required"`
	Title        string  `json:"title" binding:"required"`
	Description  *string `json:"description"`
	Severity     string  `json:"severity" binding:"required"`
	ClientID     *int64  `json:"client_id"`
}

// CreateManual handles manual ticket creation.
func (h *TicketHandler) CreateManual(c *gin.Context) {
	var req CreateManualRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket_type_id, title and severity are required"})
		return
	}

	ctx := c.Request.Context()

	// 1. Validate ticket type exists and is active.
	ticketType, err := h.ticketTypeRepo.GetByID(ctx, req.TicketTypeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket type"})
		return
	}
	if ticketType.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket type is not active"})
		return
	}

	// 2. Validate client if provided.
	if req.ClientID != nil {
		client, err := h.clientRepo.GetByID(ctx, *req.ClientID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client"})
			return
		}
		if client.Status != "active" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "client is not active"})
			return
		}
	}

	// 3. Construct ParsedAlert.
	var desc string
	if req.Description != nil {
		desc = *req.Description
	}
	parsed := &parser.ParsedAlert{
		Title:       req.Title,
		Description: desc,
		Severity:    req.Severity,
	}

	// 4. Create ticket via TicketService.
	ticket, err := h.ticketService.CreateTicket(ctx, 0, "manual", []byte("{}"), parsed, req.ClientID, nil)
	if err != nil {
		h.logger.Error("manual ticket creation failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ticket"})
		return
	}

	// 5. Set ticket_type_id.
	if err := h.ticketRepo.UpdateTicketTypeID(ctx, ticket.ID, &req.TicketTypeID); err != nil {
		h.logger.Error("set ticket type id failed", zap.Int64("ticket_id", ticket.ID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set ticket type"})
		return
	}

	// Refresh ticket to include the type.
	ticket, err = h.ticketService.GetByID(ctx, ticket.ID)
	if err != nil {
		h.logger.Warn("refresh ticket after create failed", zap.Int64("ticket_id", ticket.ID), zap.Error(err))
	}

	c.JSON(http.StatusCreated, ticket)
}
