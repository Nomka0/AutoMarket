package handlers

import (
	"net/http"
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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title, description, price (> 0) and year (> 1900) are required"})
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
