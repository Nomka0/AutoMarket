package auth

import (
	"testing"
)

func TestCrypto(t *testing.T) {
	salt1 := GenerateSalt()
	salt2 := GenerateSalt()

	if len(salt1) != 32 { // 16 bytes hex = 32 chars
		t.Fatalf("expected salt length 32, got %d", len(salt1))
	}
	if salt1 == salt2 {
		t.Fatal("expected distinct salts")
	}

	pass := "SecretPassword123!"
	hash1 := HashPassword(pass, salt1)
	hash2 := HashPassword(pass, salt1)

	if len(hash1) != 64 { // 32 bytes hex = 64 chars
		t.Fatalf("expected argon2 hash length 64, got %d", len(hash1))
	}

	if hash1 != hash2 {
		t.Fatal("expected deterministic hash for same password and salt")
	}

	// Contraseña correcta -> true
	if !CheckPassword(pass, salt1, hash1) {
		t.Fatal("expected CheckPassword to return true for correct password")
	}

	if !VerifyPassword(pass, salt1, hash1) {
		t.Fatal("expected VerifyPassword to return true for correct password")
	}

	// Contraseña incorrecta -> false
	if CheckPassword("WrongPassword", salt1, hash1) {
		t.Fatal("expected CheckPassword to return false on wrong password")
	}

	// Salt alterado -> false
	if CheckPassword(pass, salt2, hash1) {
		t.Fatal("expected CheckPassword to return false when salt is altered")
	}

	token1 := GenerateToken()
	token2 := GenerateToken()

	if len(token1) != 64 { // 32 bytes hex = 64 chars
		t.Fatalf("expected token length 64, got %d", len(token1))
	}
	if token1 == token2 {
		t.Fatal("expected distinct tokens")
	}
}

func TestCrypto_EdgeCasesAndSpecialCharacters(t *testing.T) {
	testCases := []struct {
		name     string
		password string
	}{
		{"EmptyPassword", ""},
		{"SpecialChars", "P@ssw0rd!#$&*()_+~`|}{[]:;?><,./-="},
		{"UnicodeAndEmojis", "ContraseñaSegura🔒🔑#2026!ñáéíóú"},
		{"LongPassphrase", "correct horse battery staple with many words and spaces included 2026"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			salt := GenerateSalt()
			hash := HashPassword(tc.password, salt)
			if len(hash) != 64 {
				t.Fatalf("expected 64 chars hex hash, got %d", len(hash))
			}

			// Contraseña correcta -> true
			if !CheckPassword(tc.password, salt, hash) {
				t.Fatalf("expected CheckPassword to return true for %s", tc.name)
			}

			// Contraseña incorrecta -> false
			if CheckPassword(tc.password+"_wrong", salt, hash) {
				t.Fatalf("expected CheckPassword to return false for wrong password in %s", tc.name)
			}

			// Salt alterado -> false
			otherSalt := GenerateSalt()
			if CheckPassword(tc.password, otherSalt, hash) {
				t.Fatalf("expected CheckPassword to return false for altered salt in %s", tc.name)
			}
		})
	}
}

func TestTokenStore(t *testing.T) {
	store := NewTokenStore()

	userID := "user-123"
	token := store.CreateToken(userID, "vendedor")

	if len(token) != 64 {
		t.Fatalf("expected token length 64, got %d", len(token))
	}

	foundID, ok := store.GetUserID(token)
	if !ok || foundID != userID {
		t.Fatalf("expected userID %s, got %s (ok=%v)", userID, foundID, ok)
	}

	if _, ok := store.GetUserID("invalid-token"); ok {
		t.Fatal("expected invalid token to return false")
	}

	store.DeleteToken(token)
	if _, ok := store.GetUserID(token); ok {
		t.Fatal("expected token to be deleted")
	}

	tokenA := store.CreateToken("user-456", "vendedor")
	tokenB := store.CreateToken("user-456", "vendedor")
	tokenC := store.CreateToken("user-789", "vendedor")

	store.DeleteByUserID("user-456")

	if _, ok := store.GetUserID(tokenA); ok {
		t.Fatal("expected tokenA to be invalidated")
	}
	if _, ok := store.GetUserID(tokenB); ok {
		t.Fatal("expected tokenB to be invalidated")
	}
	if _, ok := store.GetUserID(tokenC); !ok {
		t.Fatal("expected tokenC for user-789 to remain active")
	}
}

func TestTokenStoreConcurrency(t *testing.T) {
	store := NewTokenStore()
	done := make(chan bool)

	// 10 concurrent creators
	for i := 0; i < 10; i++ {
		go func(idx int) {
			token := store.CreateToken("user", "vendedor")
			_ = token
			done <- true
		}(i)
	}

	// 10 concurrent readers/getters
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = store.GetUserID("random-token")
			done <- true
		}()
	}

	// 10 concurrent deleters
	for i := 0; i < 10; i++ {
		go func() {
			store.DeleteByUserID("user")
			done <- true
		}()
	}

	for i := 0; i < 30; i++ {
		<-done
	}
}
