package store

import (
	"os"
	"testing"
	"time"

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

func TestAccountLockoutPolicy(t *testing.T) {
	s := NewUserStore()
	username := "test_lockout_user"

	// 1. Inicialmente no está bloqueado
	locked, _ := s.IsLocked(username)
	if locked {
		t.Fatal("cuenta no debe estar bloqueada inicialmente")
	}

	// 2. Acumular 4 intentos fallidos: no debe bloquearse aún
	for i := 1; i <= 4; i++ {
		count, isLocked := s.RecordFailedAttempt(username)
		if count != i {
			t.Fatalf("esperaba count %d, obtuvo %d", i, count)
		}
		if isLocked {
			t.Fatalf("intento %d no debió bloquear la cuenta", i)
		}
	}

	locked, _ = s.IsLocked(username)
	if locked {
		t.Fatal("la cuenta no debe bloquearse con menos de 5 intentos fallidos")
	}

	// 3. Quinto intento fallido: debe bloquearse
	count, isLocked := s.RecordFailedAttempt(username)
	if count != 5 || !isLocked {
		t.Fatalf("el 5to intento debió bloquear la cuenta: count=%d, isLocked=%v", count, isLocked)
	}

	locked, remaining := s.IsLocked(username)
	if !locked || remaining <= 0 {
		t.Fatalf("IsLocked debió retornar true con tiempo restante positivo, got locked=%v, remaining=%v", locked, remaining)
	}

	// 4. Reset tras login exitoso
	s.ResetFailedAttempts(username)
	locked, _ = s.IsLocked(username)
	if locked {
		t.Fatal("la cuenta debe desbloquearse tras ResetFailedAttempts")
	}
	if s.GetFailedAttempts(username) != 0 {
		t.Fatal("contador de fallos debe ser 0 tras reset")
	}

	// 5. Verificación de desbloqueo automático por expiración
	s.SetLockoutPolicy(5, 50*time.Millisecond)
	for i := 0; i < 5; i++ {
		s.RecordFailedAttempt(username)
	}
	if locked, _ := s.IsLocked(username); !locked {
		t.Fatal("cuenta debió bloquearse")
	}

	time.Sleep(60 * time.Millisecond)
	if locked, _ := s.IsLocked(username); locked {
		t.Fatal("la cuenta debió desbloquearse automáticamente tras expirar el período de bloqueo")
	}
}


