package config

import "time"

// Timer durations.
const (
	SprintDuration = 90 * time.Minute
	BreakDuration  = 30 * time.Minute
	AutoLockAfter  = 10 * time.Minute
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
	DBFileName            = "sspt.db"
	MaxPassphraseAttempts = 5
)
