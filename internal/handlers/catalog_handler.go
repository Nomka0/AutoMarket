package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

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
