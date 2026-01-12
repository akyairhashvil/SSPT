package database

import (
	"database/sql"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
)

// --- Dependencies ---

func AddGoalDependency(goalID, dependsOnID int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.AddGoalDependency(goalID, dependsOnID)
}

func (d *Database) AddGoalDependency(goalID, dependsOnID int64) error {
	goalWS, ok := d.getGoalWorkspaceID(goalID)
	if !ok {
		return nil
	}
	depWS, ok := d.getGoalWorkspaceID(dependsOnID)
	if !ok || depWS != goalWS {
		return nil
	}
	_, err := d.DB.Exec("INSERT OR IGNORE INTO task_deps (goal_id, depends_on_id) VALUES (?, ?)", goalID, dependsOnID)
	return err
}

func RemoveGoalDependency(goalID, dependsOnID int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.RemoveGoalDependency(goalID, dependsOnID)
}

func (d *Database) RemoveGoalDependency(goalID, dependsOnID int64) error {
	_, err := d.DB.Exec("DELETE FROM task_deps WHERE goal_id = ? AND depends_on_id = ?", goalID, dependsOnID)
	return err
}

func GetGoalDependencies(goalID int64) (map[int64]bool, error) {
	d, err := getDefaultDB()
	if err != nil {
		return nil, err
	}
	return d.GetGoalDependencies(goalID)
}

func (d *Database) GetGoalDependencies(goalID int64) (map[int64]bool, error) {
	rows, err := d.DB.Query("SELECT depends_on_id FROM task_deps WHERE goal_id = ?", goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deps := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		deps[id] = true
	}
	return deps, nil
}

func SetGoalDependencies(goalID int64, deps []int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.SetGoalDependencies(goalID, deps)
}

func (d *Database) SetGoalDependencies(goalID int64, deps []int64) error {
	goalWS, ok := d.getGoalWorkspaceID(goalID)
	if !ok {
		return nil
	}
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM task_deps WHERE goal_id = ?", goalID); err != nil {
		return rollbackWithLog(tx, err)
	}
	for _, id := range deps {
		if id == goalID {
			continue
		}
		depWS, ok := d.getGoalWorkspaceID(id)
		if !ok || depWS != goalWS {
			continue
		}
		if _, err := tx.Exec("INSERT OR IGNORE INTO task_deps (goal_id, depends_on_id) VALUES (?, ?)", goalID, id); err != nil {
			return rollbackWithLog(tx, err)
		}
	}
	return tx.Commit()
}

func (d *Database) regenerateRecurringGoal(goalID int64) error {
	var g models.Goal
	err := d.DB.QueryRow(`
		SELECT id, description, workspace_id, sprint_id, notes, priority, effort, tags, recurrence_rule
		FROM goals WHERE id = ?`, goalID).Scan(
		&g.ID, &g.Description, &g.WorkspaceID, &g.SprintID, &g.Notes, &g.Priority, &g.Effort, &g.Tags, &g.RecurrenceRule,
	)
	if err != nil {
		return err
	}
	if !g.RecurrenceRule.Valid || strings.TrimSpace(g.RecurrenceRule.String) == "" {
		return nil
	}
	rule := strings.ToLower(strings.TrimSpace(g.RecurrenceRule.String))
	if rule != "daily" && !strings.HasPrefix(rule, "weekly:") && !strings.HasPrefix(rule, "monthly:") {
		return nil
	}

	// Regenerate into backlog so it surfaces even if sprint is completed.
	var maxRank int
	if g.WorkspaceID.Valid {
		_ = d.DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id IS NULL AND workspace_id = ?", g.WorkspaceID.Int64).Scan(&maxRank)
	}
	var wsID interface{} = nil
	if g.WorkspaceID.Valid {
		wsID = g.WorkspaceID.Int64
	}
	_, err = d.DB.Exec(`INSERT INTO goals (workspace_id, description, sprint_id, status, rank, tags, notes, priority, effort, recurrence_rule)
		VALUES (?, ?, NULL, 'pending', ?, ?, ?, ?, ?, ?)`,
		wsID, g.Description, maxRank+1, g.Tags, g.Notes, g.Priority, g.Effort, g.RecurrenceRule,
	)
	return err
}

