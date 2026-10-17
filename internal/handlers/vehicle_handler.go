package handlers

import (
	"net/http"
	"strconv"
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
	ImageURL    string  `json:"image_url"`
	Email       string  `json:"email"`
	Phone       string  `json:"phone"`
}

type UpdateContactRequest struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type VehicleHandler struct {
	vehicleStore *store.VehicleStore
	userStore    *store.UserStore
}

func NewVehicleHandler(vehicleStore *store.VehicleStore, userStore *store.UserStore) *VehicleHandler {
	return &VehicleHandler{vehicleStore: vehicleStore, userStore: userStore}
}

func (h *VehicleHandler) CreatePublication(c *gin.Context) {
	session, ok := auth.GetCurrentUser(c)
	user, found := h.userStore.FindByID(session.UserID)
	if !ok || !found {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no autenticado"})
		return
	}

	var title, description, imageURL, email, phone string
	var price float64
	var year int

	if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		title = strings.TrimSpace(c.PostForm("title"))
		description = strings.TrimSpace(c.PostForm("description"))
		email = strings.TrimSpace(c.PostForm("email"))
		phone = strings.TrimSpace(c.PostForm("phone"))
		p, err := strconv.ParseFloat(strings.TrimSpace(c.PostForm("price")), 64)
		if err != nil || p <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "precio inválido"})
			return
		}
		y, err := strconv.Atoi(strings.TrimSpace(c.PostForm("year")))
		if err != nil || y <= 1900 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "año inválido"})
			return
		}
		price = p
		year = y

		file, err := c.FormFile("photo")
		if err == nil && file != nil {
			savedURL, saveErr := SaveUploadedImage(file, "uploads")
			if saveErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": saveErr.Error()})
				return
			}
			imageURL = savedURL
		}
	} else {
		var req CreatePublicationRequest
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Description) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "datos de publicación inválidos o incompletos"})
			return
		}
		title = strings.TrimSpace(req.Title)
		description = strings.TrimSpace(req.Description)
		price = req.Price
		year = req.Year
		imageURL = strings.TrimSpace(req.ImageURL)
		email = strings.TrimSpace(req.Email)
		phone = strings.TrimSpace(req.Phone)
	}

	if title == "" || description == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de publicación inválidos o incompletos"})
		return
	}

	sellerEmail := user.GetEmail()
	sellerPhone := user.GetPhone()
	if email != "" {
		sellerEmail = email
	}
	if phone != "" {
		sellerPhone = phone
		if user.Phone == "" {
			user.Phone = phone
			user.ContactInfo = domain.FormatContact(sellerEmail, sellerPhone)
			h.userStore.Update(user)
		}
	}

	contactInfo := user.ContactInfo
	if sellerEmail != "" || sellerPhone != "" {
		formatted := domain.FormatContact(sellerEmail, sellerPhone)
		if formatted != "" {
			contactInfo = formatted
		}
	}

	pub := domain.VehiclePublication{
		ID:            auth.GenerateToken(), // ID aleatorio de 32 bytes en hex
		SellerID:      session.UserID,
		SellerContact: contactInfo,
		SellerEmail:   sellerEmail,
		SellerPhone:   sellerPhone,
		Title:         title,
		Description:   description,
		Price:         price,
		Year:          year,
		ImageURL:      imageURL,
		Status:        domain.StatusPending, // nunca entra aprobada directamente (HU-VEN-01)
		CreatedAt:     time.Now(),
	}
	h.vehicleStore.Create(pub)

	c.JSON(http.StatusCreated, pub)
}

func (h *VehicleHandler) UpdateContact(c *gin.Context) {
	session, ok := auth.GetCurrentUser(c)
	user, found := h.userStore.FindByID(session.UserID)
	if !ok || !found {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no autenticado"})
		return
	}

	var req UpdateContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos"})
		return
	}

	email := strings.TrimSpace(req.Email)
	phone := strings.TrimSpace(req.Phone)
	if email == "" && phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email o teléfono requerido"})
		return
	}

	contactInfo := domain.FormatContact(email, phone)
	user.Email = email
	user.Phone = phone
	user.ContactInfo = contactInfo

	h.userStore.Update(user)
	h.vehicleStore.UpdateSellerContact(user.ID, contactInfo, email, phone)

	c.JSON(http.StatusOK, gin.H{
		"message":      "contacto actualizado",
		"email":        email,
		"phone":        phone,
		"contact_info": contactInfo,
	})
}

func (h *VehicleHandler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("photo")
	if err != nil || file == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "archivo de foto requerido"})
		return
	}

	url, err := SaveUploadedImage(file, "uploads")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"image_url": url})
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
