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
		Title:         req.Title,
		Description:   req.Description,
		Price:         req.Price,
		Year:          req.Year,
		Status:        domain.StatusPending, // nunca entra aprobada directamente (HU-VEN-01)
		CreatedAt:     time.Now(),
	}
	h.vehicleStore.Create(pub)

	c.JSON(http.StatusCreated, pub)
}

func (h *VehicleHandler) ListPending(c *gin.Context) {
	c.JSON(http.StatusOK, h.vehicleStore.FindPending())
}

func (h *VehicleHandler) ApprovePublication(c *gin.Context) {
	pub, ok := h.vehicleStore.FindByID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "publicación no encontrada"})
		return
	}
	// solo PENDING -> APPROVED: aprobar una vendida la devolveria al catalogo
	if pub.Status != domain.StatusPending {
		c.JSON(http.StatusConflict, gin.H{"error": "solo se pueden aprobar publicaciones pendientes"})
		return
	}
	if !h.vehicleStore.UpdateStatus(pub.ID, domain.StatusApproved) {
		c.JSON(http.StatusNotFound, gin.H{"error": "publicación no encontrada"})
		return
	}
	pub.Status = domain.StatusApproved
	c.JSON(http.StatusOK, pub)
}

func (h *VehicleHandler) DeletePublication(c *gin.Context) {
	if !h.vehicleStore.Delete(c.Param("id")) {
		c.JSON(http.StatusNotFound, gin.H{"error": "publicación no encontrada"})
		return
	}
	c.Status(http.StatusNoContent)
}
