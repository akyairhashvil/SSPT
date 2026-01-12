package tui

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/util"
)

type seedConfig struct {
	Backlog []database.GoalSeed `json:"backlog"`
	Sprints []struct {
		Number int                 `json:"number"`
		Tasks  []database.GoalSeed `json:"tasks"`
	} `json:"sprints"`
}

func EnsureSeedFile() (string, error) {
	configDir := util.ConfigDir("sspt")
	jsonPath := filepath.Join(configDir, "seed.json")
	txtPath := util.SeedPath("sspt")
	if _, err := os.Stat(jsonPath); err == nil {
		return jsonPath, nil
	}
	if _, err := os.Stat(txtPath); err == nil {
		return txtPath, nil
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", err
	}
	skeleton := []string{
		"# SSPT Seed (DSL)",
		"# = Workspace",
		"# + Sprint number",
		"# * Task",
		"# - Subtask",
		"# Tags: #tag  Priority: !1..5  Effort: @S|@M|@L  Recurrence: ~daily|~weekly:mon,tue",
		"",
		"= Personal",
		"+ 1",
		"* Ship onboarding flow #focus @L !2",
		"- Write checklist",
		"* Draft project brief #docs !3",
		"+ 2",
		"* Review backlog #review",
		"",
		"* Unassigned backlog item #later",
	}
	if err := os.WriteFile(txtPath, []byte(strings.Join(skeleton, "\n")), 0o644); err != nil {
		return "", err
	}
	return txtPath, nil
}

func ImportSeed(path string, workspaceID int64, dayID int64) (int, string, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, "", 0, err
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])

	if isJSONSeed(data) {
		imported, backlogFallback, err := importSeedJSON(data, workspaceID, dayID)
		if err != nil {
			return imported, "", backlogFallback, err
		}
		return imported, hash, backlogFallback, nil
	}

	imported, backlogFallback, err := importSeedDSL(data, workspaceID, dayID)
	if err != nil {
		return imported, "", backlogFallback, err
	}
	return imported, hash, backlogFallback, nil
}

func isJSONSeed(data []byte) bool {
	for _, b := range data {
		if b == ' ' || b == '\n' || b == '\t' || b == '\r' {
			continue
		}
		return b == '{'
	}
	return false
}

func importSeedJSON(data []byte, workspaceID int64, dayID int64) (int, int, error) {
	var cfg seedConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return 0, 0, fmt.Errorf("invalid seed file: %w", err)
	}
	sprints, err := database.GetSprints(dayID, workspaceID)
	if err != nil {
		return 0, 0, err
	}
	sprintIDs := make(map[int]int64)
	for _, s := range sprints {
		sprintIDs[s.SprintNumber] = s.ID
	}
	imported := 0
	backlogFallback := 0
	for _, task := range cfg.Backlog {
		if strings.TrimSpace(task.Description) == "" {
			continue
		}
		exists, err := database.GoalExistsDetailed(workspaceID, 0, nil, task)
		if err != nil {
			return imported, backlogFallback, err
		}
		if !exists {
			if err := database.AddGoalDetailed(workspaceID, 0, task); err != nil {
				return imported, backlogFallback, err
			}
			imported++
		}
	}
	for _, sprint := range cfg.Sprints {
		targetID, ok := sprintIDs[sprint.Number]
		if !ok {
			for {
				if err := database.AppendSprint(dayID, workspaceID); err != nil {
					if isMaxSprintErr(err) {
						targetID = 0
						backlogFallback++
						break
					}
					return imported, backlogFallback, err
				}
				sprints, err := database.GetSprints(dayID, workspaceID)
				if err != nil {
					return imported, backlogFallback, err
				}
				for _, s := range sprints {
					sprintIDs[s.SprintNumber] = s.ID
				}
				if id, ok := sprintIDs[sprint.Number]; ok {
					targetID = id
					break
				}
			}
		}
		for _, task := range sprint.Tasks {
			if strings.TrimSpace(task.Description) == "" {
				continue
			}
			exists, err := database.GoalExistsDetailed(workspaceID, targetID, nil, task)
			if err != nil {
				return imported, backlogFallback, err
			}
			if !exists {
				if err := database.AddGoalDetailed(workspaceID, targetID, task); err != nil {
					return imported, backlogFallback, err
				}
				imported++
			}
		}
	}
	return imported, backlogFallback, nil
}

