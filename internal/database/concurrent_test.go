package database

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestConcurrentGoalUpdates(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t, ctx)
	defer db.Close()

	wsID, err := db.EnsureDefaultWorkspace(ctx)
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	if err := db.AddGoal(ctx, wsID, "Test", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	goals, err := db.GetBacklogGoals(ctx, wsID)
	if err != nil {
		t.Fatalf("GetBacklogGoals failed: %v", err)
	}
	if len(goals) == 0 {
		t.Fatalf("expected goal to update")
	}
	goalID := goals[0].ID

	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := db.EditGoal(ctx, goalID, fmt.Sprintf("Title %d", i)); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent update failed: %v", err)
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t, ctx)
	defer db.Close()

	wsID, err := db.EnsureDefaultWorkspace(ctx)
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	if err := db.AddGoal(ctx, wsID, "Concurrent", 0); err != nil {
		t.Fatalf("AddGoal failed: %v", err)
	}
	goals, err := db.GetBacklogGoals(ctx, wsID)
	if err != nil {
		t.Fatalf("GetBacklogGoals failed: %v", err)
	}
	if len(goals) == 0 {
		t.Fatalf("expected goal to update")
	}
	goalID := goals[0].ID

	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := db.EditGoal(ctx, goalID, fmt.Sprintf("Update %d", i)); err != nil {
				errs <- err
			}
		}(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := db.GetBacklogGoals(ctx, wsID); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent read/write failed: %v", err)
	}
}

func TestConcurrentWorkspaceIsolation(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t, ctx)
	defer db.Close()

	wsID1, err := db.EnsureDefaultWorkspace(ctx)
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspace failed: %v", err)
	}
	wsID2, err := db.CreateWorkspace(ctx, "Second", "second")
	if err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := db.AddGoal(ctx, wsID1, fmt.Sprintf("WS1-%d", i), 0); err != nil {
				errs <- err
			}
		}(i)
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := db.AddGoal(ctx, wsID2, fmt.Sprintf("WS2-%d", i), 0); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent workspace write failed: %v", err)
	}

	ws1Goals, err := db.GetBacklogGoals(ctx, wsID1)
	if err != nil {
		t.Fatalf("GetBacklogGoals ws1 failed: %v", err)
	}
	ws2Goals, err := db.GetBacklogGoals(ctx, wsID2)
	if err != nil {
		t.Fatalf("GetBacklogGoals ws2 failed: %v", err)
	}
	if len(ws1Goals) == 0 || len(ws2Goals) == 0 {
		t.Fatalf("expected goals in both workspaces")
	}
	for _, g := range ws1Goals {
		if !strings.HasPrefix(g.Description, "WS1-") {
			t.Fatalf("expected ws1 goal to stay in ws1, got description %q", g.Description)
		}
	}
	for _, g := range ws2Goals {
		if !strings.HasPrefix(g.Description, "WS2-") {
			t.Fatalf("expected ws2 goal to stay in ws2, got description %q", g.Description)
		}
	}
}
