package models

import (
	"database/sql"
	"time"
)

// SprintStatus enumerates the possible states of a work block.
type SprintStatus string

const (
	StatusPending     SprintStatus = "pending"
	StatusActive      SprintStatus = "active"
	StatusCompleted   SprintStatus = "completed"
	StatusInterrupted SprintStatus = "interrupted"
)

// Day represents a single calendar day.
type Day struct {
	ID        int64
	Date      string
	StartedAt time.Time
}

// Workspace represents an isolated project environment.
type Workspace struct {
	ID   int64
	Name string
	Slug string
}

// Sprint represents a 90-minute block.
type Sprint struct {
	ID           int64
	DayID        int64
	WorkspaceID  sql.NullInt64
	SprintNumber int
	Status       string // pending, active, paused, completed, interrupted
	StartTime    sql.NullTime
	EndTime      sql.NullTime
	LastPausedAt sql.NullTime
	ElapsedSeconds int
	Goals        []Goal // The tasks allocated to this sprint
}

// Goal represents a single actionable item (Task).
type Goal struct {
	ID          int64
	ParentID    sql.NullInt64 // For Subtasks
	WorkspaceID sql.NullInt64 // For Multi-tenancy
	SprintID    sql.NullInt64 // If Valid=false, it belongs to the Backlog
	Description string
	Notes       string
	Status      string // open, done, blocked, waiting, archived
	Priority    int    // 1=High, 3=Low
	Effort      string // S, M, L
	Tags        string // JSON array
	Links       string // JSON array
	Rank        int
	CreatedAt   time.Time
	CompletedAt sql.NullTime

	// UI Helper fields (not in DB)
	Subtasks []Goal
	Expanded bool
	Level    int // Indentation level
}

// JournalEntry represents a contextual note linked to a day and optionally a sprint.
type JournalEntry struct {
	ID          int64
	DayID       int64
	WorkspaceID sql.NullInt64
	SprintID    sql.NullInt64
	GoalID      sql.NullInt64 // Context link to specific task
	Content     string
	Tags        string // JSON array
	CreatedAt   time.Time
}
