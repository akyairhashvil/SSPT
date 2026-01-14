package tui

import (
	"fmt"
	"time"
)

// FormatDuration formats a duration for display (e.g., "2h 15m", "45s").
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// FormatTimeRemaining formats remaining time with appropriate precision.
func FormatTimeRemaining(remaining time.Duration) string {
	if remaining <= 0 {
		return "00:00"
	}
	mins := int(remaining.Minutes())
	secs := int(remaining.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", mins, secs)
}

// FormatSprintStatus returns a human-readable sprint status.
func FormatSprintStatus(status string, remaining time.Duration) string {
	switch status {
	case "active":
		return fmt.Sprintf("Active - %s remaining", FormatTimeRemaining(remaining))
	case "paused":
		return fmt.Sprintf("Paused - %s remaining", FormatTimeRemaining(remaining))
	case "completed":
		return "Completed"
	default:
		return "Ready"
	}
}

// FormatGoalCount formats goal counts for display.
func FormatGoalCount(completed, total int) string {
	if total == 0 {
		return "No goals"
	}
	return fmt.Sprintf("%d/%d goals", completed, total)
}

func formatDuration(d time.Duration) string {
	total := int(d.Seconds())
	if total < 0 {
		total = 0
	}
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
