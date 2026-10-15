package store

import (
	"os"
	"testing"

	"automarket/internal/auth"
	"automarket/internal/domain"
)

func TestUserStoreAndSeedAdmin(t *testing.T) {
	os.Setenv("ADMIN_USERNAME", "admin")
	os.Setenv("ADMIN_PASSWORD", "ciberinsegurid4d#")

	s := NewUserStore()

	if err := SeedAdmin(s); err != nil {
		t.Fatalf("unexpected seed error: %v", err)
	}

	// Idempotent seed
	if err := SeedAdmin(s); err != nil {
		t.Fatalf("unexpected seed re-run error: %v", err)
	}

	expectedUser := "admin"
	expectedPass := "ciberinsegurid4d#"

	admin, ok := s.FindByUsername(expectedUser)
	if !ok {
		t.Fatalf("expected admin user %s to exist", expectedUser)
	}

	if admin.Role != domain.RoleAdmin {
		t.Fatalf("expected role %s, got %s", domain.RoleAdmin, admin.Role)
	}

	if !auth.CheckPassword(expectedPass, admin.Salt, admin.PasswordHash) {
		t.Fatal("password verification failed")
	}

	// Duplicate username
	err := s.Create(domain.User{ID: "admin-2", Username: "admin"})
	if err == nil {
		t.Fatal("expected duplicate username error")
	}

	// FindByID and Delete
	if _, ok := s.FindByID(admin.ID); !ok {
		t.Fatal("expected find by id to succeed")
	}
	if !s.Delete(admin.ID) {
		t.Fatal("expected delete to succeed")
	}
	if _, ok := s.FindByID(admin.ID); ok {
		t.Fatal("expected user to be deleted")
	}
}

func TestUserStoreConcurrency(t *testing.T) {
	s := NewUserStore()
	done := make(chan bool)

	// Concurrency: 10 writers
	for i := 0; i < 10; i++ {
		go func(idx int) {
			_ = s.Create(domain.User{
				ID:       string(rune('a' + idx)),
				Username: string(rune('a' + idx)),
			})
			done <- true
		}(i)
	}

	// Concurrency: 10 readers
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = s.FindByUsername("admin")
			_, _ = s.FindByID("a")
			done <- true
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestCreateAdminAndHasAdmin(t *testing.T) {
	s := NewUserStore()
	if s.HasAdmin() {
		t.Fatal("expected no admin initially")
	}

	admin, err := s.CreateAdmin("admin", "pass1234", "admin@automarket.com")
	if err != nil {
		t.Fatalf("unexpected error creating admin: %v", err)
	}
	if admin.Role != domain.RoleAdmin {
		t.Fatalf("expected role %s, got %s", domain.RoleAdmin, admin.Role)
	}
	if !s.HasAdmin() {
		t.Fatal("expected HasAdmin to return true after creation")
	}

	// Attempt to create second admin
	_, err = s.CreateAdmin("admin2", "pass5678", "admin2@automarket.com")
	if err == nil {
		t.Fatal("expected error creating second admin")
	}
}

func TestSeedAdminOptional(t *testing.T) {
	s := NewUserStore()
	oldUser := os.Getenv("ADMIN_USERNAME")
	oldPass := os.Getenv("ADMIN_PASSWORD")
	defer func() {
		os.Setenv("ADMIN_USERNAME", oldUser)
		os.Setenv("ADMIN_PASSWORD", oldPass)
	}()

	os.Unsetenv("ADMIN_USERNAME")
	os.Unsetenv("ADMIN_PASSWORD")

	if err := SeedAdmin(s); err != nil {
		t.Fatalf("SeedAdmin should not error when env vars are missing: %v", err)
	}
	if s.HasAdmin() {
		t.Fatal("expected no admin when env vars are missing")
	}

	os.Setenv("ADMIN_USERNAME", "envadmin")
	os.Setenv("ADMIN_PASSWORD", "envpass123#")
	if err := SeedAdmin(s); err != nil {
		t.Fatalf("SeedAdmin should succeed with env vars: %v", err)
	}
	if !s.HasAdmin() {
		t.Fatal("expected admin to be seeded from env vars")
	}
}

func TestUserStore_JSONPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := tmpDir + "/users_test.json"

	s1 := NewUserStore(filePath)
	u1 := domain.User{
		ID:           "u-persist-1",
		Username:     "persistuser",
		PasswordHash: "dummyhash",
		Salt:         "dummysalt",
		Role:         domain.RoleSeller,
	}
	if err := s1.Create(u1); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if _, err := s1.CreateAdmin("persistadmin", "secret123", "admin@persist.com"); err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	// Simular reinicio creando una nueva instancia apuntando al mismo archivo
	s2 := NewUserStore(filePath)
	loadedUser, ok := s2.FindByID("u-persist-1")
	if !ok || loadedUser.Username != "persistuser" {
		t.Fatalf("expected persisted user to be loaded, got ok=%v, user=%+v", ok, loadedUser)
	}
	if loadedUser.PasswordHash != "dummyhash" || loadedUser.Salt != "dummysalt" {
		t.Fatalf("expected password hash and salt to be persisted, got hash=%s, salt=%s", loadedUser.PasswordHash, loadedUser.Salt)
	}

	loadedAdmin, ok := s2.FindByUsername("persistadmin")
	if !ok || !auth.CheckPassword("secret123", loadedAdmin.Salt, loadedAdmin.PasswordHash) {
		t.Fatal("expected admin password to be verifiable after reload from JSON")
	}
	if !s2.HasAdmin() {
		t.Fatal("expected persisted admin to be loaded")
	}

	// Verificar eliminación persistida
	if !s2.Delete("u-persist-1") {
		t.Fatal("expected delete to succeed")
	}

	s3 := NewUserStore(filePath)
	if _, ok := s3.FindByID("u-persist-1"); ok {
		t.Fatal("expected deleted user to remain deleted after reload")
	}
}


