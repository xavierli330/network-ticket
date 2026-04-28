package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xavierli/network-ticket/internal/nonce"
)

// NonceCheck returns a middleware that rejects replayed requests by checking
// the X-Nonce header against the provided nonce store. Each nonce value may
// only be used once within the given TTL.
func NonceCheck(store nonce.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		nonceVal := c.GetHeader("X-Nonce")
		if nonceVal == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing nonce"})
			return
		}
		ok, err := store.CheckAndSet(c.Request.Context(), nonceVal, 5*time.Minute)
		if err != nil || !ok {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "duplicate request"})
			return
		}
		c.Next()
	}
}
