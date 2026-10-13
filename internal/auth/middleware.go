package auth

import (
	"net/http"
	"slices"
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

func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, ok := GetCurrentUser(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no autenticado"})
			return
		}
		if !slices.Contains(allowedRoles, session.Role) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permiso denegado para su rol"})
			return
		}
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
