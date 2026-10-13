package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const currentUserKey = "currentUser"

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
		c.Set(currentUserKey, session)
		c.Next()
	}
}

func GetCurrentUser(c *gin.Context) (Session, bool) {
	val, exists := c.Get(currentUserKey)
	if !exists {
		return Session{}, false
	}
	s, ok := val.(Session)
	return s, ok
}
