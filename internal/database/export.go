package database

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/akyairhashvil/SSPT/internal/util"
)

type ExportDay struct {
	ID        int64  `json:"id"`
	Date      string `json:"date"`
	StartedAt string `json:"started_at"`
}

type ExportWorkspace struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	ViewMode      int    `json:"view_mode"`
	Theme         string `json:"theme"`
	ShowBacklog   bool   `json:"show_backlog"`
	ShowCompleted bool   `json:"show_completed"`
	ShowArchived  bool   `json:"show_archived"`
}

type ExportSprint struct {
	ID             int64   `json:"id"`
	DayID          int64   `json:"day_id"`
	WorkspaceID    *int64  `json:"workspace_id,omitempty"`
	SprintNumber   int     `json:"sprint_number"`
	Status         string  `json:"status"`
	StartTime      *string `json:"start_time,omitempty"`
	EndTime        *string `json:"end_time,omitempty"`
	LastPausedAt   *string `json:"last_paused_at,omitempty"`
	ElapsedSeconds int     `json:"elapsed_seconds"`
}

type ExportGoal struct {
	ID             int64    `json:"id"`
	ParentID       *int64   `json:"parent_id,omitempty"`
	WorkspaceID    *int64   `json:"workspace_id,omitempty"`
	SprintID       *int64   `json:"sprint_id,omitempty"`
	Description    string   `json:"description"`
	Notes          *string  `json:"notes,omitempty"`
	Status         string   `json:"status"`
	Priority       int      `json:"priority"`
	Effort         *string  `json:"effort,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	RecurrenceRule *string  `json:"recurrence_rule,omitempty"`
	Links          []string `json:"links,omitempty"`
	Rank           int      `json:"rank"`
	CreatedAt      string   `json:"created_at"`
	CompletedAt    *string  `json:"completed_at,omitempty"`
	ArchivedAt     *string  `json:"archived_at,omitempty"`
	TaskStartedAt  *string  `json:"task_started_at,omitempty"`
	TaskElapsedSec int      `json:"task_elapsed_seconds,omitempty"`
	TaskActive     bool     `json:"task_active,omitempty"`
}

type ExportJournalEntry struct {
	ID          int64    `json:"id"`
	DayID       int64    `json:"day_id"`
	WorkspaceID *int64   `json:"workspace_id,omitempty"`
	SprintID    *int64   `json:"sprint_id,omitempty"`
	GoalID      *int64   `json:"goal_id,omitempty"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags,omitempty"`
	CreatedAt   string   `json:"created_at"`
}

type ExportTaskDep struct {
	GoalID      int64 `json:"goal_id"`
	DependsOnID int64 `json:"depends_on_id"`
}

func GetAllDays() ([]ExportDay, error) {
	d, err := getDefaultDB()
	if err != nil {
		return nil, err
	}
	return d.GetAllDays()
}

