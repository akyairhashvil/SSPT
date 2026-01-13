package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupStaleDBArtifacts(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sprints.db")

	targets := []string{dbPath + ".enc", dbPath + ".bak"}
	for _, path := range targets {
		if err := os.WriteFile(path, []byte("stale"), 0o600); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}

	cleanupStaleDBArtifacts(dbPath)

	for _, path := range targets {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed", path)
		}
	}
}
