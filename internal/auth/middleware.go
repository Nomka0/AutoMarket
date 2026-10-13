package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(store *TokenStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "formato de Authorization inválido, se espera: Bearer <token>"})
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")

		session, ok := store.ValidateToken(token)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token inválido o expirado"})
			return
		}
		c.Set("userID", session.UserID)
		c.Next()
	}
}
