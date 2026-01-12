package util

import "testing"

func TestValidatePassphrase(t *testing.T) {
	cases := []struct {
		name  string
		pass  string
		valid bool
	}{
		{"too short", "abc12", false},
		{"no digit", "password", false},
		{"no letter", "12345678", false},
		{"valid", "pass1234", true},
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
