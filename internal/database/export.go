package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

type ExportOptions struct {
	EncryptOutput bool
	Passphrase    string
}

type VaultExport struct {
	Workspaces []ExportWorkspace    `json:"workspaces"`
	Days       []ExportDay          `json:"days"`
	Sprints    []ExportSprint       `json:"sprints"`
	Goals      []ExportGoal         `json:"goals"`
	Journal    []ExportJournalEntry `json:"journal_entries"`
	TaskDeps   []ExportTaskDep      `json:"task_deps"`
}

func (d *Database) GetAllDays(ctx context.Context) ([]ExportDay, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	rows, err := d.DB.QueryContext(ctx, "SELECT id, date, started_at FROM days ORDER BY id ASC")
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

func (d *Database) GetAllSprintsFlat(ctx context.Context) ([]ExportSprint, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	rows, err := d.DB.QueryContext(ctx, `
		SELECT id, day_id, workspace_id, sprint_number, status, start_time, end_time, last_paused_at, elapsed_seconds
		FROM sprints ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExportSprint
	for rows.Next() {
		var s ExportSprint
		var wsID *int64
		var start, end, last *time.Time
		if err := rows.Scan(&s.ID, &s.DayID, &wsID, &s.SprintNumber, &s.Status, &start, &end, &last, &s.ElapsedSeconds); err != nil {
			return nil, err
		}
		if wsID != nil {
			id := *wsID
			s.WorkspaceID = &id
		}
		if start != nil {
			val := start.Format(time.RFC3339)
			s.StartTime = &val
		}
		if end != nil {
			val := end.Format(time.RFC3339)
			s.EndTime = &val
		}
		if last != nil {
			val := last.Format(time.RFC3339)
			s.LastPausedAt = &val
		}
		out = append(out, s)
	}
	return out, nil
}

func (d *Database) GetAllGoalsExport(ctx context.Context) ([]ExportGoal, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	rows, err := d.DB.QueryContext(ctx, `
		SELECT id, parent_id, workspace_id, sprint_id, description, notes, status, priority, effort, tags, recurrence_rule, links, rank, created_at, completed_at, archived_at, task_started_at, task_elapsed_seconds, task_active
		FROM goals ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExportGoal
	for rows.Next() {
		var g ExportGoal
		var parentID, workspaceID, sprintID *int64
		var notes, effort, recurrence, tags, links *string
		var completedAt, archivedAt, taskStarted *time.Time
		var taskActive int
		if err := rows.Scan(&g.ID, &parentID, &workspaceID, &sprintID, &g.Description, &notes, &g.Status, &g.Priority, &effort, &tags, &recurrence, &links, &g.Rank, &g.CreatedAt, &completedAt, &archivedAt, &taskStarted, &g.TaskElapsedSec, &taskActive); err != nil {
			return nil, err
		}
		if parentID != nil {
			id := *parentID
			g.ParentID = &id
		}
		if workspaceID != nil {
			id := *workspaceID
			g.WorkspaceID = &id
		}
		if sprintID != nil {
			id := *sprintID
			g.SprintID = &id
		}
		if notes != nil {
			val := *notes
			g.Notes = &val
		}
		if effort != nil {
			val := *effort
			g.Effort = &val
		}
		if recurrence != nil {
			val := *recurrence
			g.RecurrenceRule = &val
		}
		if tags != nil && *tags != "" && *tags != "[]" {
			g.Tags = util.JSONToTags(*tags)
		}
		if links != nil && *links != "" && *links != "[]" {
			if err := json.Unmarshal([]byte(*links), &g.Links); err != nil {
				return nil, err
			}
		}
		if completedAt != nil {
			val := completedAt.Format(time.RFC3339)
			g.CompletedAt = &val
		}
		if archivedAt != nil {
			val := archivedAt.Format(time.RFC3339)
			g.ArchivedAt = &val
		}
		if taskStarted != nil {
			val := taskStarted.Format(time.RFC3339)
			g.TaskStartedAt = &val
		}
		g.TaskActive = taskActive == 1
		out = append(out, g)
	}
	return out, nil
}

func (d *Database) GetAllJournalEntriesExport(ctx context.Context) ([]ExportJournalEntry, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	rows, err := d.DB.QueryContext(ctx, `
		SELECT id, day_id, workspace_id, sprint_id, goal_id, content, tags, created_at
		FROM journal_entries ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExportJournalEntry
	for rows.Next() {
		var e ExportJournalEntry
		var workspaceID, sprintID, goalID *int64
		var tags *string
		if err := rows.Scan(&e.ID, &e.DayID, &workspaceID, &sprintID, &goalID, &e.Content, &tags, &e.CreatedAt); err != nil {
			return nil, err
		}
		if workspaceID != nil {
			id := *workspaceID
			e.WorkspaceID = &id
		}
		if sprintID != nil {
			id := *sprintID
			e.SprintID = &id
		}
		if goalID != nil {
			id := *goalID
			e.GoalID = &id
		}
		if tags != nil && *tags != "" && *tags != "[]" {
			e.Tags = util.JSONToTags(*tags)
		}
		out = append(out, e)
	}
	return out, nil
}

func (d *Database) GetAllTaskDeps(ctx context.Context) ([]ExportTaskDep, error) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	rows, err := d.DB.QueryContext(ctx, `SELECT goal_id, depends_on_id FROM task_deps ORDER BY goal_id ASC`)
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

func (d *Database) ExportVault(ctx context.Context, opts ExportOptions) ([]byte, error) {
	workspaces, err := d.GetWorkspaces(ctx)
	if err != nil {
		return nil, err
	}
	exportWorkspaces := make([]ExportWorkspace, 0, len(workspaces))
	for _, ws := range workspaces {
		exportWorkspaces = append(exportWorkspaces, ExportWorkspace{
			ID:            ws.ID,
			Name:          ws.Name,
			Slug:          ws.Slug,
			ViewMode:      ws.ViewMode,
			Theme:         ws.Theme,
			ShowBacklog:   ws.ShowBacklog,
			ShowCompleted: ws.ShowCompleted,
			ShowArchived:  ws.ShowArchived,
		})
	}
	days, err := d.GetAllDays(ctx)
	if err != nil {
		return nil, err
	}
	sprints, err := d.GetAllSprintsFlat(ctx)
	if err != nil {
		return nil, err
	}
	goals, err := d.GetAllGoalsExport(ctx)
	if err != nil {
		return nil, err
	}
	journal, err := d.GetAllJournalEntriesExport(ctx)
	if err != nil {
		return nil, err
	}
	deps, err := d.GetAllTaskDeps(ctx)
	if err != nil {
		return nil, err
	}

	export := VaultExport{
		Workspaces: exportWorkspaces,
		Days:       days,
		Sprints:    sprints,
		Goals:      goals,
		Journal:    journal,
		TaskDeps:   deps,
	}
	jsonData, err := json.Marshal(export)
	if err != nil {
		return nil, err
	}
	if opts.EncryptOutput && opts.Passphrase != "" {
		return encryptData(jsonData, opts.Passphrase)
	}
	return jsonData, nil
}

// ImportVault loads exported data into the database.
func (d *Database) ImportVault(ctx context.Context, payload []byte) error {
	var export VaultExport
	if err := json.Unmarshal(payload, &export); err != nil {
		return fmt.Errorf("import vault: %w", err)
	}

	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()

	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("import vault begin: %w", err)
	}
	commit := false
	defer func() {
		if !commit {
			_ = tx.Rollback()
		}
	}()

	for _, ws := range export.Workspaces {
		if _, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO workspaces
			(id, name, slug, view_mode, theme, show_backlog, show_completed, show_archived)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			ws.ID, ws.Name, ws.Slug, ws.ViewMode, ws.Theme,
			util.BoolToInt(ws.ShowBacklog), util.BoolToInt(ws.ShowCompleted), util.BoolToInt(ws.ShowArchived),
		); err != nil {
			return fmt.Errorf("import workspace %d: %w", ws.ID, err)
		}
	}

	for _, day := range export.Days {
		startedAt := day.StartedAt
		if strings.TrimSpace(startedAt) == "" {
			startedAt = ""
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO days (id, date, started_at)
			VALUES (?, ?, ?)`,
			day.ID, day.Date, startedAt,
		); err != nil {
			return fmt.Errorf("import day %d: %w", day.ID, err)
		}
	}

	for _, sprint := range export.Sprints {
		status := sprint.Status
		if strings.TrimSpace(status) == "" {
			status = "pending"
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO sprints
			(id, day_id, workspace_id, sprint_number, status, start_time, end_time, last_paused_at, elapsed_seconds)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			sprint.ID, sprint.DayID, sprint.WorkspaceID, sprint.SprintNumber, status,
			sprint.StartTime, sprint.EndTime, sprint.LastPausedAt, sprint.ElapsedSeconds,
		); err != nil {
			return fmt.Errorf("import sprint %d: %w", sprint.ID, err)
		}
	}

	for _, goal := range export.Goals {
		status := goal.Status
		if strings.TrimSpace(status) == "" {
			status = "pending"
		}
		tags := ""
		if len(goal.Tags) > 0 {
			tags = util.TagsToJSON(goal.Tags)
		}
		links := ""
		if len(goal.Links) > 0 {
			if raw, err := json.Marshal(goal.Links); err == nil {
				links = string(raw)
			} else {
				return fmt.Errorf("import goal %d links: %w", goal.ID, err)
			}
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO goals
			(id, parent_id, workspace_id, sprint_id, description, notes, status, priority, effort, tags, recurrence_rule, links, rank,
			 created_at, completed_at, archived_at, task_started_at, task_elapsed_seconds, task_active)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			goal.ID, goal.ParentID, goal.WorkspaceID, goal.SprintID, goal.Description, goal.Notes, status,
			goal.Priority, goal.Effort, nilIfEmpty(tags), nilIfEmptyPtr(goal.RecurrenceRule), nilIfEmpty(links),
			goal.Rank, goal.CreatedAt, goal.CompletedAt, goal.ArchivedAt, goal.TaskStartedAt,
			goal.TaskElapsedSec, util.BoolToInt(goal.TaskActive),
		); err != nil {
			return fmt.Errorf("import goal %d: %w", goal.ID, err)
		}
	}

	for _, entry := range export.Journal {
		tags := ""
		if len(entry.Tags) > 0 {
			tags = util.TagsToJSON(entry.Tags)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO journal_entries
			(id, day_id, workspace_id, sprint_id, goal_id, content, tags, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			entry.ID, entry.DayID, entry.WorkspaceID, entry.SprintID, entry.GoalID, entry.Content,
			nilIfEmpty(tags), entry.CreatedAt,
		); err != nil {
			return fmt.Errorf("import journal entry %d: %w", entry.ID, err)
		}
	}

	for _, dep := range export.TaskDeps {
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO task_deps (goal_id, depends_on_id)
			VALUES (?, ?)`,
			dep.GoalID, dep.DependsOnID,
		); err != nil {
			return fmt.Errorf("import dependency %d->%d: %w", dep.GoalID, dep.DependsOnID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("import vault commit: %w", err)
	}
	commit = true
	return nil
}

func nilIfEmpty(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nilIfEmptyPtr(value *string) interface{} {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return *value
}
