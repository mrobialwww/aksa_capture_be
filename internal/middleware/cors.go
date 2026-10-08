package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORSMiddleware manually sets CORS headers on every response.
// This is simpler and more predictable than using gin-contrib/cors.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, ngrok-skip-browser-warning")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Max-Age", "43200")

		// Respond immediately to preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
