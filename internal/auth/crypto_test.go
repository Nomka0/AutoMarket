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

	if hash1 != hash2 {
		t.Fatal("expected deterministic hash for same password and salt")
	}

	if !CheckPassword(pass, salt1, hash1) {
		t.Fatal("expected CheckPassword to return true")
	}

	if !VerifyPassword(pass, salt1, hash1) {
		t.Fatal("expected VerifyPassword to return true")
	}

	if CheckPassword("WrongPassword", salt1, hash1) {
		t.Fatal("expected CheckPassword to return false on wrong password")
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

func TestTokenStore(t *testing.T) {
	store := NewTokenStore()

	userID := "user-123"
	token := store.CreateToken(userID)

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

	tokenA := store.CreateToken("user-456")
	tokenB := store.CreateToken("user-456")
	tokenC := store.CreateToken("user-789")

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
