package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"automarket/internal/auth"
	"automarket/internal/domain"
	"automarket/internal/store"
)

type WebHandler struct {
	userStore    *store.UserStore
	vehicleStore *store.VehicleStore
	tokenStore   *auth.TokenStore
}

func NewWebHandler(userStore *store.UserStore, vehicleStore *store.VehicleStore, tokenStore *auth.TokenStore) *WebHandler {
	return &WebHandler{
		userStore:    userStore,
		vehicleStore: vehicleStore,
		tokenStore:   tokenStore,
	}
}

func (h *WebHandler) Index(c *gin.Context) {
	c.Redirect(http.StatusFound, "/catalog")
}

func (h *WebHandler) Catalog(c *gin.Context) {
	user, _ := auth.GetCurrentUser(c)
	vehicles := h.vehicleStore.FindApproved()
	c.HTML(http.StatusOK, "catalog.html", gin.H{
		"Title":    "Catálogo Público",
		"Vehicles": vehicles,
		"User":     user,
		"HasAdmin": h.userStore.HasAdmin(),
		"Error":    c.Query("error"),
		"Msg":      c.Query("msg"),
	})
}

func (h *WebHandler) CatalogDetail(c *gin.Context) {
	user, _ := auth.GetCurrentUser(c)
	id := c.Param("id")
	pub, ok := h.vehicleStore.FindByID(id)
	if !ok || pub.Status != domain.StatusApproved {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"Title": "Vehículo no disponible",
			"Error": "El vehículo solicitado no existe o no se encuentra disponible para venta pública.",
			"User":  user,
		})
		return
	}

	c.HTML(http.StatusOK, "catalog_detail.html", gin.H{
		"Title":   pub.Title,
		"Vehicle": pub,
		"User":    user,
	})
}

func (h *WebHandler) LoginPage(c *gin.Context) {
	user, ok := auth.GetCurrentUser(c)
	if ok {
		if user.Role == domain.RoleAdmin {
			c.Redirect(http.StatusSeeOther, "/admin")
			return
		}
		c.Redirect(http.StatusSeeOther, "/seller")
		return
	}

	c.HTML(http.StatusOK, "login.html", gin.H{
		"Title":    "Iniciar Sesión",
		"HasAdmin": h.userStore.HasAdmin(),
		"Error":    c.Query("error"),
		"Msg":      c.Query("msg"),
	})
}

func (h *WebHandler) LoginSubmit(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")

	user, ok := h.userStore.FindByUsername(username)
	if !ok || !auth.CheckPassword(password, user.Salt, user.PasswordHash) {
		c.Redirect(http.StatusSeeOther, "/login?error="+url.QueryEscape("Credenciales inválidas"))
		return
	}

	token := h.tokenStore.CreateToken(user.ID, user.Role)
	c.SetCookie("token", token, 86400, "/", "", false, true)

	if user.Role == domain.RoleAdmin {
		c.Redirect(http.StatusSeeOther, "/admin")
		return
	}
	c.Redirect(http.StatusSeeOther, "/seller")
}

func (h *WebHandler) Logout(c *gin.Context) {
	if token, err := c.Cookie("token"); err == nil && token != "" {
		h.tokenStore.DeleteToken(token)
	}
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/catalog?msg="+url.QueryEscape("Sesión cerrada exitosamente"))
}

func (h *WebHandler) RegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{
		"Title":    "Registro de Vendedor",
		"HasAdmin": h.userStore.HasAdmin(),
		"Error":    c.Query("error"),
	})
}

func (h *WebHandler) RegisterSubmit(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")
	contactInfo := strings.TrimSpace(c.PostForm("contact_info"))

	if username == "" || password == "" || contactInfo == "" {
		c.Redirect(http.StatusSeeOther, "/register?error="+url.QueryEscape("Todos los campos son obligatorios"))
		return
	}

	salt := auth.GenerateSalt()
	user := domain.User{
		ID:           auth.GenerateToken(),
		Username:     username,
		PasswordHash: auth.HashPassword(password, salt),
		Salt:         salt,
		Role:         domain.RoleSeller,
		ContactInfo:  contactInfo,
	}

	if err := h.userStore.Create(user); err != nil {
		c.Redirect(http.StatusSeeOther, "/register?error="+url.QueryEscape(err.Error()))
		return
	}

	token := h.tokenStore.CreateToken(user.ID, user.Role)
	c.SetCookie("token", token, 86400, "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/seller?msg="+url.QueryEscape("¡Registro exitoso! Bienvenido a tu panel de vendedor."))
}

func (h *WebHandler) SetupAdminPage(c *gin.Context) {
	if h.userStore.HasAdmin() {
		c.Redirect(http.StatusSeeOther, "/login?error="+url.QueryEscape("El administrador ya se encuentra registrado"))
		return
	}

	c.HTML(http.StatusOK, "setup_admin.html", gin.H{
		"Title": "Configuración Inicial de Administrador",
		"Error": c.Query("error"),
	})
}

func (h *WebHandler) SetupAdminSubmit(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")
	contactInfo := strings.TrimSpace(c.PostForm("contact_info"))
	if contactInfo == "" {
		contactInfo = "N/A"
	}

	if username == "" || password == "" {
		c.Redirect(http.StatusSeeOther, "/setup-admin?error="+url.QueryEscape("Usuario y contraseña son obligatorios"))
		return
	}

	admin, err := h.userStore.CreateAdmin(username, password, contactInfo)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/setup-admin?error="+url.QueryEscape(err.Error()))
		return
	}

	token := h.tokenStore.CreateToken(admin.ID, admin.Role)
	c.SetCookie("token", token, 86400, "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/admin?msg="+url.QueryEscape("Administrador inicial creado exitosamente."))
}

