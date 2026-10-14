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
		log.Fatalf("failed to seed admin: %v", err)
	}

	r := gin.Default()

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

	// un solo vehicle store: lo que publica el vendedor es lo que modera el admin
	vehicleStore := store.NewVehicleStore()
	vehicleHandler := handlers.NewVehicleHandler(vehicleStore, userStore)
	adminHandler := handlers.NewAdminHandler(vehicleStore)
	sellerGroup := r.Group("/api/seller")
	sellerGroup.Use(auth.AuthMiddleware(tokenStore), auth.RequireRole(domain.RoleSeller))
	sellerGroup.POST("/publications", vehicleHandler.CreatePublication)

	adminGroup := r.Group("/api/admin")
	adminGroup.Use(auth.AuthMiddleware(tokenStore), auth.RequireRole(domain.RoleAdmin))
	adminGroup.GET("/publications/pending", adminHandler.GetPendingPublications)
	adminGroup.PATCH("/publications/:id/approve", adminHandler.ApprovePublication)
	adminGroup.DELETE("/publications/:id", vehicleHandler.DeletePublication)

	if err := r.Run(":8080"); err != nil {
		panic(err)
	}
}
