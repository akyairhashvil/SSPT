package database

import (
	"context"

	"github.com/akyairhashvil/SSPT/internal/models"
)

// GoalRepository defines goal-related database operations.
type GoalRepository interface {
	AddGoal(ctx context.Context, workspaceID int64, description string, sprintID int64) error
	AddGoalDetailed(ctx context.Context, workspaceID int64, sprintID int64, seed GoalSeed) error
	AddSubtask(ctx context.Context, description string, parentID int64) error
	AddSubtaskDetailed(ctx context.Context, parentID int64, seed GoalSeed) error
	UpdateGoalStatus(ctx context.Context, goalID int64, status models.GoalStatus) error
	DeleteGoal(ctx context.Context, id int64) error
	GetBacklogGoals(ctx context.Context, workspaceID int64) ([]models.Goal, error)
	GetGoalsForSprint(ctx context.Context, sprintID int64) ([]models.Goal, error)
}

// SprintRepository defines sprint-related database operations.
type SprintRepository interface {
	BootstrapDay(ctx context.Context, workspaceID int64, numSprints int) error
	GetSprints(ctx context.Context, dayID int64, workspaceID int64) ([]models.Sprint, error)
	StartSprint(ctx context.Context, sprintID int64) error
	PauseSprint(ctx context.Context, sprintID int64, elapsedSeconds int) error
	CompleteSprint(ctx context.Context, sprintID int64) error
	ResetSprint(ctx context.Context, sprintID int64) error
	AppendSprint(ctx context.Context, dayID int64, workspaceID int64) error
	RemoveLastSprint(ctx context.Context, dayID int64, workspaceID int64) error
}

// WorkspaceRepository defines workspace-related database operations.
type WorkspaceRepository interface {
	GetWorkspaces(ctx context.Context) ([]models.Workspace, error)
	EnsureDefaultWorkspace(ctx context.Context) (int64, error)
	CreateWorkspace(ctx context.Context, name, slug string) (int64, error)
}

// Repository combines all repository interfaces.
//
//go:generate mockgen -source=interface.go -destination=mock_repository_test.go -package=database
type Repository interface {
	GoalRepository
	SprintRepository
	WorkspaceRepository
}

var _ Repository = (*Database)(nil)