func (h *WebHandler) SellerDashboard(c *gin.Context) {
	session, _ := auth.GetCurrentUser(c)
	myVehicles := h.vehicleStore.FindBySellerID(session.UserID)

	var pendingCount, approvedCount, soldCount int
	for _, v := range myVehicles {
		switch v.Status {
		case domain.StatusPending:
			pendingCount++
		case domain.StatusApproved:
			approvedCount++
		case domain.StatusSold:
			soldCount++
		}
	}

	c.HTML(http.StatusOK, "seller.html", gin.H{
		"Title":         "Panel del Vendedor",
		"Vehicles":      myVehicles,
		"PendingCount":  pendingCount,
		"ApprovedCount": approvedCount,
		"SoldCount":      soldCount,
		"User":          session,
		"Error":         c.Query("error"),
		"Msg":           c.Query("msg"),
	})
}

func (h *WebHandler) CreateVehicleSubmit(c *gin.Context) {
	session, _ := auth.GetCurrentUser(c)
	title := strings.TrimSpace(c.PostForm("title"))
	description := strings.TrimSpace(c.PostForm("description"))
	priceStr := strings.TrimSpace(c.PostForm("price"))
	yearStr := strings.TrimSpace(c.PostForm("year"))

	if title == "" || description == "" || priceStr == "" || yearStr == "" {
		c.Redirect(http.StatusSeeOther, "/seller?error="+url.QueryEscape("Todos los campos del vehículo son obligatorios"))
		return
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil || price <= 0 {
		c.Redirect(http.StatusSeeOther, "/seller?error="+url.QueryEscape("Precio inválido"))
		return
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 1900 || year > time.Now().Year()+1 {
		c.Redirect(http.StatusSeeOther, "/seller?error="+url.QueryEscape("Año de vehículo inválido"))
		return
	}

	user, ok := h.userStore.FindByID(session.UserID)
	contactInfo := "No especificado"
	if ok && user.ContactInfo != "" {
		contactInfo = user.ContactInfo
	}

	pub := domain.VehiclePublication{
		ID:            auth.GenerateToken(),
		SellerID:      session.UserID,
		SellerContact: contactInfo,
		Title:         title,
		Description:   description,
		Price:         price,
		Year:          year,
		Status:        domain.StatusPending,
		CreatedAt:     time.Now().UTC(),
	}

	h.vehicleStore.Create(pub)
	c.Redirect(http.StatusSeeOther, "/seller?msg="+url.QueryEscape("Publicación registrada con éxito. Estado: PENDIENTE de moderación."))
}

func (h *WebHandler) MarkSoldSubmit(c *gin.Context) {
	session, _ := auth.GetCurrentUser(c)
	id := c.Param("id")

	pub, ok := h.vehicleStore.FindByID(id)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/seller?error="+url.QueryEscape("Publicación no encontrada"))
		return
	}

	if pub.SellerID != session.UserID {
		c.Redirect(http.StatusSeeOther, "/seller?error="+url.QueryEscape("No está autorizado para modificar esta publicación"))
		return
	}

	if pub.Status != domain.StatusApproved {
		c.Redirect(http.StatusSeeOther, "/seller?error="+url.QueryEscape(fmt.Sprintf("No se puede marcar como vendido un vehículo en estado %s", pub.Status)))
		return
	}

	h.vehicleStore.UpdateStatus(id, domain.StatusSold)
	c.Redirect(http.StatusSeeOther, "/seller?msg="+url.QueryEscape("Vehículo marcado como vendido y retirado del catálogo público."))
}

func (h *WebHandler) AdminDashboard(c *gin.Context) {
	session, _ := auth.GetCurrentUser(c)
	pending := h.vehicleStore.FindPending()
	allUsers := h.userStore.FindAll()

	c.HTML(http.StatusOK, "admin.html", gin.H{
		"Title":           "Panel de Moderación",
		"PendingVehicles": pending,
		"PendingCount":    len(pending),
		"Users":           allUsers,
		"UserCount":       len(allUsers),
		"User":            session,
		"Error":           c.Query("error"),
		"Msg":             c.Query("msg"),
	})
}

func (h *WebHandler) ApprovePublicationSubmit(c *gin.Context) {
	id := c.Param("id")
	pub, ok := h.vehicleStore.FindByID(id)
	if !ok || pub.Status != domain.StatusPending {
		c.Redirect(http.StatusSeeOther, "/admin?error="+url.QueryEscape("La publicación no existe o no está en estado pendiente"))
		return
	}

	h.vehicleStore.UpdateStatus(id, domain.StatusApproved)
	c.Redirect(http.StatusSeeOther, "/admin?msg="+url.QueryEscape("Publicación aprobada. Ahora es visible en el catálogo público."))
}

func (h *WebHandler) DeletePublicationSubmit(c *gin.Context) {
	id := c.Param("id")
	if !h.vehicleStore.Delete(id) {
		c.Redirect(http.StatusSeeOther, "/admin?error="+url.QueryEscape("Publicación no encontrada"))
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin?msg="+url.QueryEscape("Publicación eliminada correctamente."))
}

func (h *WebHandler) DeleteUserSubmit(c *gin.Context) {
	session, _ := auth.GetCurrentUser(c)
	id := c.Param("id")

	if session.UserID == id {
		c.Redirect(http.StatusSeeOther, "/admin?error="+url.QueryEscape("Un administrador no puede autoeliminarse"))
		return
	}

	if !h.userStore.Delete(id) {
		c.Redirect(http.StatusSeeOther, "/admin?error="+url.QueryEscape("Usuario no encontrado"))
		return
	}

	h.tokenStore.DeleteByUserID(id)
	c.Redirect(http.StatusSeeOther, "/admin?msg="+url.QueryEscape("Usuario y sus sesiones eliminados correctamente."))
}
