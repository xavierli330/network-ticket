package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authService *service.AuthService
	logger      *zap.Logger
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(as *service.AuthService, l *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authService: as,
		logger:      l,
	}
}

// LoginRequest represents the login request body.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login authenticates a user and returns a JWT token.
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
		return
	}

	token, claims, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		h.logger.Warn("login failed", zap.String("username", req.Username), zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       claims.UserID,
			"username": claims.Username,
			"role":     claims.Role,
		},
	})
}
