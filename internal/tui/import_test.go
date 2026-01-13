package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestIsJSONSeed(t *testing.T) {
	if !isJSONSeed([]byte("  \n\t{ \"a\": 1 }")) {
		t.Fatalf("expected JSON seed detection")
	}
	if isJSONSeed([]byte(" # not json")) {
		t.Fatalf("expected non-JSON seed detection")
	}
}

func TestParseSeedTask(t *testing.T) {
	seed, err := parseSeedTask("Write docs #Docs #tag !3 @l ~weekly:mon,tue")
	if err != nil {
		t.Fatalf("parseSeedTask failed: %v", err)
	}
	if seed.Description != "Write docs" {
		t.Fatalf("expected description %q, got %q", "Write docs", seed.Description)
	}
	if len(seed.Tags) != 2 || seed.Tags[0] != "docs" || seed.Tags[1] != "tag" {
		t.Fatalf("unexpected tags: %#v", seed.Tags)
	}
	if seed.Priority != 3 {
		t.Fatalf("expected priority 3, got %d", seed.Priority)
	}
	if seed.Effort != "L" {
		t.Fatalf("expected effort L, got %q", seed.Effort)
	}
	if seed.Recurrence != "weekly:mon,tue" {
		t.Fatalf("expected recurrence weekly:mon,tue, got %q", seed.Recurrence)
	}

	empty, err := parseSeedTask("  ")
	if err != nil {
		t.Fatalf("parseSeedTask empty failed: %v", err)
	}
	if empty.Description != "" || len(empty.Tags) != 0 {
		t.Fatalf("expected empty seed, got %#v", empty)
	}
}

func TestParseSprintNumber(t *testing.T) {
	if num, ok := parseSprintNumber("+ 2"); !ok || num != 2 {
		t.Fatalf("expected sprint number 2, got %d (ok=%v)", num, ok)
	}
	if num, ok := parseSprintNumber("+ sprint 3"); !ok || num != 3 {
		t.Fatalf("expected sprint number 3, got %d (ok=%v)", num, ok)
	}
	if _, ok := parseSprintNumber("+ sprint"); ok {
		t.Fatalf("expected invalid sprint line to return ok=false")
	}
}

func TestSlugify(t *testing.T) {
	if got := slugify("My Workspace_Name"); got != "my-workspace-name" {
		t.Fatalf("expected slug %q, got %q", "my-workspace-name", got)
	}
}

func TestIsMaxSprintErr(t *testing.T) {
	if isMaxSprintErr(nil) {
		t.Fatalf("expected nil error to be false")
	}
	if !isMaxSprintErr(errors.New("Max Sprints reached")) {
		t.Fatalf("expected max sprints error to be true")
	}
}

func TestEnsureSeedFileCreates(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", temp)

	path, err := EnsureSeedFile()
	if err != nil {
		t.Fatalf("EnsureSeedFile failed: %v", err)
	}
	if filepath.Ext(path) != ".txt" && filepath.Ext(path) != ".json" {
		t.Fatalf("unexpected seed file extension: %s", filepath.Ext(path))
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("seed file not created: %v", err)
	}
}
