package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"automarket/internal/domain"
	"automarket/internal/store"
)

type AdminHandler struct {
	vehicleStore *store.VehicleStore
}

func NewAdminHandler(vehicleStore *store.VehicleStore) *AdminHandler {
	return &AdminHandler{vehicleStore: vehicleStore}
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
	// Delete comprueba y borra bajo el mismo Lock: atomico ante borrados concurrentes
	deleted := h.vehicleStore.Delete(c.Param("id"))
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "publicación no encontrada"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "publicación eliminada"})
}
