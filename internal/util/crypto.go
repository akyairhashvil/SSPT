package util

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"unicode"
)

func HashPassphrase(pass string) string {
	sum := sha256.Sum256([]byte(pass))
	return hex.EncodeToString(sum[:])
}

func ValidatePassphrase(pass string) error {
	if len(pass) < 8 {
		return fmt.Errorf("passphrase must be at least 8 characters")
	}
	var hasUpper, hasLower, hasDigit bool
	for _, r := range pass {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return fmt.Errorf("passphrase must contain uppercase, lowercase, and digit")
	}
	return nil
}
