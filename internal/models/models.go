// Package models defines the core domain types for SSPT including
// Workspace, Day, Sprint, Goal, and their status enumerations.
// These types are used by both the database and TUI packages.
package models

import "time"

// SprintStatus enumerates the possible states of a work block.
type SprintStatus string

const (
	StatusPending     SprintStatus = "pending"
	StatusActive      SprintStatus = "active"
	StatusPaused      SprintStatus = "paused"
	StatusCompleted   SprintStatus = "completed"
	StatusInterrupted SprintStatus = "interrupted"
)

func (s SprintStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusActive, StatusPaused, StatusCompleted, StatusInterrupted:
		return true
	default:
		return false
	}
}

// GoalStatus enumerates the possible states of a goal.
type GoalStatus string

const (
	GoalStatusPending    GoalStatus = "pending"
	GoalStatusInProgress GoalStatus = "in_progress"
	GoalStatusCompleted  GoalStatus = "completed"
	GoalStatusBlocked    GoalStatus = "blocked"
	GoalStatusArchived   GoalStatus = "archived"
)

func (s GoalStatus) IsValid() bool {
	switch s {
	case GoalStatusPending, GoalStatusInProgress, GoalStatusCompleted, GoalStatusBlocked, GoalStatusArchived:
		return true
	default:
		return false
	}
}

// Day represents a single calendar day.
type Day struct {
	ID        int64
	Date      string
	StartedAt time.Time
}

// Workspace represents an isolated project environment.
type Workspace struct {
	ID            int64
	Name          string
	Slug          string
	ViewMode      int // 0=All, 1=Focused, 2=Minimal
	Theme         string
	ShowBacklog   bool
	ShowCompleted bool
	ShowArchived  bool
}

// Sprint represents a 90-minute block.
type Sprint struct {
	ID             int64
	DayID          int64
	WorkspaceID    *int64
	SprintNumber   int
	Status         SprintStatus // pending, active, paused, completed, interrupted
	StartTime      *time.Time
	EndTime        *time.Time
	LastPausedAt   *time.Time
	ElapsedSeconds int
}

// Goal represents a single actionable item (Task).
type Goal struct {
	ID             int64
	ParentID       *int64 // For Subtasks
	WorkspaceID    *int64 // For Multi-tenancy
	SprintID       *int64 // Nil means backlog
	Description    string
	Notes          *string
	Status         GoalStatus // open, done, blocked, waiting, archived
	Priority       int        // 1=High, 3=Low
	Effort         *string    // S, M, L
	Tags           *string    // JSON array
	RecurrenceRule *string
	Links          *string // JSON array
	Rank           int
	CreatedAt      time.Time
	CompletedAt    *time.Time
	ArchivedAt     *time.Time
	TaskStartedAt  *time.Time
	TaskElapsedSec int
	TaskActive     bool
}

// JournalEntry represents a contextual note linked to a day and optionally a sprint.
type JournalEntry struct {
	ID          int64
	DayID       int64
	WorkspaceID *int64
	SprintID    *int64
	GoalID      *int64 // Context link to specific task
	Content     string
	Tags        *string // JSON array
	CreatedAt   time.Time
}
