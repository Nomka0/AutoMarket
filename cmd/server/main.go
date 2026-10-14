package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"automarket/internal/auth"
	"automarket/internal/domain"
	"automarket/internal/handlers"
	"automarket/internal/store"
)

func loadEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			if os.Getenv(k) == "" {
				_ = os.Setenv(k, v)
			}
		}
	}
}

func main() {
	loadEnv(".env")

	userStore := store.NewUserStore()
	if err := store.SeedAdmin(userStore); err != nil {
		log.Printf("seed admin note: %v", err)
	}

	auditLogger, err := auth.NewFileAuditLogger("audit.log")
	if err != nil {
		log.Fatalf("failed to init audit logger: %v", err)
	}

	r := gin.Default()
	r.Use(auth.AuditLogMiddleware(auditLogger))

	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// un solo token store: el login emite los tokens que el middleware valida
	tokenStore := auth.NewTokenStore()

	authHandler := handlers.NewAuthHandler(userStore, tokenStore)
	authGroup := r.Group("/api/auth")
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/setup-admin", authHandler.SetupAdmin)
	authGroup.POST("/register-admin", authHandler.SetupAdmin)
	authGroup.GET("/has-admin", authHandler.HasAdmin)

	vehicleStore := store.NewVehicleStore()
	vehicleHandler := handlers.NewVehicleHandler(vehicleStore, userStore)

	// publico: sin middleware de autenticacion, el visitante no necesita token
	catalogHandler := handlers.NewCatalogHandler(vehicleStore)
	catalogGroup := r.Group("/api/catalog")
	catalogGroup.GET("", catalogHandler.GetCatalog)
	catalogGroup.GET("/:id", catalogHandler.GetCatalogDetail)

	sellerGroup := r.Group("/api/seller")
	sellerGroup.Use(auth.AuthMiddleware(tokenStore), auth.RequireRole(domain.RoleSeller))
	sellerGroup.POST("/vehicles", vehicleHandler.CreatePublication)
	sellerGroup.PATCH("/vehicles/:id/sold", vehicleHandler.MarkAsSold)

	adminHandler := handlers.NewAdminHandler(vehicleStore, userStore, tokenStore)
	adminGroup := r.Group("/api/admin")
	adminGroup.Use(auth.AuthMiddleware(tokenStore), auth.RequireRole(domain.RoleAdmin))
	adminGroup.GET("/publications/pending", adminHandler.GetPendingPublications)
	adminGroup.PATCH("/publications/:id/approve", adminHandler.ApprovePublication)
	adminGroup.DELETE("/publications/:id", adminHandler.DeletePublication)
	adminGroup.DELETE("/users/:id", adminHandler.DeleteUser)

	// Rutas Web con plantillas HTML
	webHandler := handlers.NewWebHandler(userStore, vehicleStore, tokenStore)

	// Vistas públicas
	r.GET("/", webHandler.Index)
	r.GET("/catalog", auth.WebOptionalAuthMiddleware(tokenStore), webHandler.Catalog)
	r.GET("/catalog/:id", auth.WebOptionalAuthMiddleware(tokenStore), webHandler.CatalogDetail)
	r.GET("/login", auth.WebOptionalAuthMiddleware(tokenStore), webHandler.LoginPage)
	r.POST("/login", webHandler.LoginSubmit)
	r.GET("/logout", webHandler.Logout)
	r.GET("/register", webHandler.RegisterPage)
	r.POST("/register", webHandler.RegisterSubmit)
	r.GET("/setup-admin", webHandler.SetupAdminPage)
	r.POST("/setup-admin", webHandler.SetupAdminSubmit)

	// Vistas protegidas Vendedor
	sellerWeb := r.Group("/seller").Use(auth.WebAuthMiddleware(tokenStore), auth.WebRequireRole(domain.RoleSeller))
	sellerWeb.GET("", webHandler.SellerDashboard)
	sellerWeb.POST("/vehicles", webHandler.CreateVehicleSubmit)
	sellerWeb.POST("/vehicles/:id/sold", webHandler.MarkSoldSubmit)

	// Vistas protegidas Administrador
	adminWeb := r.Group("/admin").Use(auth.WebAuthMiddleware(tokenStore), auth.WebRequireRole(domain.RoleAdmin))
	adminWeb.GET("", webHandler.AdminDashboard)
	adminWeb.POST("/publications/:id/approve", webHandler.ApprovePublicationSubmit)
	adminWeb.POST("/publications/:id/delete", webHandler.DeletePublicationSubmit)
	adminWeb.POST("/users/:id/delete", webHandler.DeleteUserSubmit)

	if err := r.Run(":8080"); err != nil {
		panic(err)
	}
}
