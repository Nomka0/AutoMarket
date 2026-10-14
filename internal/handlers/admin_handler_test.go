package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"automarket/internal/auth"
	"automarket/internal/domain"
	"automarket/internal/store"
)

func TestDeleteUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userStore := store.NewUserStore()
	tokenStore := auth.NewTokenStore()

	_ = userStore.Create(domain.User{ID: "admin-1", Username: "admin", Role: domain.RoleAdmin})
	_ = userStore.Create(domain.User{ID: "user-1", Username: "vendedor1", Role: domain.RoleSeller})

	adminToken := tokenStore.CreateToken("admin-1", domain.RoleAdmin)
	userToken := tokenStore.CreateToken("user-1", domain.RoleSeller)

	handler := NewAdminHandler(userStore, tokenStore)

	r := gin.New()
	adminGroup := r.Group("/api/admin")
	adminGroup.Use(auth.AuthMiddleware(tokenStore), auth.RequireRole(domain.RoleAdmin))
	adminGroup.DELETE("/users/:id", handler.DeleteUser)

	del := func(token, url string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, url, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		r.ServeHTTP(w, req)
		return w
	}

	// 1. Sin autenticación -> 401
	if w := del("", "/api/admin/users/user-1"); w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated request, got %d", w.Code)
	}

	// 2. Rol no administrador -> 403
	if w := del(userToken, "/api/admin/users/user-1"); w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin user, got %d", w.Code)
	}

	// 3. Usuario inexistente -> 404
	if w := del(adminToken, "/api/admin/users/non-existent"); w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent user, got %d", w.Code)
	}

	// 4. Eliminación exitosa por admin -> 200
	if w := del(adminToken, "/api/admin/users/user-1"); w.Code != http.StatusOK {
		t.Fatalf("expected 200 on successful delete, got %d: %s", w.Code, w.Body.String())
	}

	// Verificar que el usuario ya no existe en el store
	if _, ok := userStore.FindByID("user-1"); ok {
		t.Error("user-1 should have been deleted from userStore")
	}

	// Verificar que las sesiones del usuario fueron invalidadas
	if _, ok := tokenStore.ValidateToken(userToken); ok {
		t.Error("user-1 sessions should have been invalidated in tokenStore")
	}

	// 5. Autoeliminación de administrador -> 400 Bad Request
	wSelf := del(adminToken, "/api/admin/users/admin-1")
	if wSelf.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for admin self-deletion, got %d", wSelf.Code)
	}
	if _, ok := userStore.FindByID("admin-1"); !ok {
		t.Error("admin-1 should not have been deleted from userStore")
	}
}
