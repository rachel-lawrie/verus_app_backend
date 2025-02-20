package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rachel-lawrie/verus_backend_core/common"
	"github.com/rachel-lawrie/verus_backend_core/utils"
)

// APIKeyAuthMiddleware authenticates requests using an API key
func APIKeyAuthMiddleware(collection common.CollectionInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key is missing"})
			c.Abort() // Prevent further processing
			return
		}

		// Hash the provided API key
		hashedKey := utils.HashAPIKey(apiKey)

		// Query the database for the hashed key
		var secret struct {
			ClientID string `bson:"client_id"`
		}
		err := collection.FindOne(context.Background(), map[string]interface{}{
			"client_secret_hash": hashedKey,
			"revoked":            false, // Ensure key is active
			"deleted_at":         nil,   // Ensure key is not deleted
		}).Decode(&secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or inactive API key"})
			c.Abort() // Prevent further processing
			return
		}

		// Pass the validated client ID to the next handler
		c.Set("client_id", secret.ClientID)
		c.Next() // Continue to the next middleware or handler
	}
}
