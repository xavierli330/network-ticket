package middleware

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/xavierli/network-ticket/internal/pkg"
	"github.com/xavierli/network-ticket/internal/repository"
)

// HMACSignature returns a middleware that verifies HMAC request signatures
// sent by external clients. It expects X-Api-Key, X-Timestamp, and
// X-Signature headers, looks up the client by API key, verifies the
// timestamp drift, and validates the HMAC of (timestamp || body).
//
// On success it sets "body_bytes", "client", and "client_id" in the context.
func HMACSignature(clientRepo *repository.ClientRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-Api-Key")
		timestampStr := c.GetHeader("X-Timestamp")
		signature := c.GetHeader("X-Signature")

		if apiKey == "" || timestampStr == "" || signature == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing auth headers"})
			return
		}

		client, err := clientRepo.GetByAPIKey(c.Request.Context(), apiKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}

		timestamp, _ := strconv.ParseInt(timestampStr, 10, 64)
		if err := pkg.VerifyTimestamp(timestamp, 300); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		body, _ := io.ReadAll(c.Request.Body)
		if !pkg.VerifyHMAC(client.HMACSecret, timestamp, body, signature) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}

		// Store body bytes and client in context for handler to use.
		c.Set("body_bytes", body)
		c.Set("client", client)
		c.Set("client_id", client.ID)
		c.Next()
	}
}
