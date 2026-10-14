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

func WebAuthMiddleware(store *TokenStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("token")
		if err != nil || token == "" {
			c.Redirect(http.StatusSeeOther, "/login")
			c.Abort()
			return
		}

		session, ok := store.ValidateToken(token)
		if !ok {
			c.SetCookie("token", "", -1, "/", "", false, true)
			c.Redirect(http.StatusSeeOther, "/login")
			c.Abort()
			return
		}
		c.Set(currentUserKey, session)
		c.Next()
	}
}

func WebOptionalAuthMiddleware(store *TokenStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token, err := c.Cookie("token"); err == nil && token != "" {
			if session, ok := store.ValidateToken(token); ok {
				c.Set(currentUserKey, session)
			}
		}
		c.Next()
	}
}

func WebRequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, ok := GetCurrentUser(c)
		if !ok {
			c.Redirect(http.StatusSeeOther, "/login")
			c.Abort()
			return
		}
		if !slices.Contains(allowedRoles, session.Role) {
			c.HTML(http.StatusForbidden, "error.html", gin.H{
				"Title": "Acceso Denegado",
				"Error": "Su rol no tiene autorización para acceder a este recurso.",
				"User":  session,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
