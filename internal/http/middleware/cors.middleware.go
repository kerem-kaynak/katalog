package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		environment := os.Getenv("ENVIRONMENT")

		if environment == "production" {
			allowedOrigins := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
			origin := c.Request.Header.Get("Origin")

			if contains(allowedOrigins, origin) {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			}
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