func (d *Database) getGoalWorkspaceID(goalID int64) (int64, bool) {
	var wsID sql.NullInt64
	err := d.DB.QueryRow("SELECT workspace_id FROM goals WHERE id = ?", goalID).Scan(&wsID)
	if err != nil || !wsID.Valid {
		return 0, false
	}
	return wsID.Int64, true
}

func IsGoalBlocked(goalID int64) (bool, error) {
	d, err := getDefaultDB()
	if err != nil {
		return false, err
	}
	return d.IsGoalBlocked(goalID)
}

func (d *Database) IsGoalBlocked(goalID int64) (bool, error) {
	var count int
	err := d.DB.QueryRow(`
		SELECT COUNT(1)
		FROM task_deps td
		JOIN goals g ON td.depends_on_id = g.id
		WHERE td.goal_id = ? AND g.status != 'completed'`, goalID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func GetBlockedGoalIDs(workspaceID int64) (map[int64]bool, error) {
	d, err := getDefaultDB()
	if err != nil {
		return nil, err
	}
	return d.GetBlockedGoalIDs(workspaceID)
}

func (d *Database) GetBlockedGoalIDs(workspaceID int64) (map[int64]bool, error) {
	rows, err := d.DB.Query(`
		SELECT DISTINCT td.goal_id
		FROM task_deps td
		JOIN goals g ON td.depends_on_id = g.id
		JOIN goals gg ON td.goal_id = gg.id
		WHERE gg.workspace_id = ? AND g.status != 'completed'`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocked := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		blocked[id] = true
	}
	return blocked, nil
}

// GetBacklogGoals retrieves goals that are not assigned to any sprint and belong to the workspace.
func GetBacklogGoals(workspaceID int64) ([]models.Goal, error) {
	d, err := getDefaultDB()
	if err != nil {
		return nil, err
	}
	return d.GetBacklogGoals(workspaceID)
}

// GetBacklogGoals retrieves goals that are not assigned to any sprint and belong to the workspace.
func (d *Database) GetBacklogGoals(workspaceID int64) ([]models.Goal, error) {
	rows, err := d.DB.Query(`
		SELECT id, parent_id, description, status, rank, priority, effort, tags, recurrence_rule, created_at, archived_at, task_started_at, task_elapsed_seconds, task_active
		FROM goals 
		WHERE sprint_id IS NULL AND status != 'completed' AND status != 'archived' AND workspace_id = ?
		ORDER BY rank ASC, created_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		var active int
		if err := rows.Scan(&g.ID, &g.ParentID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.Tags, &g.RecurrenceRule, &g.CreatedAt, &g.ArchivedAt, &g.TaskStartedAt, &g.TaskElapsedSec, &active); err != nil {
			return nil, err
		}
		g.TaskActive = active == 1
		goals = append(goals, g)
	}
	return goals, nil
}

// GetGoalsForSprint retrieves goals for a specific sprint ID.
func GetGoalsForSprint(sprintID int64) ([]models.Goal, error) {
	d, err := getDefaultDB()
	if err != nil {
		return nil, err
	}
	return d.GetGoalsForSprint(sprintID)
}

// GetGoalsForSprint retrieves goals for a specific sprint ID.
func (d *Database) GetGoalsForSprint(sprintID int64) ([]models.Goal, error) {
	rows, err := d.DB.Query(`
		SELECT id, parent_id, sprint_id, description, status, rank, priority, effort, tags, recurrence_rule, created_at, archived_at, task_started_at, task_elapsed_seconds, task_active
		FROM goals 
		WHERE sprint_id = ? AND status != 'archived' 
		ORDER BY rank ASC, created_at ASC`, sprintID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		var active int
		if err := rows.Scan(&g.ID, &g.ParentID, &g.SprintID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.Tags, &g.RecurrenceRule, &g.CreatedAt, &g.ArchivedAt, &g.TaskStartedAt, &g.TaskElapsedSec, &active); err != nil {
			return nil, err
		}
		g.TaskActive = active == 1
		goals = append(goals, g)
	}
	return goals, nil
}

// AddGoal inserts a new goal into the database.
func AddGoal(workspaceID int64, description string, sprintID int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.AddGoal(workspaceID, description, sprintID)
}

// AddGoal inserts a new goal into the database.
func (d *Database) AddGoal(workspaceID int64, description string, sprintID int64) error {
	var maxRank int
	var err error
	if sprintID > 0 {
		err = d.DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id = ?", sprintID).Scan(&maxRank)
	} else {
		err = d.DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id IS NULL AND workspace_id = ?", workspaceID).Scan(&maxRank)
	}
	if err != nil {
		return err
	}

	tags := util.TagsToJSON(util.ExtractTags(description))
	query := `INSERT INTO goals (workspace_id, description, sprint_id, status, rank, tags) VALUES (?, ?, ?, 'pending', ?, ?)`

	var sprintIDArg interface{}
	if sprintID > 0 {
		sprintIDArg = sprintID
	} else {
		sprintIDArg = nil // SQL NULL
	}

	_, err = d.DB.Exec(query, workspaceID, description, sprintIDArg, maxRank+1, tags)
	return err
}

func UpdateGoalPriority(goalID int64, priority int) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.UpdateGoalPriority(goalID, priority)
}

func (d *Database) UpdateGoalPriority(goalID int64, priority int) error {
	if priority < 1 {
		priority = 1
	}
	if priority > 5 {
		priority = 5
	}
	_, err := d.DB.Exec("UPDATE goals SET priority = ? WHERE id = ?", priority, goalID)
	return err
}

func GoalExists(workspaceID int64, sprintID int64, parentID *int64, description string) (bool, error) {
	d, err := getDefaultDB()
	if err != nil {
		return false, err
	}
	return d.GoalExists(workspaceID, sprintID, parentID, description)
}

func (d *Database) GoalExists(workspaceID int64, sprintID int64, parentID *int64, description string) (bool, error) {
	desc := strings.TrimSpace(description)
	if desc == "" {
		return false, nil
	}
	if parentID != nil {
		var count int
		err := d.DB.QueryRow(
			"SELECT COUNT(1) FROM goals WHERE parent_id = ? AND description = ?",
			*parentID, desc,
		).Scan(&count)
		return count > 0, err
	}
	if sprintID > 0 {
		var count int
		err := d.DB.QueryRow(
			"SELECT COUNT(1) FROM goals WHERE workspace_id = ? AND sprint_id = ? AND parent_id IS NULL AND description = ?",
			workspaceID, sprintID, desc,
		).Scan(&count)
		return count > 0, err
	}
	var count int
	err := d.DB.QueryRow(
		"SELECT COUNT(1) FROM goals WHERE workspace_id = ? AND sprint_id IS NULL AND parent_id IS NULL AND description = ?",
		workspaceID, desc,
	).Scan(&count)
	return count > 0, err
}

func GoalExistsDetailed(workspaceID int64, sprintID int64, parentID *int64, seed GoalSeed) (bool, error) {
	d, err := getDefaultDB()
	if err != nil {
		return false, err
	}
	return d.GoalExistsDetailed(workspaceID, sprintID, parentID, seed)
}

func (d *Database) GoalExistsDetailed(workspaceID int64, sprintID int64, parentID *int64, seed GoalSeed) (bool, error) {
	desc, priority, effort, tags, recurrence, notes, links := normalizeSeed(seed)
	if desc == "" {
		return false, nil
	}

	var rows *sql.Rows
	var err error
	if parentID != nil {
		rows, err = d.DB.Query(
			"SELECT description, priority, effort, tags, recurrence_rule, notes, links FROM goals WHERE parent_id = ? AND description = ?",
			*parentID, desc,
		)
	} else if sprintID > 0 {
		rows, err = d.DB.Query(
			"SELECT description, priority, effort, tags, recurrence_rule, notes, links FROM goals WHERE workspace_id = ? AND sprint_id = ? AND parent_id IS NULL AND description = ?",
			workspaceID, sprintID, desc,
		)
	} else {
		rows, err = d.DB.Query(
			"SELECT description, priority, effort, tags, recurrence_rule, notes, links FROM goals WHERE workspace_id = ? AND sprint_id IS NULL AND parent_id IS NULL AND description = ?",
			workspaceID, desc,
		)
	}
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var dbDesc string
		var dbPriority int
		var dbEffort, dbTags, dbRecurrence, dbNotes, dbLinks sql.NullString
		if err := rows.Scan(&dbDesc, &dbPriority, &dbEffort, &dbTags, &dbRecurrence, &dbNotes, &dbLinks); err != nil {
			return false, err
		}
		dbPriority = normalizePriority(dbPriority)
		dbEffortStr := normalizeEffort(dbEffort.String)
		dbTagsList := normalizeTags(dbTags.String)
		dbRecurrenceStr := strings.TrimSpace(dbRecurrence.String)
		dbNotesStr := strings.TrimSpace(dbNotes.String)
		dbLinksList := normalizeLinks(dbLinks.String)

		if dbPriority != priority {
			continue
		}
		if !strings.EqualFold(dbEffortStr, effort) {
			continue
		}
		if dbRecurrenceStr != recurrence {
			continue
		}
		if dbNotesStr != notes {
			continue
		}
		if !equalStringSlices(dbTagsList, tags) {
			continue
		}
		if !equalStringSlices(dbLinksList, links) {
			continue
		}
		return true, nil
	}
	return false, nil
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

func AddGoalDetailed(workspaceID int64, sprintID int64, seed GoalSeed) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.AddGoalDetailed(workspaceID, sprintID, seed)
}

func (d *Database) AddGoalDetailed(workspaceID int64, sprintID int64, seed GoalSeed) error {
	var maxRank int
	var err error
	if sprintID > 0 {
		err = d.DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id = ?", sprintID).Scan(&maxRank)
	} else {
		err = d.DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id IS NULL AND workspace_id = ?", workspaceID).Scan(&maxRank)
	}
	if err != nil {
		return err
	}

	priority := normalizePriority(seed.Priority)
	effort := normalizeEffort(seed.Effort)
	tags := seed.Tags
	if len(tags) == 0 {
		tags = util.ExtractTags(seed.Description)
	}
	tagsJSON := util.TagsToJSON(normalizeTagsFromSlice(tags))
	linksJSON, _ := json.Marshal(seed.Links)

	var sprintIDArg interface{}
	if sprintID > 0 {
		sprintIDArg = sprintID
	} else {
		sprintIDArg = nil
	}
	var notesArg interface{}
	if strings.TrimSpace(seed.Notes) != "" {
		notesArg = seed.Notes
	} else {
		notesArg = nil
	}
	var recurrenceArg interface{}
	if strings.TrimSpace(seed.Recurrence) != "" {
		recurrenceArg = seed.Recurrence
	} else {
		recurrenceArg = nil
	}

	_, err = d.DB.Exec(`INSERT INTO goals (workspace_id, description, sprint_id, status, rank, tags, priority, effort, notes, recurrence_rule, links)
		VALUES (?, ?, ?, 'pending', ?, ?, ?, ?, ?, ?, ?)`,
		workspaceID, seed.Description, sprintIDArg, maxRank+1, tagsJSON, priority, effort, notesArg, recurrenceArg, string(linksJSON))
	return err
}

// GetCompletedGoalsForDay retrieves all goals completed on a specific day and workspace across all sprints.
func GetCompletedGoalsForDay(dayID int64, workspaceID int64) ([]models.Goal, error) {
	d, err := getDefaultDB()
	if err != nil {
		return nil, err
	}
	return d.GetCompletedGoalsForDay(dayID, workspaceID)
}

// GetCompletedGoalsForDay retrieves all goals completed on a specific day and workspace across all sprints.
func (d *Database) GetCompletedGoalsForDay(dayID int64, workspaceID int64) ([]models.Goal, error) {
	dateStr := ""
	err := d.DB.QueryRow("SELECT date FROM days WHERE id = ?", dayID).Scan(&dateStr)
	if err != nil {
		return nil, err
	}

	rows, err := d.DB.Query(`
		SELECT id, parent_id, description, status, rank, priority, effort, tags, recurrence_rule, created_at, archived_at, task_started_at, task_elapsed_seconds, task_active
		FROM goals 
		WHERE status = 'completed' AND workspace_id = ?
		AND (
			sprint_id IN (SELECT id FROM sprints WHERE day_id = ?)
			OR (sprint_id IS NULL AND strftime('%Y-%m-%d', completed_at) = ?)
		)
		ORDER BY completed_at DESC`, workspaceID, dayID, dateStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		var active int
		if err := rows.Scan(&g.ID, &g.ParentID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.Tags, &g.RecurrenceRule, &g.CreatedAt, &g.ArchivedAt, &g.TaskStartedAt, &g.TaskElapsedSec, &active); err != nil {
			return nil, err
		}
		g.TaskActive = active == 1
		goals = append(goals, g)
	}
	return goals, nil
}

// AddSubtask inserts a new subtask linked to a parent goal.
func AddSubtask(description string, parentID int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.AddSubtask(description, parentID)
}

// AddSubtask inserts a new subtask linked to a parent goal.
func (d *Database) AddSubtask(description string, parentID int64) error {
	// Inherit sprint_id and workspace_id from parent
	var sprintID sql.NullInt64
	var workspaceID sql.NullInt64
	err := d.DB.QueryRow("SELECT sprint_id, workspace_id FROM goals WHERE id = ?", parentID).Scan(&sprintID, &workspaceID)
	if err != nil {
		return err
	}

	// Calculate rank among siblings
	var maxRank int
	err = d.DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE parent_id = ?", parentID).Scan(&maxRank)
	if err != nil {
		return err
	}

	tags := util.TagsToJSON(util.ExtractTags(description))
	_, err = d.DB.Exec(`INSERT INTO goals (description, parent_id, sprint_id, workspace_id, status, rank, tags) VALUES (?, ?, ?, ?, 'pending', ?, ?)`,
		description, parentID, sprintID, workspaceID, maxRank+1, tags)
	return err
}

func AddSubtaskDetailed(parentID int64, seed GoalSeed) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.AddSubtaskDetailed(parentID, seed)
}

func (d *Database) AddSubtaskDetailed(parentID int64, seed GoalSeed) error {
	var sprintID sql.NullInt64
	var workspaceID sql.NullInt64
	err := d.DB.QueryRow("SELECT sprint_id, workspace_id FROM goals WHERE id = ?", parentID).Scan(&sprintID, &workspaceID)
	if err != nil {
		return err
	}

	var maxRank int
	if err := d.DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE parent_id = ?", parentID).Scan(&maxRank); err != nil {
		return err
	}

	priority := normalizePriority(seed.Priority)
	effort := normalizeEffort(seed.Effort)
	tags := seed.Tags
	if len(tags) == 0 {
		tags = util.ExtractTags(seed.Description)
	}
	tagsJSON := util.TagsToJSON(normalizeTagsFromSlice(tags))
	linksJSON, _ := json.Marshal(seed.Links)

	var notesArg interface{}
	if strings.TrimSpace(seed.Notes) != "" {
		notesArg = seed.Notes
	} else {
		notesArg = nil
	}
	var recurrenceArg interface{}
	if strings.TrimSpace(seed.Recurrence) != "" {
		recurrenceArg = seed.Recurrence
	} else {
		recurrenceArg = nil
	}

	_, err = d.DB.Exec(`INSERT INTO goals (description, parent_id, sprint_id, workspace_id, status, rank, tags, priority, effort, notes, recurrence_rule, links)
		VALUES (?, ?, ?, ?, 'pending', ?, ?, ?, ?, ?, ?, ?)`,
		seed.Description, parentID, sprintID, workspaceID, maxRank+1, tagsJSON, priority, effort, notesArg, recurrenceArg, string(linksJSON))
	return err
}

// --- Task Management ---

func UpdateGoalStatus(goalID int64, status string) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.UpdateGoalStatus(goalID, status)
}

func (d *Database) UpdateGoalStatus(goalID int64, status string) error {
	if status == "completed" {
		var active int
		if err := d.DB.QueryRow("SELECT task_active FROM goals WHERE id = ?", goalID).Scan(&active); err != nil {
			return err
		}
		if active == 1 {
			if err := d.PauseTaskTimer(goalID); err != nil {
				return err
			}
		}
	}
	var err error
	if status == "completed" {
		_, err = d.DB.Exec("UPDATE goals SET status = ?, completed_at = CURRENT_TIMESTAMP WHERE id = ?", status, goalID)
	} else {
		_, err = d.DB.Exec("UPDATE goals SET status = ?, completed_at = NULL WHERE id = ?", status, goalID)
	}
	if err != nil {
		return err
	}
	if status == "completed" {
		return d.regenerateRecurringGoal(goalID)
	}
	return nil
}

func SwapGoalRanks(goalID1, goalID2 int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.SwapGoalRanks(goalID1, goalID2)
}

func (d *Database) SwapGoalRanks(goalID1, goalID2 int64) error {
	var rank1, rank2 int
	err := d.DB.QueryRow("SELECT rank FROM goals WHERE id = ?", goalID1).Scan(&rank1)
	if err != nil {
		return err
	}
	err = d.DB.QueryRow("SELECT rank FROM goals WHERE id = ?", goalID2).Scan(&rank2)
	if err != nil {
		return err
	}

	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE goals SET rank = ? WHERE id = ?", rank2, goalID1)
	if err != nil {
		return rollbackWithLog(tx, err)
	}

	_, err = tx.Exec("UPDATE goals SET rank = ? WHERE id = ?", rank1, goalID2)
	if err != nil {
		return rollbackWithLog(tx, err)
	}

	return tx.Commit()
}

func StartTaskTimer(goalID int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.StartTaskTimer(goalID)
}

func (d *Database) StartTaskTimer(goalID int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	var workspaceID sql.NullInt64
	err = tx.QueryRow("SELECT workspace_id FROM goals WHERE id = ?", goalID).Scan(&workspaceID)
	if err != nil {
		return rollbackWithLog(tx, err)
	}
	wsID := workspaceID.Int64

	rows, err := tx.Query(`SELECT id, task_started_at, task_elapsed_seconds FROM goals WHERE workspace_id = ? AND task_active = 1 AND id != ?`, wsID, goalID)
	if err != nil {
		return rollbackWithLog(tx, err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var started sql.NullTime
		var elapsed int
		if err := rows.Scan(&id, &started, &elapsed); err != nil {
			rows.Close()
			return rollbackWithLog(tx, err)
		}
		if started.Valid {
			elapsed += int(time.Since(started.Time).Seconds())
		}
		if _, err := tx.Exec("UPDATE goals SET task_active = 0, task_started_at = NULL, task_elapsed_seconds = ? WHERE id = ?", elapsed, id); err != nil {
			rows.Close()
			return rollbackWithLog(tx, err)
		}
	}
	if _, err := tx.Exec("UPDATE goals SET task_active = 1, task_started_at = CURRENT_TIMESTAMP WHERE id = ?", goalID); err != nil {
		return rollbackWithLog(tx, err)
	}
	return tx.Commit()
}

func PauseTaskTimer(goalID int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.PauseTaskTimer(goalID)
}

func (d *Database) PauseTaskTimer(goalID int64) error {
	var started sql.NullTime
	var elapsed int
	var active int
	if err := d.DB.QueryRow("SELECT task_active, task_started_at, task_elapsed_seconds FROM goals WHERE id = ?", goalID).Scan(&active, &started, &elapsed); err != nil {
		return err
	}
	if active == 0 {
		return nil
	}
	if started.Valid {
		elapsed += int(time.Since(started.Time).Seconds())
	}
	_, err := d.DB.Exec("UPDATE goals SET task_active = 0, task_started_at = NULL, task_elapsed_seconds = ? WHERE id = ?", elapsed, goalID)
	return err
}

func GetActiveTask(workspaceID int64) (*models.Goal, error) {
	d, err := getDefaultDB()
	if err != nil {
		return nil, err
	}
	return d.GetActiveTask(workspaceID)
}

func (d *Database) GetActiveTask(workspaceID int64) (*models.Goal, error) {
	row := d.DB.QueryRow(`
		SELECT id, parent_id, sprint_id, description, status, rank, priority, effort, tags, recurrence_rule, created_at, archived_at, task_started_at, task_elapsed_seconds, task_active
		FROM goals WHERE workspace_id = ? AND task_active = 1 LIMIT 1`, workspaceID)
	var g models.Goal
	var active int
	if err := row.Scan(&g.ID, &g.ParentID, &g.SprintID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.Tags, &g.RecurrenceRule, &g.CreatedAt, &g.ArchivedAt, &g.TaskStartedAt, &g.TaskElapsedSec, &active); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	g.TaskActive = active == 1
	return &g, nil
}

func MoveGoal(goalID int64, targetSprintID int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.MoveGoal(goalID, targetSprintID)
}

func (d *Database) MoveGoal(goalID int64, targetSprintID int64) error {
	var sprintArg interface{}
	if targetSprintID == 0 {
		sprintArg = nil // SQL NULL for Backlog
	} else {
		sprintArg = targetSprintID
	}

	_, err := d.DB.Exec("UPDATE goals SET sprint_id = ? WHERE id = ?", sprintArg, goalID)
	return err
}

func EditGoal(goalID int64, newDescription string) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.EditGoal(goalID, newDescription)
}

func (d *Database) EditGoal(goalID int64, newDescription string) error {
	tags := util.TagsToJSON(util.ExtractTags(newDescription))
	_, err := d.DB.Exec("UPDATE goals SET description = ?, tags = ? WHERE id = ?", newDescription, tags, goalID)
	return err
}

func UpdateGoalRecurrence(goalID int64, rule string) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.UpdateGoalRecurrence(goalID, rule)
}

func (d *Database) UpdateGoalRecurrence(goalID int64, rule string) error {
	var value interface{} = nil
	if strings.TrimSpace(rule) != "" {
		value = rule
	}
	_, err := d.DB.Exec("UPDATE goals SET recurrence_rule = ? WHERE id = ?", value, goalID)
	return err
}

func DeleteGoal(goalID int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.DeleteGoal(goalID)
}

func (d *Database) DeleteGoal(goalID int64) error {
	_, err := d.DB.Exec("DELETE FROM goals WHERE id = ?", goalID)
	return err
}

func AddTagsToGoal(goalID int64, tagsToAdd []string) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.AddTagsToGoal(goalID, tagsToAdd)
}

func (d *Database) AddTagsToGoal(goalID int64, tagsToAdd []string) error {
	if len(tagsToAdd) == 0 {
		return nil
	}

	// Fetch existing tags
	var existingTags sql.NullString
	if err := d.DB.QueryRow("SELECT tags FROM goals WHERE id = ?", goalID).Scan(&existingTags); err != nil {
		return err
	}

	var tags []string
	if existingTags.Valid {
		tags = util.JSONToTags(existingTags.String)
	}
	tags = append(tags, tagsToAdd...)
	tags = normalizeTagsFromSlice(tags)
	tagsJSON := util.TagsToJSON(tags)

	_, err := d.DB.Exec("UPDATE goals SET tags = ? WHERE id = ?", tagsJSON, goalID)
	return err
}

func SetGoalTags(goalID int64, tags []string) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.SetGoalTags(goalID, tags)
}

func (d *Database) SetGoalTags(goalID int64, tags []string) error {
	tagsJSON := util.TagsToJSON(normalizeTagsFromSlice(tags))
	_, err := d.DB.Exec("UPDATE goals SET tags = ? WHERE id = ?", tagsJSON, goalID)
	return err
}

func Search(query util.SearchQuery, workspaceID int64) ([]models.Goal, error) {
	d, err := getDefaultDB()
	if err != nil {
		return nil, err
	}
	return d.Search(query, workspaceID)
}

func (d *Database) Search(query util.SearchQuery, workspaceID int64) ([]models.Goal, error) {
	var args []interface{}
	sql := `
		SELECT id, parent_id, sprint_id, description, status, rank, priority, effort, tags, recurrence_rule, created_at, archived_at, task_started_at, task_elapsed_seconds, task_active
		FROM goals
		WHERE workspace_id = ?
	`
	args = append(args, workspaceID)

	if len(query.Status) > 0 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(query.Status)), ",")
		sql += " AND status IN (" + placeholders + ")"
		for _, s := range query.Status {
			args = append(args, s)
		}
	}

	if len(query.Tags) > 0 {
		for _, t := range query.Tags {
			sql += " AND tags LIKE ?"
			args = append(args, "%"+t+"%")
		}
	}

	if len(query.Text) > 0 {
		for _, term := range query.Text {
			if strings.TrimSpace(term) == "" {
				continue
			}
			sql += " AND description LIKE ?"
			args = append(args, "%"+term+"%")
		}
	}

	sql += " ORDER BY created_at DESC LIMIT 50"

	return d.scanGoals(sql, args...)
}

func (d *Database) scanGoals(query string, args ...interface{}) ([]models.Goal, error) {
	rows, err := d.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		var active int
		if err := rows.Scan(&g.ID, &g.ParentID, &g.SprintID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.Tags, &g.RecurrenceRule, &g.CreatedAt, &g.ArchivedAt, &g.TaskStartedAt, &g.TaskElapsedSec, &active); err != nil {
			return nil, err
		}
		g.TaskActive = active == 1
		goals = append(goals, g)
	}
	return goals, nil
}

func GetAllGoals() ([]models.Goal, error) {
	d, err := getDefaultDB()
	if err != nil {
		return nil, err
	}
	return d.GetAllGoals()
}

func (d *Database) GetAllGoals() ([]models.Goal, error) {
	return d.scanGoals(`
		SELECT id, parent_id, sprint_id, description, status, rank, priority, effort, tags, recurrence_rule, created_at, archived_at, task_started_at, task_elapsed_seconds, task_active
		FROM goals 
		ORDER BY rank ASC, created_at ASC`)
}

// Archived goals
func GetArchivedGoals(workspaceID int64) ([]models.Goal, error) {
	d, err := getDefaultDB()
	if err != nil {
		return nil, err
	}
	return d.GetArchivedGoals(workspaceID)
}

// Archived goals
func (d *Database) GetArchivedGoals(workspaceID int64) ([]models.Goal, error) {
	rows, err := d.DB.Query(`
		SELECT id, parent_id, sprint_id, description, status, rank, priority, effort, tags, recurrence_rule, created_at, archived_at, task_started_at, task_elapsed_seconds, task_active
		FROM goals 
		WHERE status = 'archived' AND workspace_id = ?
		ORDER BY archived_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		var active int
		if err := rows.Scan(&g.ID, &g.ParentID, &g.SprintID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.Tags, &g.RecurrenceRule, &g.CreatedAt, &g.ArchivedAt, &g.TaskStartedAt, &g.TaskElapsedSec, &active); err != nil {
			return nil, err
		}
		g.TaskActive = active == 1
		goals = append(goals, g)
	}
	return goals, nil
}

func ArchiveGoal(goalID int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.ArchiveGoal(goalID)
}

func (d *Database) ArchiveGoal(goalID int64) error {
	_, err := d.DB.Exec("UPDATE goals SET status = 'archived', archived_at = CURRENT_TIMESTAMP WHERE id = ?", goalID)
	return err
}

func UnarchiveGoal(goalID int64) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.UnarchiveGoal(goalID)
}

func (d *Database) UnarchiveGoal(goalID int64) error {
	_, err := d.DB.Exec("UPDATE goals SET status = 'pending', archived_at = NULL, sprint_id = NULL WHERE id = ?", goalID)
	return err
}
