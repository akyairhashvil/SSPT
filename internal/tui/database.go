package tui

import (
	"context"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
)

// Database defines the persistence methods the TUI requires.
type Database interface {
	EncryptionStatus() database.EncryptionInfo
	EncryptDatabase(ctx context.Context, key string) error
	DatabaseHasData(ctx context.Context) bool
	RecreateEncryptedDatabase(ctx context.Context, key string) error
	RekeyDB(ctx context.Context, key string) error
	ClearDatabase(ctx context.Context) error

	GetSetting(ctx context.Context, key string) (string, bool)
	SetSetting(ctx context.Context, key, value string) error

	GetWorkspaces(ctx context.Context) ([]models.Workspace, error)
	EnsureDefaultWorkspace(ctx context.Context) (int64, error)
	CreateWorkspace(ctx context.Context, name, slug string) (int64, error)
	GetWorkspaceIDBySlug(ctx context.Context, slug string) (int64, bool, error)
	UpdateWorkspaceViewMode(ctx context.Context, workspaceID int64, mode int) error
	UpdateWorkspaceTheme(ctx context.Context, workspaceID int64, theme string) error
	UpdateWorkspacePaneVisibility(ctx context.Context, workspaceID int64, showBacklog, showCompleted, showArchived bool) error

	CheckCurrentDay(ctx context.Context) int64
	BootstrapDay(ctx context.Context, workspaceID int64, numSprints int) error
	GetDay(ctx context.Context, id int64) (models.Day, error)
	GetAdjacentDay(ctx context.Context, currentDayID int64, direction int) (int64, string, error)
	GetSprints(ctx context.Context, dayID int64, workspaceID int64) ([]models.Sprint, error)
	AppendSprint(ctx context.Context, dayID int64, workspaceID int64) error
	RemoveLastSprint(ctx context.Context, dayID int64, workspaceID int64) error
	GetSprintGoalCounts(ctx context.Context, sprintID int64) (int, int, error)
	StartSprint(ctx context.Context, sprintID int64) error
	PauseSprint(ctx context.Context, sprintID int64, elapsedSeconds int) error
	CompleteSprint(ctx context.Context, sprintID int64) error
	ResetSprint(ctx context.Context, sprintID int64) error
	MovePendingToBacklog(ctx context.Context, sprintID int64) error

	AddGoal(ctx context.Context, workspaceID int64, description string, sprintID int64) error
	AddGoalDetailed(ctx context.Context, workspaceID int64, sprintID int64, seed database.GoalSeed) error
	AddSubtask(ctx context.Context, description string, parentID int64) error
	AddSubtaskDetailed(ctx context.Context, parentID int64, seed database.GoalSeed) error
	EditGoal(ctx context.Context, goalID int64, newDescription string) error
	DeleteGoal(ctx context.Context, goalID int64) error
	MoveGoal(ctx context.Context, goalID int64, targetSprintID int64) error
	UpdateGoalPriority(ctx context.Context, goalID int64, priority int) error
	UpdateGoalStatus(ctx context.Context, goalID int64, status models.GoalStatus) error
	UpdateGoalRecurrence(ctx context.Context, goalID int64, rule string) error
	SetGoalTags(ctx context.Context, goalID int64, tags []string) error
	SetGoalDependencies(ctx context.Context, goalID int64, deps []int64) error
	GetGoalByID(ctx context.Context, goalID int64) (models.Goal, error)
	GetGoalDependencies(ctx context.Context, goalID int64) (map[int64]bool, error)
	IsGoalBlocked(ctx context.Context, goalID int64) (bool, error)
	GetBlockedGoalIDs(ctx context.Context, workspaceID int64) (map[int64]bool, error)
	ArchiveGoal(ctx context.Context, goalID int64) error
	UnarchiveGoal(ctx context.Context, goalID int64) error
	StartTaskTimer(ctx context.Context, goalID int64) error
	PauseTaskTimer(ctx context.Context, goalID int64) error

	GetBacklogGoals(ctx context.Context, workspaceID int64) ([]models.Goal, error)
	GetGoalsForSprint(ctx context.Context, sprintID int64) ([]models.Goal, error)
	GetArchivedGoals(ctx context.Context, workspaceID int64) ([]models.Goal, error)
	GetCompletedGoalsForDay(ctx context.Context, dayID int64, workspaceID int64) ([]models.Goal, error)
	GetAllGoals(ctx context.Context) ([]models.Goal, error)
	GetActiveTask(ctx context.Context, workspaceID int64) (*models.Goal, error)
	GoalExistsDetailed(ctx context.Context, workspaceID int64, sprintID int64, parentID *int64, seed database.GoalSeed) (bool, error)
	GetLastGoalID(ctx context.Context) (int64, error)
	Search(ctx context.Context, query util.SearchQuery, workspaceID int64) ([]models.Goal, error)

	AddJournalEntry(ctx context.Context, dayID int64, workspaceID int64, sprintID *int64, goalID *int64, content string) error
	GetJournalEntries(ctx context.Context, dayID int64, workspaceID int64) ([]models.JournalEntry, error)

	GetAllDays(ctx context.Context) ([]database.ExportDay, error)
	GetAllSprintsFlat(ctx context.Context) ([]database.ExportSprint, error)
	GetAllGoalsExport(ctx context.Context) ([]database.ExportGoal, error)
	GetAllJournalEntriesExport(ctx context.Context) ([]database.ExportJournalEntry, error)
	GetAllTaskDeps(ctx context.Context) ([]database.ExportTaskDep, error)
}
