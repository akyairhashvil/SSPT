package config

import "testing"

func TestConstants(t *testing.T) {
	if SprintDuration <= 0 {
		t.Fatalf("SprintDuration must be positive")
	}
	if BreakDuration <= 0 {
		t.Fatalf("BreakDuration must be positive")
	}
	if AutoLockAfter <= 0 {
		t.Fatalf("AutoLockAfter must be positive")
	}
	if AppName == "" {
		t.Fatalf("AppName should not be empty")
	}
	if DBFileName == "" {
		t.Fatalf("DBFileName should not be empty")
	}
	if MaxPassphraseAttempts <= 0 {
		t.Fatalf("MaxPassphraseAttempts must be positive")
	}
	if ViewModeAll != 0 || ViewModeFocused != 1 || ViewModeMinimal != 2 {
		t.Fatalf("unexpected view mode constants")
	}
}
