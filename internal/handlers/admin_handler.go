package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"automarket/internal/auth"
	"automarket/internal/store"
)

type AdminHandler struct {
	userStore  *store.UserStore
	tokenStore *auth.TokenStore
}

func NewAdminHandler(userStore *store.UserStore, tokenStore *auth.TokenStore) *AdminHandler {
	return &AdminHandler{userStore: userStore, tokenStore: tokenStore}
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	session, _ := auth.GetCurrentUser(c)
	if session.UserID == id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "un administrador no puede autoeliminarse"})
		return
	}

	if _, ok := h.userStore.FindByID(id); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}

	h.userStore.Delete(id)
	if h.tokenStore != nil {
		h.tokenStore.DeleteByUserID(id)
	}

	c.JSON(http.StatusOK, gin.H{"message": "usuario eliminado correctamente"})
}
