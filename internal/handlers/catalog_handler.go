package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"automarket/internal/domain"
	"automarket/internal/store"
)

type CatalogHandler struct {
	vehicleStore *store.VehicleStore
}

func NewCatalogHandler(vehicleStore *store.VehicleStore) *CatalogHandler {
	return &CatalogHandler{vehicleStore: vehicleStore}
}

func (h *CatalogHandler) GetCatalog(c *gin.Context) {
	// FindApproved filtra en el store: PENDING y SOLD nunca llegan al catalogo publico
	pubs := h.vehicleStore.FindApproved()
	c.JSON(http.StatusOK, pubs)
}

func (h *CatalogHandler) GetCatalogDetail(c *gin.Context) {
	id := c.Param("id")
	pub, exists := h.vehicleStore.FindByID(id)
	// mismo 404 para inexistente, PENDING y SOLD: el visitante no distingue si el ID existe
	if !exists || pub.Status != domain.StatusApproved {
		c.JSON(http.StatusNotFound, gin.H{"error": "vehículo no disponible en catálogo"})
		return
	}
	c.JSON(http.StatusOK, pub)
}
