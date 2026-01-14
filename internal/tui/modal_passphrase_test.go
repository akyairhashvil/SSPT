package tui

import "testing"

func TestPassphraseModal_Validation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		wantStage int
	}{
		{"empty passphrase", "", true, 1},
		{"short passphrase", "abc", true, 1},
		{"valid passphrase", "Secure123", false, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := setupTestDashboard(t)
			m.security.changingPassphrase = true
			m.security.passphraseStage = 1
			m.inputs.passphraseNew.SetValue(tt.input)

			updated, _, handled := m.handleModalConfirmPassphrase()
			if !handled {
				t.Fatalf("expected modal handler to run")
			}
			if tt.wantErr {
				if updated.security.passphraseStatus == "" {
					t.Fatalf("expected validation error status")
				}
			} else if updated.security.passphraseStatus != "" {
				t.Fatalf("expected no validation error, got %q", updated.security.passphraseStatus)
			}
			if updated.security.passphraseStage != tt.wantStage {
				t.Fatalf("expected passphrase stage %d, got %d", tt.wantStage, updated.security.passphraseStage)
			}
		})
	}
}
