package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/repository"
	"github.com/xavierli/network-ticket/internal/service"
)

// AlertHandler handles alert webhook and AlertSource CRUD endpoints.
type AlertHandler struct {
	alertService   *service.AlertService
	alertSourceRepo *repository.AlertSourceRepo
	logger         *zap.Logger
}

// NewAlertHandler creates a new AlertHandler.
func NewAlertHandler(as *service.AlertService, ar *repository.AlertSourceRepo, l *zap.Logger) *AlertHandler {
	return &AlertHandler{
		alertService:   as,
		alertSourceRepo: ar,
		logger:         l,
	}
}

// Webhook receives an alert from an external source and ingests it.
func (h *AlertHandler) Webhook(c *gin.Context) {
	sourceIDStr := c.Param("source_id")
	sourceID, err := strconv.ParseInt(sourceIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source_id"})
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	result, err := h.alertService.Ingest(c.Request.Context(), sourceID, json.RawMessage(body))
	if err != nil {
		h.logger.Error("alert ingest failed", zap.Int64("source_id", sourceID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ingest failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ticket_id": result.TicketID,
		"ticket_no": result.TicketNo,
		"status":    result.Status,
		"is_new":    result.IsNew,
	})
}

// CreateAlertRequest represents a manual alert creation request.
type CreateAlertRequest struct {
	SourceType string          `json:"source_type" binding:"required"`
	RawData    json.RawMessage `json:"raw_data" binding:"required"`
	Severity   string          `json:"severity"`
	Title      string          `json:"title"`
}

// Create handles manual alert creation for testing.
func (h *AlertHandler) Create(c *gin.Context) {
	var req CreateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source_type and raw_data are required"})
		return
	}

	// For manual creation, use source_type directly and a dummy source ID of 0.
	result, err := h.alertService.Ingest(c.Request.Context(), 0, req.RawData)
	if err != nil {
		h.logger.Error("manual alert create failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create alert"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"ticket_id": result.TicketID,
		"ticket_no": result.TicketNo,
		"status":    result.Status,
		"is_new":    result.IsNew,
	})
}

// ListSources returns all alert sources.
func (h *AlertHandler) ListSources(c *gin.Context) {
	sources, err := h.alertSourceRepo.List(c.Request.Context())
	if err != nil {
		h.logger.Error("list alert sources failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list sources"})
		return
	}
	if sources == nil {
		sources = []model.AlertSource{}
	}
	c.JSON(http.StatusOK, gin.H{"items": sources})
}

// CreateSourceRequest represents the body for creating an alert source.
type CreateSourceRequest struct {
	Name          string          `json:"name" binding:"required"`
	Type          string          `json:"type" binding:"required"`
	Config        model.JSON      `json:"config"`
	ParserConfig  model.JSON      `json:"parser_config"`
	WebhookSecret *string         `json:"webhook_secret"`
	PollEndpoint  *string         `json:"poll_endpoint"`
	PollInterval  int             `json:"poll_interval"`
	DedupFields   model.JSON      `json:"dedup_fields"`
	DedupWindow   int             `json:"dedup_window_sec"`
	Status        string          `json:"status"`
}

// CreateSource creates a new alert source.
func (h *AlertHandler) CreateSource(c *gin.Context) {
	var req CreateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and type are required"})
		return
	}

	source := &model.AlertSource{
		Name:          req.Name,
		Type:          req.Type,
		Config:        req.Config,
		ParserConfig:  req.ParserConfig,
		WebhookSecret: req.WebhookSecret,
		PollEndpoint:  req.PollEndpoint,
		PollInterval:  req.PollInterval,
		DedupFields:   req.DedupFields,
		DedupWindow:   req.DedupWindow,
		Status:        req.Status,
	}
	if source.Status == "" {
		source.Status = "active"
	}

	id, err := h.alertSourceRepo.Create(c.Request.Context(), source)
	if err != nil {
		h.logger.Error("create alert source failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create source"})
		return
	}
	source.ID = id
	c.JSON(http.StatusCreated, source)
}

// UpdateSource updates an existing alert source.
func (h *AlertHandler) UpdateSource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req CreateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	source := &model.AlertSource{
		ID:            id,
		Name:          req.Name,
		Type:          req.Type,
		Config:        req.Config,
		ParserConfig:  req.ParserConfig,
		WebhookSecret: req.WebhookSecret,
		PollEndpoint:  req.PollEndpoint,
		PollInterval:  req.PollInterval,
		DedupFields:   req.DedupFields,
		DedupWindow:   req.DedupWindow,
		Status:        req.Status,
	}

	if err := h.alertSourceRepo.Update(c.Request.Context(), source); err != nil {
		h.logger.Error("update alert source failed", zap.Int64("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update source"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// DeleteSource removes an alert source.
func (h *AlertHandler) DeleteSource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.alertSourceRepo.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("delete alert source failed", zap.Int64("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete source"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
