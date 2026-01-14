package database

import (
	"context"

	"github.com/akyairhashvil/SSPT/internal/util"
)

func (d *Database) AddTagsToGoal(ctx context.Context, goalID int64, tagsToAdd []string) error {
	return d.withDBContext(ctx, func(ctx context.Context) error {
		if len(tagsToAdd) == 0 {
			return nil
		}

		var existingTags *string
		if err := d.DB.QueryRowContext(ctx, "SELECT tags FROM goals WHERE id = ?", goalID).Scan(&existingTags); err != nil {
			return wrapErr(EntityGoal, "add tags", goalID, err)
		}

		var tags []string
		if existingTags != nil {
			tags = util.JSONToTags(*existingTags)
		}
		tags = append(tags, tagsToAdd...)
		tags = normalizeTagsFromSlice(tags)
		tagsJSON := util.TagsToJSON(tags)

		_, err := d.DB.ExecContext(ctx, "UPDATE goals SET tags = ? WHERE id = ?", tagsJSON, goalID)
		return wrapErr(EntityGoal, "add tags", goalID, err)
	})
}

func (d *Database) SetGoalTags(ctx context.Context, goalID int64, tags []string) error {
	return d.withDBContext(ctx, func(ctx context.Context) error {
		tagsJSON := util.TagsToJSON(normalizeTagsFromSlice(tags))
		_, err := d.DB.ExecContext(ctx, "UPDATE goals SET tags = ? WHERE id = ?", tagsJSON, goalID)
		return wrapErr(EntityGoal, "set tags", goalID, err)
	})
}
