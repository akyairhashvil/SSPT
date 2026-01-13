package database

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type TestDataBuilder struct {
	t            *testing.T
	ctx          context.Context
	db           *Database
	workspaceIDs []int64
	sprintIDs    []int64
	goalIDs      []int64
}

func NewTestDataBuilder(t *testing.T) *TestDataBuilder {
	t.Helper()
	ctx := context.Background()
	db := setupTestDB(t, ctx)
	return &TestDataBuilder{t: t, ctx: ctx, db: db}
}

func (b *TestDataBuilder) WithWorkspace(name string) *TestDataBuilder {
	b.t.Helper()
	slug := strings.ToLower(name)
	id, err := b.db.CreateWorkspace(b.ctx, name, slug)
	if err != nil {
		b.t.Fatalf("CreateWorkspace failed: %v", err)
	}
	b.workspaceIDs = append(b.workspaceIDs, id)
	return b
}

func (b *TestDataBuilder) WithSprints(count int) *TestDataBuilder {
	b.t.Helper()
	if len(b.workspaceIDs) == 0 {
		id, err := b.db.EnsureDefaultWorkspace(b.ctx)
		if err != nil {
			b.t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
		}
		b.workspaceIDs = append(b.workspaceIDs, id)
	}
	wsID := b.workspaceIDs[0]
	if err := b.db.BootstrapDay(b.ctx, wsID, count); err != nil {
		b.t.Fatalf("BootstrapDay failed: %v", err)
	}
	dayID := b.db.CheckCurrentDay(b.ctx)
	if dayID == 0 {
		b.t.Fatalf("CheckCurrentDay returned zero ID")
	}
	sprints, err := b.db.GetSprints(b.ctx, dayID, wsID)
	if err != nil {
		b.t.Fatalf("GetSprints failed: %v", err)
	}
	for _, sprint := range sprints {
		b.sprintIDs = append(b.sprintIDs, sprint.ID)
	}
	return b
}

func (b *TestDataBuilder) WithGoals(perSprint int) *TestDataBuilder {
	b.t.Helper()
	if len(b.sprintIDs) == 0 {
		b.WithSprints(1)
	}
	wsID := b.workspaceIDs[0]
	for sprintIdx, sprintID := range b.sprintIDs {
		for i := 0; i < perSprint; i++ {
			description := fmt.Sprintf("Goal %d-%d", sprintIdx+1, i+1)
			if err := b.db.AddGoal(b.ctx, wsID, description, sprintID); err != nil {
				b.t.Fatalf("AddGoal failed: %v", err)
			}
		}
	}
	return b
}

func (b *TestDataBuilder) Build() *Database {
	return b.db
}

func (b *TestDataBuilder) PrimaryWorkspaceID() int64 {
	if len(b.workspaceIDs) == 0 {
		return 0
	}
	return b.workspaceIDs[0]
}
