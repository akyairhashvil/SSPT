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

// Goal IDs.
const (
	GoalIDNone int64 = 0
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
	PriorityLow    = 1
	PriorityMedium = 2
	PriorityHigh   = 3
)

// Database/application settings.
const (
	AppName               = "sspt"
	DBFileName            = "sprints.db"
	MaxPassphraseAttempts = 5
)

// Database timeouts.
const (
	DefaultDBTimeout     = 5 * time.Second
	LongOperationTimeout = 30 * time.Second
)

// Display settings.
const (
	MinDisplayColumns      = 3
	MaxDisplayColumns      = 4
	AnalyticsChartPadding  = 24
	AnalyticsChartMaxWidth = 48
	AnalyticsChartMinWidth = 10
	MinTerminalWidth       = 80
	MinTerminalHeight      = 24
)
