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

// Sprint represents a 90-minute block.
type Sprint struct {
	ID           int64
	DayID        int64
	SprintNumber int
	Status       string // pending, active, paused, completed, interrupted
	StartTime    sql.NullTime
	EndTime      sql.NullTime
	LastPausedAt sql.NullTime
	ElapsedSeconds int
	Goals        []Goal // The tasks allocated to this sprint
}

// Goal represents a single actionable item.
type Goal struct {
	ID          int64
	SprintID    sql.NullInt64 // If Valid=false, it belongs to the Backlog
	Description string
	Status      string // pending, completed
	Rank        int
	CreatedAt   time.Time
	CompletedAt sql.NullTime
}

// JournalEntry represents a contextual note linked to a day and optionally a sprint.
type JournalEntry struct {
	ID        int64
	DayID     int64
	SprintID  sql.NullInt64
	Content   string
	CreatedAt time.Time
}
