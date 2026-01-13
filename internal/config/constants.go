// Package config defines application-wide constants and configuration
// values including timer durations, display settings, and paths.
package config

import "time"

// Timer durations.
const (
	SprintDuration = 90 * time.Minute
	BreakDuration  = 30 * time.Minute
	AutoLockAfter  = 5 * time.Minute
)

// View modes.
const (
	ViewModeAll = iota
	ViewModeFocused
	ViewModeMinimal
)

// Task statuses.
const (
	StatusPending   = "pending"
	StatusActive    = "active"
	StatusCompleted = "completed"
	StatusArchived  = "archived"
)

// Priority levels.
const (
	PriorityLow      = "low"
	PriorityMedium   = "medium"
	PriorityHigh     = "high"
	PriorityCritical = "critical"
)

// Database/application settings.
const (
	AppName               = "sspt"
	DBFileName            = "sprints.db"
	MaxPassphraseAttempts = 5
)

// Display settings.
const (
	MinDisplayColumns      = 3
	MaxDisplayColumns      = 4
	AnalyticsChartPadding  = 24
	AnalyticsChartMaxWidth = 48
	AnalyticsChartMinWidth = 10
)
