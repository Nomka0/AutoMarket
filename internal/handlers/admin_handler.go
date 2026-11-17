package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"automarket/internal/auth"
	"automarket/internal/domain"
	"automarket/internal/store"
)

type AdminHandler struct {
	vehicleStore *store.VehicleStore
	userStore    *store.UserStore
	tokenStore   *auth.TokenStore
}

func NewAdminHandler(vehicleStore *store.VehicleStore, userStore *store.UserStore, tokenStore *auth.TokenStore) *AdminHandler {
	return &AdminHandler{
		vehicleStore: vehicleStore,
		userStore:    userStore,
		tokenStore:   tokenStore,
	}
}

func (h *AdminHandler) GetPendingPublications(c *gin.Context) {
	pubs := h.vehicleStore.FindPending()
	c.JSON(http.StatusOK, pubs)
}

func (h *AdminHandler) ApprovePublication(c *gin.Context) {
	id := c.Param("id")
	pub, ok := h.vehicleStore.FindByID(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "publicación no encontrada"})
		return
	}
	// solo PENDING -> APPROVED: aprobar una vendida la devolveria al catalogo
	if pub.Status != domain.StatusPending {
		c.JSON(http.StatusConflict, gin.H{"error": "solo se pueden aprobar publicaciones pendientes"})
		return
	}
	// puede fallar si otro admin la elimina entre FindByID y UpdateStatus
	if !h.vehicleStore.UpdateStatus(id, domain.StatusApproved) {
		c.JSON(http.StatusNotFound, gin.H{"error": "publicación no encontrada"})
		return
	}
	pub.Status = domain.StatusApproved
	c.JSON(http.StatusOK, pub)
}

func (h *AdminHandler) DeletePublication(c *gin.Context) {
	id := c.Param("id")
	pub, found := h.vehicleStore.FindByID(id)
	// Delete comprueba y borra bajo el mismo Lock: atomico ante borrados concurrentes
	deleted := h.vehicleStore.Delete(id)
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "publicación no encontrada"})
		return
	}
	if found && pub.ImageURL != "" {
		DeleteUploadedImage(pub.ImageURL, "uploads")
	}
	c.JSON(http.StatusOK, gin.H{"message": "publicación eliminada"})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	session, _ := auth.GetCurrentUser(c)
	if session.UserID == id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "un administrador no puede autoeliminarse"})
		return
	}

	_, exists := h.userStore.FindByID(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}

	h.userStore.Delete(id)
	if h.tokenStore != nil {
		h.tokenStore.DeleteByUserID(id)
	}

	c.JSON(http.StatusOK, gin.H{"message": "usuario eliminado correctamente"})
}

func (h *AdminHandler) SetupTOTP(c *gin.Context) {
	session, ok := auth.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no autenticado"})
		return
	}

	user, exists := h.userStore.FindByID(session.UserID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}
	if user.Role != domain.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "permiso denegado para su rol"})
		return
	}
	if user.TOTPSecret != "" {
		c.JSON(http.StatusConflict, gin.H{"error": "TOTP ya está configurado"})
		return
	}

	secret, otpauthURL, qrCode, err := generateTOTPSetup(user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate TOTP enrollment"})
		return
	}
	if err := h.userStore.EnrollTOTP(user.ID, secret); err != nil {
		switch {
		case errors.Is(err, store.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		case errors.Is(err, store.ErrUserNotAdmin):
			c.JSON(http.StatusForbidden, gin.H{"error": "permiso denegado para su rol"})
		case errors.Is(err, store.ErrTOTPAlreadyConfigured):
			c.JSON(http.StatusConflict, gin.H{"error": "TOTP ya está configurado"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save TOTP enrollment"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"secret":      secret,
		"otpauth_url": otpauthURL,
		"qr_code":     qrCode,
	})
}
