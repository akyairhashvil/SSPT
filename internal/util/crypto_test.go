package util

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestValidatePassphrase(t *testing.T) {
	cases := []struct {
		name  string
		pass  string
		valid bool
	}{
		{"too short", "abc12", false},
		{"no digit", "Password", false},
		{"no upper", "password1", false},
		{"no lower", "PASSWORD1", false},
		{"valid", "Pass1234", true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePassphrase(tc.pass)
			if tc.valid && err != nil {
				t.Fatalf("expected valid, got error %v", err)
			}
			if !tc.valid && err == nil {
				t.Fatalf("expected error for %q", tc.pass)
			}
		})
	}
}

func TestHashPassphraseAndVerify(t *testing.T) {
	pass := "Abcdefg1"
	hash := HashPassphrase(pass)
	if hash == "" {
		t.Fatalf("expected hash to be generated")
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("expected argon2id hash format, got %q", hash)
	}
	if !VerifyPassphrase(hash, pass) {
		t.Fatalf("expected verification to succeed")
	}
	if VerifyPassphrase(hash, "Wrongpass1") {
		t.Fatalf("expected verification to fail for wrong passphrase")
	}
}

func TestVerifyPassphraseUpgradeLegacySHA256(t *testing.T) {
	pass := "Abcdefg1"
	sum := sha256.Sum256([]byte(pass))
	legacy := hex.EncodeToString(sum[:])

	ok, upgraded := VerifyPassphraseWithUpgrade(legacy, pass)
	if !ok {
		t.Fatalf("expected legacy hash to verify")
	}
	if upgraded == "" {
		t.Fatalf("expected upgraded hash to be returned")
	}
	if !strings.HasPrefix(upgraded, "$argon2id$") {
		t.Fatalf("expected argon2id hash after upgrade, got %q", upgraded)
	}
	if !VerifyPassphrase(upgraded, pass) {
		t.Fatalf("expected upgraded hash to verify")
	}
}

func BenchmarkHashPassphrase(b *testing.B) {
	pass := "Abcdefg1"
	for i := 0; i < b.N; i++ {
		if HashPassphrase(pass) == "" {
			b.Fatalf("expected hash to be generated")
		}
	}
}
