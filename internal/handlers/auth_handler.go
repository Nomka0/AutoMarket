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

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthHandler struct {
	userStore  *store.UserStore
	tokenStore *auth.TokenStore
}

func NewAuthHandler(userStore *store.UserStore, tokenStore *auth.TokenStore) *AuthHandler {
	return &AuthHandler{userStore: userStore, tokenStore: tokenStore}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
		return
	}

	// mismo 401 para usuario inexistente y contraseña incorrecta: no revela si el usuario existe
	user, ok := h.userStore.FindByUsername(req.Username)
	if !ok || !auth.CheckPassword(req.Password, user.Salt, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token := h.tokenStore.CreateToken(user.ID, user.Role) // expira en 24h (sessionTTL)
	c.JSON(http.StatusOK, gin.H{"token": token, "role": user.Role, "username": user.Username})
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
