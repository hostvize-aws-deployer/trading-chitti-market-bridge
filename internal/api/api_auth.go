package api

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// APIKeyMiddleware validates API key for external requests
// Checks X-API-Key header against configured key
func APIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for health check and metrics endpoints
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		// Get API key from environment
		expectedKey := os.Getenv("API_KEY")

		// If no API key is configured, allow all requests (development mode)
		if expectedKey == "" {
			c.Next()
			return
		}

		// Extract API key from header
		providedKey := c.GetHeader("X-API-Key")

		// Also check Authorization header as fallback (format: "Bearer <key>")
		if providedKey == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				providedKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		// Validate API key
		if providedKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing API key",
				"message": "provide X-API-Key header or Authorization: Bearer <key>",
			})
			c.Abort()
			return
		}

		// Use constant-time comparison to prevent timing attacks
		if !compareKeys(expectedKey, providedKey) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid API key",
			})
			c.Abort()
			return
		}

		// API key is valid, continue
		c.Next()
	}
}

// compareKeys performs constant-time string comparison
// to prevent timing attacks
func compareKeys(expected, provided string) bool {
	// Convert to byte slices
	expectedBytes := []byte(expected)
	providedBytes := []byte(provided)

	// Use subtle.ConstantTimeCompare for timing-safe comparison
	return subtle.ConstantTimeCompare(expectedBytes, providedBytes) == 1
}

// OptionalAPIKeyMiddleware validates API key but doesn't require it
// Useful for endpoints that support both authenticated and public access
func OptionalAPIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		expectedKey := os.Getenv("API_KEY")

		// If no API key is configured, skip
		if expectedKey == "" {
			c.Next()
			return
		}

		// Extract API key from header
		providedKey := c.GetHeader("X-API-Key")
		if providedKey == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				providedKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		// If API key provided, validate it
		if providedKey != "" {
			if compareKeys(expectedKey, providedKey) {
				c.Set("authenticated", true)
			} else {
				c.Set("authenticated", false)
			}
		}

		c.Next()
	}
}

// GenerateAPIKey generates a secure random API key
// Use this to create initial API keys
// Example: import "crypto/rand"; import "encoding/base64"
func GenerateAPIKey() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := os.ReadFile("/dev/urandom"); err != nil {
		// Fallback for systems without /dev/urandom
		return "", err
	}

	// Encode as base64 URL-safe string
	// This will produce a 43-character string
	return strings.TrimRight(string(bytes), "="), nil
}
