package middleware

import (
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware returns a Gin middleware that allows cross-origin requests
// from the origins listed in the CORS_ALLOWED_ORIGINS environment variable,
// plus a hardcoded set of local dev origins.
//
// Example .env entry:
//
//	CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001
func CORSMiddleware() gin.HandlerFunc {

	// Default dev origins — always allowed
	defaultOrigins := []string{
		"http://localhost:3000",
		"http://localhost:3001",
		"https://aksa-capture.vercel.app",
	}

	// Merge with any additional origins set via environment variable
	if extra := os.Getenv("CORS_ALLOWED_ORIGINS"); extra != "" {
		for _, o := range strings.Split(extra, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				defaultOrigins = append(defaultOrigins, o)
			}
		}
	}

	return cors.New(cors.Config{
		AllowOrigins:     defaultOrigins,
		AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "ngrok-skip-browser-warning"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
