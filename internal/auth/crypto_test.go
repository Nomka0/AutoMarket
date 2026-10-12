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

	if CheckPassword("WrongPassword", salt1, hash1) {
		t.Fatal("expected CheckPassword to return false on wrong password")
	}
}
