package auth_test

import (
	"errors"
	"testing"

	"automarket/internal/auth"
)

func TestValidatePassword_Valid(t *testing.T) {
	validPasswords := []struct {
		password string
		username string
	}{
		{"C0mpl3x#P@ssw0rd!", "user123"},
		{"Correct-Horse-Battery-Staple", "alice"},
		{"AutoMarket_Secure_2026", "bob_seller"},
		{"Str0ngP@ss10", "charlie"},
		{"ValidLongSecret123$", "admin_corp"},
	}

	for _, tc := range validPasswords {
		if err := auth.ValidatePassword(tc.password, tc.username); err != nil {
			t.Errorf("expected password '%s' for user '%s' to be valid, got: %v", tc.password, tc.username, err)
		}
	}
}

func TestValidatePassword_TooShort(t *testing.T) {
	shortPasswords := []string{
		"",
		"12345",
		"short",
		"12345678",
		"Abc12345!", // 9 characters
	}

	for _, pwd := range shortPasswords {
		err := auth.ValidatePassword(pwd, "validuser")
		if !errors.Is(err, auth.ErrPasswordTooShort) {
			t.Errorf("expected ErrPasswordTooShort for '%s', got: %v", pwd, err)
		}
	}
}

func TestValidatePassword_ContainsUsername(t *testing.T) {
	testCases := []struct {
		password string
		username string
	}{
		{"Admin123456!", "admin"},
		{"superADMIN2026", "admin"},
		{"myalicepassword", "alice"},
		{"userBob_SecretPass", "bob"},
		{"charlie99999", "Charlie"},
	}

	for _, tc := range testCases {
		err := auth.ValidatePassword(tc.password, tc.username)
		if !errors.Is(err, auth.ErrPasswordContainsUsername) {
			t.Errorf("expected ErrPasswordContainsUsername for pwd '%s' and user '%s', got: %v", tc.password, tc.username, err)
		}
	}
}

func TestValidatePassword_Blacklisted(t *testing.T) {
	blacklistedPasswords := []string{
		"1234567890",
		"qwertyuiop",
		"password123",
		"password1234",
		"administrator",
		"iloveyou123",
		"trustnoone1",
		"changeme123",
		"welcome1234",
		"QWERTYUIOP", // Case-insensitive check
		"PASSWORD123",
	}

	for _, pwd := range blacklistedPasswords {
		err := auth.ValidatePassword(pwd, "unrelated_user")
		if !errors.Is(err, auth.ErrPasswordBlacklisted) {
			t.Errorf("expected ErrPasswordBlacklisted for '%s', got: %v", pwd, err)
		}
	}
}
