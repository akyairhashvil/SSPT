package tui

import (
	"strings"
	"testing"
)

func TestRenderFooterStatusMessage(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.statusMessage = "Status ok"
	m.statusIsError = false
	footer := m.renderFooter()
	if !strings.Contains(footer, "Status ok") {
		t.Fatalf("expected footer to include status message")
	}
}

func TestRenderFooterClearDBPrompt(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.security.confirmingClearDB = true
	m.security.clearDBNeedsPass = true
	m.security.lock.PassphraseInput.SetValue("secret")
	footer := m.renderFooter()
	if !strings.Contains(footer, "Clear database") {
		t.Fatalf("expected footer to include clear db prompt")
	}
	if !strings.Contains(footer, "Enter passphrase") {
		t.Fatalf("expected footer to include passphrase prompt")
	}
}

func TestRenderFooterDefaultHelp(t *testing.T) {
	m := setupTestDashboard(t)
	m.width = 80
	m.statusMessage = ""
	m.Message = ""
	footer := m.renderFooter()
	if !strings.Contains(footer, "[n]New") {
		t.Fatalf("expected footer to include default help")
	}
}
