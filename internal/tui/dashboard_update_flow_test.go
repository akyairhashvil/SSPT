package tui

import (
	"errors"
	"testing"
	"time"

	"github.com/akyairhashvil/SSPT/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDashboardUpdateClearsErrorOnKey(t *testing.T) {
	m := setupTestDashboard(t)
	m.err = errors.New("boom")

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	updated := model.(DashboardModel)
	if updated.err != nil {
		t.Fatalf("expected err to be cleared")
	}
}

func TestDashboardUpdateClearsMessageOnKey(t *testing.T) {
	m := setupTestDashboard(t)
	m.Message = "hello"

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	updated := model.(DashboardModel)
	if updated.Message != "" {
		t.Fatalf("expected Message to be cleared")
	}
}

func TestDashboardUpdateLockedRequiresPassphrase(t *testing.T) {
	m := setupTestDashboard(t)
	m.security.lock.Locked = true
	m.security.lock.PassphraseHash = ""
	m.security.lock.PassphraseInput.SetValue("")

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(DashboardModel)
	if updated.security.lock.Message == "" {
		t.Fatalf("expected lock message to be set")
	}
}

func TestDashboardUpdateWindowSize(t *testing.T) {
	m := setupTestDashboard(t)

	model, _ := m.Update(tea.WindowSizeMsg{Width: 50, Height: 20})
	updated := model.(DashboardModel)
	if updated.width != 50 || updated.height != 20 {
		t.Fatalf("expected width/height to be updated")
	}
	if updated.progress.Width < config.MinTitleWidth {
		t.Fatalf("expected progress width >= %d, got %d", config.MinTitleWidth, updated.progress.Width)
	}
}

func TestPassphraseRateLimited(t *testing.T) {
	m := setupTestDashboard(t)
	m.security.lock.LockUntil = time.Now().Add(2 * time.Second)

	limited, wait := m.passphraseRateLimited()
	if !limited {
		t.Fatalf("expected rate limit to be active")
	}
	if wait <= 0 {
		t.Fatalf("expected positive wait duration")
	}
}
