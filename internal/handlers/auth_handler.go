package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"automarket/internal/auth"
	"automarket/internal/domain"
	"automarket/internal/store"
)

type RegisterRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	ContactInfo string `json:"contact_info" binding:"required"`
}

type AuthHandler struct {
	userStore *store.UserStore
}

func NewAuthHandler(userStore *store.UserStore) *AuthHandler {
	return &AuthHandler{userStore: userStore}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username, password and contact_info are required"})
		return
	}

	salt := auth.GenerateSalt()
	user := domain.User{
		ID:           auth.GenerateToken(), // ID aleatorio de 32 bytes en hex
		Username:     req.Username,
		PasswordHash: auth.HashPassword(req.Password, salt),
		Salt:         salt,
		Role:         domain.RoleSeller, // el registro publico nunca crea admins
		ContactInfo:  req.ContactInfo,
	}

	if err := h.userStore.Create(user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": user.ID, "username": user.Username})
}
