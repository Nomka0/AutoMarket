package auth

import (
	_ "embed"
	"errors"
	"strings"
	"sync"
)

var (
	ErrPasswordTooShort         = errors.New("password must be at least 10 characters long")
	ErrPasswordContainsUsername = errors.New("password cannot contain username")
	ErrPasswordBlacklisted      = errors.New("password is in the list of common weak passwords")
)

//go:embed seclists_top500.txt
var seclistsTop500Raw string

var (
	weakPasswordsMap  map[string]struct{}
	weakPasswordsOnce sync.Once
)

func initWeakPasswords() {
	weakPasswordsOnce.Do(func() {
		lines := strings.Split(seclistsTop500Raw, "\n")
		weakPasswordsMap = make(map[string]struct{}, len(lines))
		for _, line := range lines {
			trimmed := strings.ToLower(strings.TrimSpace(line))
			if trimmed != "" {
				weakPasswordsMap[trimmed] = struct{}{}
			}
		}
	})
}

// ValidatePassword enforces password security guidelines aligned with NIST SP 800-63B and CIS Safeguard 5.2:
// 1. Minimum length of 10 characters.
// 2. Password cannot be identical to or contain the username (case-insensitive).
// 3. Password cannot be in the embedded blacklist of common weak passwords (Top 500 SecLists).
func ValidatePassword(password, username string) error {
	if len(password) < 10 {
		return ErrPasswordTooShort
	}

	trimmedUser := strings.ToLower(strings.TrimSpace(username))
	lowerPassword := strings.ToLower(strings.TrimSpace(password))

	if trimmedUser != "" {
		if len(trimmedUser) < 3 {
			if lowerPassword == trimmedUser {
				return ErrPasswordContainsUsername
			}
		} else if strings.Contains(lowerPassword, trimmedUser) {
			return ErrPasswordContainsUsername
		}
	}

	initWeakPasswords()
	if _, found := weakPasswordsMap[lowerPassword]; found {
		return ErrPasswordBlacklisted
	}

	return nil
}
