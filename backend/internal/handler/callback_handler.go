package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/service"
)

// CallbackHandler processes client authorization callbacks.
// This is the CORE integration point: external clients call this endpoint
// to authorize or reject a ticket after receiving a push notification.
type CallbackHandler struct {
	ticketService *service.TicketService
	logger        *zap.Logger
}

// NewCallbackHandler creates a new CallbackHandler.
func NewCallbackHandler(ts *service.TicketService, l *zap.Logger) *CallbackHandler {
	return &CallbackHandler{
		ticketService: ts,
		logger:        l,
	}
}

// AuthorizationCallback represents the callback request body from a client.
type AuthorizationCallback struct {
	TicketNo     string `json:"ticket_no"`
	Action       string `json:"action"`        // "authorize" | "reject"
	Operator     string `json:"operator"`
	Comment      string `json:"comment"`
	AuthorizedAt string `json:"authorized_at"`
}

// Handle processes an authorization callback from a client.
// It expects body_bytes and client_id to be set in the Gin context by
// upstream signature verification middleware.
func (h *CallbackHandler) Handle(c *gin.Context) {
	// 1. Get body_bytes from context (set by signature middleware).
	bodyBytesVal, exists := c.Get("body_bytes")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing body bytes"})
		return
	}
	bodyBytes, ok := bodyBytesVal.([]byte)
	if !ok || len(bodyBytes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body bytes"})
		return
	}

	// 2. Unmarshal to AuthorizationCallback.
	var req AuthorizationCallback
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	if req.TicketNo == "" || req.Action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket_no and action are required"})
		return
	}

	// 3. Find ticket by ticket_no.
	ticket, err := h.ticketService.GetByTicketNo(c.Request.Context(), req.TicketNo)
	if err != nil {
		h.logger.Warn("callback: ticket not found", zap.String("ticket_no", req.TicketNo), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}

	// 4. Verify client ownership.
	clientIDVal, _ := c.Get("client_id")
	clientID, ok := clientIDVal.(int64)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid client identity"})
		return
	}
	if ticket.ClientID == nil || *ticket.ClientID != clientID {
		c.JSON(http.StatusForbidden, gin.H{"error": "ticket not belong to this client"})
		return
	}

	// 5. Process action.
	operator := "client:" + req.Operator
	if operator == "client:" {
		operator = "client:unknown"
	}

	switch req.Action {
	case "authorize":
		if err := h.ticketService.TransitionStatus(c.Request.Context(), ticket.ID, model.TicketStatusInProgress, operator); err != nil {
			h.logger.Error("callback authorize failed", zap.Int64("ticket_id", ticket.ID), zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":      0,
			"message":   "ok",
			"ticket_no": req.TicketNo,
			"status":    "authorized",
		})

	case "reject":
		if err := h.ticketService.TransitionStatus(c.Request.Context(), ticket.ID, model.TicketStatusRejected, operator); err != nil {
			h.logger.Error("callback reject failed", zap.Int64("ticket_id", ticket.ID), zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":      0,
			"message":   "ok",
			"ticket_no": req.TicketNo,
			"status":    "rejected",
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action, must be 'authorize' or 'reject'"})
	}
}
