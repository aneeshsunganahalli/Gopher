package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log after request is processed
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if query != "" {
			path = path + "?" + query
		}

		// Skip health check logs in production to reduce noise
		if path != "/health" || statusCode != 200 {
			logger.Info("API request",
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", statusCode),
				zap.Duration("latency", latency),
				zap.String("ip", clientIP),
				zap.String("error", errorMessage),
			)
		}
	}
}

// RateLimitMiddleware limits request rates by IP address
func RateLimitMiddleware(limit int, duration time.Duration) gin.HandlerFunc {
	type client struct {
		count    int
		lastSeen time.Time
	}

	// Store clients with their request counts
	clients := make(map[string]*client)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		// Get or create client
		cl, exists := clients[ip]
		if !exists {
			clients[ip] = &client{count: 0, lastSeen: now}
			cl = clients[ip]
		}

		// Reset if outside window
		if now.Sub(cl.lastSeen) > duration {
			cl.count = 0
			cl.lastSeen = now
		}

		// Check limit
		if cl.count >= limit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		// Update count and continue
		cl.count++
		cl.lastSeen = now

		c.Next()
	}
}

// APIKeyMiddleware validates API keys
func APIKeyMiddleware(validKeys []string) gin.HandlerFunc {
	// Convert to map for O(1) lookup
	keysMap := make(map[string]bool)
	for _, key := range validKeys {
		keysMap[key] = true
	}

	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")

		// Skip auth for health check
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		if key == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "API key is required",
			})
			c.Abort()
			return
		}

		if !keysMap[key] {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Invalid API key",
			})
			c.Abort()
			return
		}

		// Valid key
		c.Next()
	}
}

// CORSMiddleware handles CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("Request handler panic",
					zap.Any("error", err),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
				)

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}
