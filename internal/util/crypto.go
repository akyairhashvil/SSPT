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
	hasLetter := false
	hasDigit := false
	for _, r := range pass {
		if unicode.IsLetter(r) {
			hasLetter = true
		} else if unicode.IsDigit(r) {
			hasDigit = true
		}
		if hasLetter && hasDigit {
			return nil
		}
	}
	return fmt.Errorf("passphrase must include letters and numbers")
}
