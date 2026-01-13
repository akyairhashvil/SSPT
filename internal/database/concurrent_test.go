package database

import (
	"context"
	"fmt"
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
