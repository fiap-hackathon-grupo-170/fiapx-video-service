package http

import (
	"strings"

	"github.com/fiapx/fiapx-video-service/internal/domain/port"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware(validator port.TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "missing Authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid Authorization header format"})
			return
		}

		claims, err := validator.Validate(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.UserEmail)
		c.Next()
	}
}