func importSeedDSL(data []byte, defaultWorkspaceID int64, dayID int64) (int, int, error) {
	sprintsByWorkspace := make(map[int64]map[int]int64)
	backlogFallback := 0
	getSprintID := func(workspaceID int64, number int) (int64, bool, error) {
		if sprintsByWorkspace[workspaceID] == nil {
			sprints, err := database.GetSprints(dayID, workspaceID)
			if err != nil {
				return 0, false, err
			}
			sprintsByWorkspace[workspaceID] = make(map[int]int64)
			for _, s := range sprints {
				sprintsByWorkspace[workspaceID][s.SprintNumber] = s.ID
			}
		}
		if id, ok := sprintsByWorkspace[workspaceID][number]; ok {
			return id, true, nil
		}
		for {
			if err := database.AppendSprint(dayID, workspaceID); err != nil {
				if isMaxSprintErr(err) {
					backlogFallback++
					return 0, true, nil
				}
				return 0, false, err
			}
			sprints, err := database.GetSprints(dayID, workspaceID)
			if err != nil {
				return 0, false, err
			}
			for _, s := range sprints {
				sprintsByWorkspace[workspaceID][s.SprintNumber] = s.ID
			}
			if id, ok := sprintsByWorkspace[workspaceID][number]; ok {
				return id, true, nil
			}
		}
	}

	currentWorkspaceID := defaultWorkspaceID
	currentSprint := 0
	var lastGoalID int64
	imported := 0

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch line[0] {
		case '=':
			name := strings.TrimSpace(strings.TrimPrefix(line, "="))
			if name == "" {
				return imported, backlogFallback, fmt.Errorf("workspace name required")
			}
			wsID, err := ensureWorkspaceByName(name)
			if err != nil {
				return imported, backlogFallback, err
			}
			currentWorkspaceID = wsID
			currentSprint = 0
			lastGoalID = 0
		case '+':
			num, ok := parseSprintNumber(line)
			if !ok {
				return imported, backlogFallback, fmt.Errorf("invalid sprint line: %q", line)
			}
			currentSprint = num
			lastGoalID = 0
		case '*':
			task, err := parseSeedTask(strings.TrimSpace(strings.TrimPrefix(line, "*")))
			if err != nil {
				return imported, backlogFallback, err
			}
			if task.Description == "" {
				continue
			}
			targetID := int64(0)
			if currentSprint > 0 {
				id, _, err := getSprintID(currentWorkspaceID, currentSprint)
				if err != nil {
					return imported, backlogFallback, err
				}
				targetID = id
			}
			exists, err := database.GoalExistsDetailed(currentWorkspaceID, targetID, nil, task)
			if err != nil {
				return imported, backlogFallback, err
			}
			if !exists {
				if err := database.AddGoalDetailed(currentWorkspaceID, targetID, task); err != nil {
					return imported, backlogFallback, err
				}
				imported++
			}
			lastGoalID = getLastGoalID()
		case '-':
			task, err := parseSeedTask(strings.TrimSpace(strings.TrimPrefix(line, "-")))
			if err != nil {
				return imported, backlogFallback, err
			}
			if task.Description == "" {
				continue
			}
			if lastGoalID == 0 {
				return imported, backlogFallback, fmt.Errorf("subtask without parent: %q", line)
			}
			exists, err := database.GoalExistsDetailed(currentWorkspaceID, 0, &lastGoalID, task)
			if err != nil {
				return imported, backlogFallback, err
			}
			if !exists {
				if err := database.AddSubtaskDetailed(lastGoalID, task); err != nil {
					return imported, backlogFallback, err
				}
				imported++
			}
		default:
			return imported, backlogFallback, fmt.Errorf("unknown seed line: %q", line)
		}
	}
	if err := scanner.Err(); err != nil {
		return imported, backlogFallback, err
	}
	return imported, backlogFallback, nil
}

func parseSeedTask(line string) (database.GoalSeed, error) {
	var seed database.GoalSeed
	if strings.TrimSpace(line) == "" {
		return seed, nil
	}
	parts := strings.Fields(line)
	var desc []string
	for _, part := range parts {
		switch {
		case strings.HasPrefix(part, "#"):
			tag := strings.TrimPrefix(part, "#")
			if tag != "" {
				seed.Tags = append(seed.Tags, strings.ToLower(tag))
			}
		case strings.HasPrefix(part, "!"):
			val := strings.TrimPrefix(part, "!")
			if val != "" {
				if p, err := strconv.Atoi(val); err == nil {
					seed.Priority = p
				}
			}
		case strings.HasPrefix(part, "@"):
			seed.Effort = strings.ToUpper(strings.TrimPrefix(part, "@"))
		case strings.HasPrefix(part, "~"):
			seed.Recurrence = strings.TrimPrefix(part, "~")
		default:
			desc = append(desc, part)
		}
	}
	seed.Description = strings.Join(desc, " ")
	if seed.Priority < 0 {
		seed.Priority = 0
	}
	return seed, nil
}

func parseSprintNumber(line string) (int, bool) {
	for _, part := range strings.Fields(line) {
		part = strings.TrimSpace(part)
		part = strings.TrimPrefix(strings.ToLower(part), "sprint")
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if num, err := strconv.Atoi(part); err == nil {
			return num, true
		}
	}
	return 0, false
}

func ensureWorkspaceByName(name string) (int64, error) {
	slug := slugify(name)
	if id, ok, err := database.GetWorkspaceIDBySlug(slug); err != nil {
		return 0, err
	} else if ok {
		return id, nil
	}
	return database.CreateWorkspace(name, slug)
}

func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

func getLastGoalID() int64 {
	var id int64
	if err := database.DB.QueryRow("SELECT id FROM goals ORDER BY id DESC LIMIT 1").Scan(&id); err != nil {
		return 0
	}
	return id
}

func isMaxSprintErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "max sprints")
}
