package database

import (
	"context"
	"testing"

	"github.com/akyairhashvil/SSPT/internal/util"
)

func TestGoalSearch(t *testing.T) {
	ctx := context.Background()
	builder := NewTestDataBuilder(t).
		WithWorkspace("Work").
		WithSprints(3).
		WithGoals(5)
	db := builder.Build()
	defer db.Close()

	query := util.SearchQuery{Text: []string{"Goal"}}
	results, err := db.Search(ctx, query, builder.PrimaryWorkspaceID())
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 15 {
		t.Fatalf("expected 15 results, got %d", len(results))
	}
}