func (d *Database) GetAllDays() ([]ExportDay, error) {
	rows, err := d.DB.Query("SELECT id, date, started_at FROM days ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExportDay
	for rows.Next() {
		var d ExportDay
		var startedAt string
		if err := rows.Scan(&d.ID, &d.Date, &startedAt); err != nil {
			return nil, err
		}
		d.StartedAt = startedAt
		out = append(out, d)
	}
	return out, nil
}

func GetAllSprintsFlat() ([]ExportSprint, error) {
	d, err := getDefaultDB()
	if err != nil {
		return nil, err
	}
	return d.GetAllSprintsFlat()
}

func (d *Database) GetAllSprintsFlat() ([]ExportSprint, error) {
	rows, err := d.DB.Query(`
		SELECT id, day_id, workspace_id, sprint_number, status, start_time, end_time, last_paused_at, elapsed_seconds
		FROM sprints ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExportSprint
	for rows.Next() {
		var s ExportSprint
		var wsID sql.NullInt64
		var start, end, last sql.NullTime
		if err := rows.Scan(&s.ID, &s.DayID, &wsID, &s.SprintNumber, &s.Status, &start, &end, &last, &s.ElapsedSeconds); err != nil {
			return nil, err
		}
		if wsID.Valid {
			id := wsID.Int64
			s.WorkspaceID = &id
		}
		if start.Valid {
			val := start.Time.Format(time.RFC3339)
			s.StartTime = &val
		}
		if end.Valid {
			val := end.Time.Format(time.RFC3339)
			s.EndTime = &val
		}
		if last.Valid {
			val := last.Time.Format(time.RFC3339)
			s.LastPausedAt = &val
		}
		out = append(out, s)
	}
	return out, nil
}

func GetAllGoalsExport() ([]ExportGoal, error) {
	d, err := getDefaultDB()
	if err != nil {
		return nil, err
	}
	return d.GetAllGoalsExport()
}

func (d *Database) GetAllGoalsExport() ([]ExportGoal, error) {
	rows, err := d.DB.Query(`
		SELECT id, parent_id, workspace_id, sprint_id, description, notes, status, priority, effort, tags, recurrence_rule, links, rank, created_at, completed_at, archived_at, task_started_at, task_elapsed_seconds, task_active
		FROM goals ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExportGoal
	for rows.Next() {
		var g ExportGoal
		var parentID, workspaceID, sprintID sql.NullInt64
		var notes, effort, recurrence, tags, links sql.NullString
		var completedAt, archivedAt, taskStarted sql.NullTime
		var taskActive int
		if err := rows.Scan(&g.ID, &parentID, &workspaceID, &sprintID, &g.Description, &notes, &g.Status, &g.Priority, &effort, &tags, &recurrence, &links, &g.Rank, &g.CreatedAt, &completedAt, &archivedAt, &taskStarted, &g.TaskElapsedSec, &taskActive); err != nil {
			return nil, err
		}
		if parentID.Valid {
			id := parentID.Int64
			g.ParentID = &id
		}
		if workspaceID.Valid {
			id := workspaceID.Int64
			g.WorkspaceID = &id
		}
		if sprintID.Valid {
			id := sprintID.Int64
			g.SprintID = &id
		}
		if notes.Valid {
			val := notes.String
			g.Notes = &val
		}
		if effort.Valid {
			val := effort.String
			g.Effort = &val
		}
		if recurrence.Valid {
			val := recurrence.String
			g.RecurrenceRule = &val
		}
		if tags.Valid && tags.String != "" && tags.String != "[]" {
			g.Tags = util.JSONToTags(tags.String)
		}
		if links.Valid && links.String != "" && links.String != "[]" {
			_ = json.Unmarshal([]byte(links.String), &g.Links)
		}
		if completedAt.Valid {
			val := completedAt.Time.Format(time.RFC3339)
			g.CompletedAt = &val
		}
		if archivedAt.Valid {
			val := archivedAt.Time.Format(time.RFC3339)
			g.ArchivedAt = &val
		}
		if taskStarted.Valid {
			val := taskStarted.Time.Format(time.RFC3339)
			g.TaskStartedAt = &val
		}
		g.TaskActive = taskActive == 1
		out = append(out, g)
	}
	return out, nil
}

func GetAllJournalEntriesExport() ([]ExportJournalEntry, error) {
	d, err := getDefaultDB()
	if err != nil {
		return nil, err
	}
	return d.GetAllJournalEntriesExport()
}

func (d *Database) GetAllJournalEntriesExport() ([]ExportJournalEntry, error) {
	rows, err := d.DB.Query(`
		SELECT id, day_id, workspace_id, sprint_id, goal_id, content, tags, created_at
		FROM journal_entries ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExportJournalEntry
	for rows.Next() {
		var e ExportJournalEntry
		var workspaceID, sprintID, goalID sql.NullInt64
		var tags sql.NullString
		if err := rows.Scan(&e.ID, &e.DayID, &workspaceID, &sprintID, &goalID, &e.Content, &tags, &e.CreatedAt); err != nil {
			return nil, err
		}
		if workspaceID.Valid {
			id := workspaceID.Int64
			e.WorkspaceID = &id
		}
		if sprintID.Valid {
			id := sprintID.Int64
			e.SprintID = &id
		}
		if goalID.Valid {
			id := goalID.Int64
			e.GoalID = &id
		}
		if tags.Valid && tags.String != "" && tags.String != "[]" {
			e.Tags = util.JSONToTags(tags.String)
		}
		out = append(out, e)
	}
	return out, nil
}

func GetAllTaskDeps() ([]ExportTaskDep, error) {
	d, err := getDefaultDB()
	if err != nil {
		return nil, err
	}
	return d.GetAllTaskDeps()
}

func (d *Database) GetAllTaskDeps() ([]ExportTaskDep, error) {
	rows, err := d.DB.Query(`SELECT goal_id, depends_on_id FROM task_deps ORDER BY goal_id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExportTaskDep
	for rows.Next() {
		var dep ExportTaskDep
		if err := rows.Scan(&dep.GoalID, &dep.DependsOnID); err != nil {
			return nil, err
		}
		out = append(out, dep)
	}
	return out, nil
}
