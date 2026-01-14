package testutil

import (
	"time"

	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
)

// GoalBuilder provides fluent API for creating test goals.
type GoalBuilder struct {
	goal models.Goal
}

func NewGoal() *GoalBuilder {
	effort := "M"
	return &GoalBuilder{
		goal: models.Goal{
			Description: "Test Goal",
			Priority:    2,
			Effort:      &effort,
			Status:      models.GoalStatusPending,
			CreatedAt:   time.Now(),
		},
	}
}

func (b *GoalBuilder) WithDescription(d string) *GoalBuilder {
	b.goal.Description = d
	return b
}

func (b *GoalBuilder) WithPriority(p int) *GoalBuilder {
	b.goal.Priority = p
	return b
}

func (b *GoalBuilder) WithTags(tags ...string) *GoalBuilder {
	jsonTags := util.TagsToJSON(tags)
	b.goal.Tags = &jsonTags
	return b
}

func (b *GoalBuilder) WithStatus(s models.GoalStatus) *GoalBuilder {
	b.goal.Status = s
	return b
}

func (b *GoalBuilder) Build() models.Goal {
	return b.goal
}

// SprintBuilder provides fluent API for creating test sprints.
type SprintBuilder struct {
	sprint models.Sprint
}

func NewSprint() *SprintBuilder {
	return &SprintBuilder{
		sprint: models.Sprint{
			SprintNumber:   1,
			Status:         models.StatusPending,
			ElapsedSeconds: 0,
		},
	}
}

func (b *SprintBuilder) WithSprintNumber(n int) *SprintBuilder {
	b.sprint.SprintNumber = n
	return b
}

func (b *SprintBuilder) WithStatus(s models.SprintStatus) *SprintBuilder {
	b.sprint.Status = s
	return b
}

func (b *SprintBuilder) WithDayID(id int64) *SprintBuilder {
	b.sprint.DayID = id
	return b
}

func (b *SprintBuilder) WithWorkspaceID(id int64) *SprintBuilder {
	b.sprint.WorkspaceID = &id
	return b
}

func (b *SprintBuilder) Build() models.Sprint {
	return b.sprint
}
