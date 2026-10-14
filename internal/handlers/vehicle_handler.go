package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"automarket/internal/auth"
	"automarket/internal/domain"
	"automarket/internal/store"
)

type CreatePublicationRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	Year        int     `json:"year" binding:"required,gt=1900"`
}

type VehicleHandler struct {
	vehicleStore *store.VehicleStore
	userStore    *store.UserStore
}

func NewVehicleHandler(vehicleStore *store.VehicleStore, userStore *store.UserStore) *VehicleHandler {
	return &VehicleHandler{vehicleStore: vehicleStore, userStore: userStore}
}

func (h *VehicleHandler) CreatePublication(c *gin.Context) {
	var req CreatePublicationRequest
	// "required" acepta "   "; TrimSpace rechaza los textos solo con espacios
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Description) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de publicación inválidos o incompletos"})
		return
	}

	// sin sesion o con sesion de un usuario ya eliminado: no hay vendedor al que asociar
	session, ok := auth.GetCurrentUser(c)
	user, found := h.userStore.FindByID(session.UserID)
	if !ok || !found {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no autenticado"})
		return
	}

	pub := domain.VehiclePublication{
		ID:            auth.GenerateToken(), // ID aleatorio de 32 bytes en hex
		SellerID:      session.UserID,
		SellerContact: user.ContactInfo,
		Title:         strings.TrimSpace(req.Title),
		Description:   strings.TrimSpace(req.Description),
		Price:         req.Price,
		Year:          req.Year,
		Status:        domain.StatusPending, // nunca entra aprobada directamente (HU-VEN-01)
		CreatedAt:     time.Now(),
	}
	h.vehicleStore.Create(pub)

	c.JSON(http.StatusCreated, pub)
}

func (h *VehicleHandler) MarkAsSold(c *gin.Context) {
	id := c.Param("id")
	pub, exists := h.vehicleStore.FindByID(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "publicación no encontrada"})
		return
	}

	session, ok := auth.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no autenticado"})
		return
	}
	if pub.SellerID != session.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "no tienes permiso para modificar esta publicación"})
		return
	}
	// solo APPROVED -> SOLD: una pendiente nunca estuvo en el catalogo y una vendida ya salio
	if pub.Status != domain.StatusApproved {
		c.JSON(http.StatusConflict, gin.H{"error": "solo se pueden marcar como vendidas publicaciones aprobadas"})
		return
	}
	// puede fallar si un admin la elimina entre FindByID y UpdateStatus
	if !h.vehicleStore.UpdateStatus(id, domain.StatusSold) {
		c.JSON(http.StatusNotFound, gin.H{"error": "publicación no encontrada"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "vehículo reportado como vendido exitosamente"})
}
