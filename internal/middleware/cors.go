package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORSMiddleware manually sets CORS headers on every response.
// This is simpler and more predictable than using gin-contrib/cors.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
		// Izinkan semua header yang dikirim oleh request (karena * kadang tidak mempan di beberapa browser/klien lama)
		reqHeaders := c.GetHeader("Access-Control-Request-Headers")
		if reqHeaders == "" {
			reqHeaders = "*"
		}
		c.Header("Access-Control-Allow-Headers", reqHeaders)
		
		c.Header("Access-Control-Expose-Headers", "*")
		c.Header("Access-Control-Max-Age", "86400") // 24 jam cache CORS config

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
