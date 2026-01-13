package database

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
)

const goalColumnsWithSprint = `id, parent_id, sprint_id, description, status, rank, priority, effort, tags, recurrence_rule, created_at, archived_at, task_started_at, task_elapsed_seconds, task_active`

// scanGoalWithSprint scans a database row into a Goal struct.
// The row parameter accepts any type with a Scan method (sql.Row or sql.Rows).
//
// Expected columns (in order):
//
//	id, parent_id, sprint_id, description, status, rank, priority, effort, tags,
//	recurrence_rule, created_at, archived_at, task_started_at, task_elapsed_seconds,
//	task_active
//
// Returns ErrNoRows if the row is empty.
func scanGoalWithSprint(row interface{ Scan(...interface{}) error }) (models.Goal, error) {
	var g models.Goal
	var active int
	if err := row.Scan(
		&g.ID,
		&g.ParentID,
		&g.SprintID,
		&g.Description,
		&g.Status,
		&g.Rank,
		&g.Priority,
		&g.Effort,
		&g.Tags,
		&g.RecurrenceRule,
		&g.CreatedAt,
		&g.ArchivedAt,
		&g.TaskStartedAt,
		&g.TaskElapsedSec,
		&active,
	); err != nil {
		return models.Goal{}, err
	}
	g.TaskActive = active == 1
	return g, nil
}

func normalizeSeed(seed GoalSeed) (string, int, string, []string, string, string, []string) {
	desc := strings.TrimSpace(seed.Description)
	priority := normalizePriority(seed.Priority)
	effort := normalizeEffort(seed.Effort)
	recurrence := strings.TrimSpace(seed.Recurrence)
	notes := strings.TrimSpace(seed.Notes)
	tags := seed.Tags
	if len(tags) == 0 && desc != "" {
		tags = util.ExtractTags(desc)
	}
	tags = normalizeTagsFromSlice(tags)
	links := normalizeLinksFromSlice(seed.Links)
	return desc, priority, effort, tags, recurrence, notes, links
}

func normalizePriority(priority int) int {
	if priority <= 0 {
		return 3
	}
	if priority > 5 {
		return 5
	}
	return priority
}

func normalizeEffort(effort string) string {
	effort = strings.ToUpper(strings.TrimSpace(effort))
	if effort == "" {
		return "M"
	}
	return effort
}

func normalizeTagsFromSlice(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		val := strings.TrimSpace(strings.ToLower(strings.TrimPrefix(t, "#")))
		if val != "" {
			out = append(out, val)
		}
	}
	sort.Strings(out)
	return out
}

func normalizeTags(tagsJSON string) []string {
	if strings.TrimSpace(tagsJSON) == "" || tagsJSON == "[]" {
		return nil
	}
	return normalizeTagsFromSlice(util.JSONToTags(tagsJSON))
}

func normalizeLinksFromSlice(links []string) []string {
	if len(links) == 0 {
		return nil
	}
	out := make([]string, 0, len(links))
	for _, link := range links {
		val := strings.TrimSpace(link)
		if val != "" {
			out = append(out, val)
		}
	}
	sort.Strings(out)
	return out
}

func normalizeLinks(linksJSON string) []string {
	if strings.TrimSpace(linksJSON) == "" || linksJSON == "[]" {
		return nil
	}
	var links []string
	if err := json.Unmarshal([]byte(linksJSON), &links); err != nil {
		return nil
	}
	return normalizeLinksFromSlice(links)
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type GoalSeed struct {
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	Effort      string   `json:"effort,omitempty"`
	Notes       string   `json:"notes,omitempty"`
	Recurrence  string   `json:"recurrence,omitempty"`
	Links       []string `json:"links,omitempty"`
}
