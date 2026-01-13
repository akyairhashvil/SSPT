package tui

import (
	"testing"
	"time"

	"github.com/akyairhashvil/SSPT/internal/models"
)

func TestTaskElapsed(t *testing.T) {
	start := time.Now().Add(-2 * time.Second)
	goal := models.Goal{
		TaskElapsedSec: 5,
		TaskActive:     true,
		TaskStartedAt:  &start,
	}
	elapsed := taskElapsed(goal)
	if elapsed < 5*time.Second {
		t.Fatalf("expected elapsed to include stored seconds, got %s", elapsed)
	}
}

func TestFormatDuration(t *testing.T) {
	if got := formatDuration(65 * time.Second); got != "01:05" {
		t.Fatalf("expected 01:05, got %q", got)
	}
	if got := formatDuration(2*time.Hour + 3*time.Minute + 4*time.Second); got != "02:03:04" {
		t.Fatalf("expected 02:03:04, got %q", got)
	}
}

func TestTruncateLabel(t *testing.T) {
	if got := truncateLabel("short", 10); got != "short" {
		t.Fatalf("expected label to remain unchanged, got %q", got)
	}
	if got := truncateLabel("longer text", 4); got == "longer text" {
		t.Fatalf("expected label to be truncated, got %q", got)
	}
}

func TestTagHelpers(t *testing.T) {
	if !containsTag([]string{"a", "b"}, "b") {
		t.Fatalf("expected containsTag to find tag")
	}
	if containsTag([]string{"a"}, "c") {
		t.Fatalf("expected containsTag to return false")
	}
	if icon, ok := tagIcon("urgent"); !ok || icon == "" {
		t.Fatalf("expected urgent tag to return icon")
	}
	if _, ok := tagIcon("unknown"); ok {
		t.Fatalf("expected unknown tag to return false")
	}
}
