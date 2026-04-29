package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/repository"
)

// AdminHandler handles admin endpoints such as audit logs.
type AdminHandler struct {
	auditRepo *repository.AuditLogRepo
	logger    *zap.Logger
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(ar *repository.AuditLogRepo, l *zap.Logger) *AdminHandler {
	return &AdminHandler{
		auditRepo: ar,
		logger:    l,
	}
}

// ListAuditLogs returns a paginated list of audit log entries.
func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	operator := c.DefaultQuery("operator", "")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	logs, total, err := h.auditRepo.List(c.Request.Context(), page, pageSize, operator)
	if err != nil {
		h.logger.Error("list audit logs failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs"})
		return
	}
	if logs == nil {
		logs = []model.AuditLog{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
