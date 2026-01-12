# AGENTS.md - Refactoring & Improvement Plans for SSPT

This document provides detailed, actionable plans for AI coding agents (and human developers) to systematically improve the SSPT codebase. Each section contains specific tasks with file locations, code examples, and implementation guidance.

---

## Table of Contents

1. [Priority Overview](#i-priority-overview)
2. [Phase 1: Foundation (Critical)](#ii-phase-1-foundation-critical)
3. [Phase 2: Architecture (High Priority)](#iii-phase-2-architecture-high-priority)
4. [Phase 3: Quality (Medium Priority)](#iv-phase-3-quality-medium-priority)
5. [Phase 4: Polish (Lower Priority)](#v-phase-4-polish-lower-priority)
6. [Code Patterns & Standards](#vi-code-patterns--standards)
7. [Testing Strategy](#vii-testing-strategy)

---

## I. Priority Overview

| Phase | Focus Area | Impact | Effort |
|-------|------------|--------|--------|
| 1 | Foundation | Enables all other improvements | Medium |
| 2 | Architecture | Maintainability, testability | High |
| 3 | Quality | Code cleanliness, consistency | Medium |
| 4 | Polish | Performance, organization | Low |

### Dependency Graph

```
Phase 1: Global DB Singleton Removal
    │
    ├──► Phase 2: TUI Decomposition
    │        │
    │        └──► Phase 3: Comprehensive Testing
    │
    └──► Phase 2: Dependency Injection
             │
             └──► Phase 3: Query Helpers
```

---

## II. Phase 1: Foundation (Critical)

These changes unblock all subsequent improvements and should be completed first.

### Task 1.1: Eliminate Global Database Singleton

**Problem:** Global variables `DefaultDB` and `DB` in `internal/database/db.go:23-24` prevent testability and require 78+ wrapper functions.

**Current Pattern (Bad):**
```go
// internal/database/db.go:23-24
var DefaultDB *Database
var DB *sql.DB

// Every function repeats this pattern:
func AddGoal(workspaceID int64, description string, sprintID int64) error {
    d, err := getDefaultDB()
    if err != nil { return err }
    return d.AddGoal(workspaceID, description, sprintID)
}
```

**Target Pattern (Good):**
```go
// Remove globals entirely. Database instance passed via dependency injection.
// Only receiver methods remain:
func (d *Database) AddGoal(workspaceID int64, description string, sprintID int64) error {
    // Implementation
}
```

**Implementation Steps:**

1. **Update `cmd/app/main.go`** to pass `*Database` to TUI initialization:
   ```go
   // Before
   m := tui.InitialModel()

   // After
   db, err := database.Open(dbPath, passphrase)
   if err != nil { log.Fatal(err) }
   defer db.Close()
   m := tui.InitialModel(db)
   ```

2. **Modify `internal/tui/dashboard.go`** - Add `db *database.Database` field to `DashboardModel`:
   ```go
   type DashboardModel struct {
       db *database.Database  // Add this field
       // ... existing fields
   }

   func InitialModel(db *database.Database) DashboardModel {
       return DashboardModel{
           db: db,
           // ... existing initialization
       }
   }
   ```

3. **Update all TUI database calls** - Replace global function calls with receiver calls:
   ```go
   // Before (scattered throughout dashboard_update.go)
   goals, err := database.GetGoalsForSprint(sprintID)

   // After
   goals, err := m.db.GetGoalsForSprint(sprintID)
   ```

4. **Remove wrapper functions** from these files:
   - `internal/database/goal.go` - Remove ~32 wrapper functions
   - `internal/database/sprint.go` - Remove ~13 wrapper functions
   - `internal/database/workspace.go` - Remove ~7 wrapper functions
   - `internal/database/journal.go` - Remove wrapper functions
   - `internal/database/settings.go` - Remove wrapper functions

5. **Remove global variables** from `internal/database/db.go`:
   - Delete lines 23-24 (`var DefaultDB`, `var DB`)
   - Delete `getDefaultDB()` function
   - Delete `SetDefaultDB()` function

**Files to Modify:**
- `cmd/app/main.go`
- `internal/database/db.go`
- `internal/database/goal.go`
- `internal/database/sprint.go`
- `internal/database/workspace.go`
- `internal/database/journal.go`
- `internal/database/settings.go`
- `internal/tui/dashboard.go`
- `internal/tui/dashboard_update.go`
- `internal/tui/model.go`

**Verification:**
```bash
# Should find no references to DefaultDB or getDefaultDB
grep -r "DefaultDB\|getDefaultDB" internal/
```

---

### Task 1.2: Fix Silent Error Handling

**Problem:** 380+ instances of errors being silently ignored with `_ = ` or `_, _ =` patterns.

**Locations to Audit:**

| File | Approx Count | Priority |
|------|--------------|----------|
| `internal/tui/dashboard_update.go` | ~50 | High |
| `internal/tui/dashboard_render.go` | ~30 | Medium |
| `internal/tui/model.go` | ~10 | High |
| `internal/database/goal.go` | ~15 | High |
| `internal/database/db.go` | ~20 | Medium |

**Critical Instances to Fix:**

1. **`internal/tui/model.go:121`**
   ```go
   // Before (silent failure)
   wsID, _ := database.EnsureDefaultWorkspace()

   // After (handle error)
   wsID, err := m.db.EnsureDefaultWorkspace()
   if err != nil {
       // Log error, set status message, or return error
       log.Printf("Failed to ensure default workspace: %v", err)
       // Provide fallback or show user error
   }
   ```

2. **`internal/database/goal.go:132`**
   ```go
   // Before (silent max rank lookup failure)
   _ = d.DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals...").Scan(&maxRank)

   // After
   if err := d.DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals...").Scan(&maxRank); err != nil {
       log.Printf("Failed to get max rank, defaulting to 0: %v", err)
       maxRank = 0
   }
   ```

3. **`internal/tui/dashboard_update.go:77`**
   ```go
   // Before
   newProg, _ := m.progress.Update(msg)

   // After (progress update errors are typically safe to ignore, but log)
   newProg, cmd := m.progress.Update(msg)
   // cmd can be safely ignored for progress bar
   ```

**Implementation Strategy:**

For each `_ = ` instance, categorize and handle:

| Category | Action | Example |
|----------|--------|---------|
| Critical DB operations | Return/propagate error | `AddGoal`, `UpdateGoal` |
| UI state updates | Log + continue | Progress bar updates |
| Optional features | Log + fallback | Cache operations |
| Cleanup operations | Log only | File deletions in defer |

**Add Logging Helper:**
```go
// internal/util/logging.go (new file)
package util

import "log"

// LogError logs an error if non-nil, with context
func LogError(context string, err error) {
    if err != nil {
        log.Printf("%s: %v", context, err)
    }
}

// MustSucceed logs and panics on error (use sparingly)
func MustSucceed(context string, err error) {
    if err != nil {
        log.Fatalf("%s: %v", context, err)
    }
}
```

**Verification:**
```bash
# Count remaining silent errors (should decrease significantly)
grep -r "_ = \|_, _ =" internal/ | wc -l

# Review remaining instances
grep -rn "_ = \|_, _ =" internal/
```

---

### Task 1.3: Add Structured Error Types

**Problem:** Error messages are inconsistent, formatted with `fmt.Sprintf` throughout.

**Current Pattern:**
```go
m.setStatusError(fmt.Sprintf("Error completing sprint: %v", err))
m.setStatusError(fmt.Sprintf("Error moving pending tasks: %v", err))
```

**Target Pattern:**
```go
// internal/database/errors.go (new file)
package database

import "fmt"

type GoalError struct {
    Op  string // Operation: "add", "update", "delete", "query"
    ID  int64  // Goal ID if applicable
    Err error  // Underlying error
}

func (e *GoalError) Error() string {
    if e.ID > 0 {
        return fmt.Sprintf("goal %s (id=%d): %v", e.Op, e.ID, e.Err)
    }
    return fmt.Sprintf("goal %s: %v", e.Op, e.Err)
}

func (e *GoalError) Unwrap() error { return e.Err }

type SprintError struct {
    Op  string
    ID  int64
    Err error
}

func (e *SprintError) Error() string {
    if e.ID > 0 {
        return fmt.Sprintf("sprint %s (id=%d): %v", e.Op, e.ID, e.Err)
    }
    return fmt.Sprintf("sprint %s: %v", e.Op, e.Err)
}

func (e *SprintError) Unwrap() error { return e.Err }

type WorkspaceError struct {
    Op  string
    ID  int64
    Err error
}

func (e *WorkspaceError) Error() string {
    return fmt.Sprintf("workspace %s: %v", e.Op, e.Err)
}

func (e *WorkspaceError) Unwrap() error { return e.Err }
```

**Usage in Database Layer:**
```go
func (d *Database) AddGoal(workspaceID int64, description string, sprintID int64) error {
    _, err := d.DB.Exec(...)
    if err != nil {
        return &GoalError{Op: "add", Err: err}
    }
    return nil
}
```

---

## III. Phase 2: Architecture (High Priority)

### Task 2.1: Decompose DashboardModel

**Problem:** `DashboardModel` in `internal/tui/dashboard.go:37-119` has 107+ fields managing unrelated concerns.

**Current Structure:**
```go
type DashboardModel struct {
    // Timer state (6 fields)
    activeSprint, activeTask, breakActive, breakStart...

    // Lock state (10 fields)
    locked, passphraseHash, passphraseInput, lockTimer...

    // Search state (5 fields)
    searching, searchQuery, searchInput, searchResults...

    // Input modes (12+ fields)
    creatingGoal, editingGoal, movingGoal, confirmingDelete...

    // ... 70+ more fields
}
```

**Target Structure - Extract Sub-Models:**

1. **Create `internal/tui/timer_model.go`:**
   ```go
   package tui

   import "time"
   import "github.com/your/project/internal/models"

   type TimerModel struct {
       ActiveSprint  *models.Sprint
       ActiveTask    *models.Goal
       BreakActive   bool
       BreakStart    time.Time
       SprintStart   time.Time
       Paused        bool
       PausedElapsed time.Duration
   }

   func NewTimerModel() TimerModel {
       return TimerModel{}
   }

   func (t *TimerModel) StartSprint(sprint *models.Sprint) {
       t.ActiveSprint = sprint
       t.SprintStart = time.Now()
       t.Paused = false
   }

   func (t *TimerModel) PauseSprint() {
       if !t.Paused {
           t.Paused = true
           t.PausedElapsed = time.Since(t.SprintStart)
       }
   }

   func (t *TimerModel) ResumeSprint() {
       if t.Paused {
           t.Paused = false
           t.SprintStart = time.Now().Add(-t.PausedElapsed)
       }
   }

   func (t *TimerModel) ElapsedTime() time.Duration {
       if t.Paused {
           return t.PausedElapsed
       }
       if t.ActiveSprint != nil {
           return time.Since(t.SprintStart)
       }
       return 0
   }

   func (t *TimerModel) Reset() {
       t.ActiveSprint = nil
       t.ActiveTask = nil
       t.BreakActive = false
       t.Paused = false
       t.PausedElapsed = 0
   }
   ```

2. **Create `internal/tui/lock_model.go`:**
   ```go
   package tui

   import (
       "time"
       "github.com/charmbracelet/bubbles/textinput"
   )

   type LockModel struct {
       Locked          bool
       PassphraseHash  string
       PassphraseInput textinput.Model
       LockTimer       time.Time
       AutoLockAfter   time.Duration
       Attempts        int
       MaxAttempts     int
   }

   func NewLockModel(autoLockMinutes int) LockModel {
       input := textinput.New()
       input.EchoMode = textinput.EchoPassword
       input.Placeholder = "Enter passphrase..."

       return LockModel{
           PassphraseInput: input,
           AutoLockAfter:   time.Duration(autoLockMinutes) * time.Minute,
           MaxAttempts:     5,
       }
   }

   func (l *LockModel) Lock() {
       l.Locked = true
       l.Attempts = 0
       l.PassphraseInput.Reset()
   }

   func (l *LockModel) TryUnlock(enteredHash string) bool {
       if enteredHash == l.PassphraseHash {
           l.Locked = false
           l.Attempts = 0
           l.ResetTimer()
           return true
       }
       l.Attempts++
       return false
   }

   func (l *LockModel) ResetTimer() {
       l.LockTimer = time.Now()
   }

   func (l *LockModel) ShouldAutoLock() bool {
       if l.AutoLockAfter == 0 || l.Locked {
           return false
       }
       return time.Since(l.LockTimer) > l.AutoLockAfter
   }
   ```

3. **Create `internal/tui/search_model.go`:**
   ```go
   package tui

   import (
       "github.com/charmbracelet/bubbles/textinput"
       "github.com/your/project/internal/models"
   )

   type SearchModel struct {
       Active       bool
       Input        textinput.Model
       Query        string
       Results      []models.Goal
       SelectedIdx  int
       SearchColumn int // Which column to search
   }

   func NewSearchModel() SearchModel {
       input := textinput.New()
       input.Placeholder = "Search..."
       input.CharLimit = 100

       return SearchModel{
           Input: input,
       }
   }

   func (s *SearchModel) Start() {
       s.Active = true
       s.Input.Focus()
       s.Query = ""
       s.Results = nil
       s.SelectedIdx = 0
   }

   func (s *SearchModel) Cancel() {
       s.Active = false
       s.Input.Blur()
       s.Input.Reset()
   }

   func (s *SearchModel) Submit() string {
       s.Query = s.Input.Value()
       s.Active = false
       s.Input.Blur()
       return s.Query
   }

   func (s *SearchModel) SetResults(results []models.Goal) {
       s.Results = results
       s.SelectedIdx = 0
   }
   ```

4. **Update `DashboardModel`** to use sub-models:
   ```go
   type DashboardModel struct {
       db     *database.Database
       timer  TimerModel
       lock   LockModel
       search SearchModel

       // Remaining fields that don't fit elsewhere
       currentWorkspace *models.Workspace
       workspaces       []models.Workspace
       // ... UI state fields
   }
   ```

**Migration Steps:**

1. Create new files with sub-models
2. Add sub-model fields to `DashboardModel`
3. Update `InitialModel()` to initialize sub-models
4. Migrate field access one sub-model at a time:
   - Find all references to e.g., `m.locked`
   - Replace with `m.lock.Locked`
5. Move related methods to sub-model receivers
6. Remove old fields from `DashboardModel`

---

### Task 2.2: Split Update Method

**Problem:** `Update()` in `internal/tui/dashboard_update.go` is 1491 lines with deeply nested switch statements.

**Current Structure:**
```go
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // 1491 lines of nested switches
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle all key events
        if m.locked { /* 100+ lines */ }
        if m.searching { /* 100+ lines */ }
        if m.creatingGoal { /* 100+ lines */ }
        // ... many more modes
    case tickMsg:
        // Timer handling
    // ... more message types
    }
}
```

**Target Structure - Extract Handlers:**

1. **Create `internal/tui/update_handlers.go`:**
   ```go
   package tui

   import tea "github.com/charmbracelet/bubbletea"

   // handleLockedState handles input when the app is locked
   func (m DashboardModel) handleLockedState(msg tea.KeyMsg) (DashboardModel, tea.Cmd) {
       switch msg.String() {
       case "enter":
           entered := m.lock.PassphraseInput.Value()
           hash := util.HashPassphrase(entered)
           if m.lock.TryUnlock(hash) {
               return m, nil
           }
           m.setStatusError("Invalid passphrase")
           return m, nil
       case "esc":
           // Handle escape
       }

       var cmd tea.Cmd
       m.lock.PassphraseInput, cmd = m.lock.PassphraseInput.Update(msg)
       return m, cmd
   }

   // handleSearching handles input during search mode
   func (m DashboardModel) handleSearching(msg tea.KeyMsg) (DashboardModel, tea.Cmd) {
       switch msg.String() {
       case "enter":
           query := m.search.Submit()
           m.performSearch(query)
           return m, nil
       case "esc":
           m.search.Cancel()
           return m, nil
       case "up", "ctrl+p":
           m.search.SelectedIdx = max(0, m.search.SelectedIdx-1)
           return m, nil
       case "down", "ctrl+n":
           m.search.SelectedIdx = min(len(m.search.Results)-1, m.search.SelectedIdx+1)
           return m, nil
       }

       var cmd tea.Cmd
       m.search.Input, cmd = m.search.Input.Update(msg)
       return m, cmd
   }

   // handleCreatingGoal handles input when creating a new goal
   func (m DashboardModel) handleCreatingGoal(msg tea.KeyMsg) (DashboardModel, tea.Cmd) {
       switch msg.String() {
       case "enter":
           return m.submitNewGoal()
       case "esc":
           return m.cancelGoalCreation()
       case "tab":
           return m.cycleGoalInputField()
       }

       return m.updateGoalInput(msg)
   }

   // handleEditingGoal handles input when editing an existing goal
   func (m DashboardModel) handleEditingGoal(msg tea.KeyMsg) (DashboardModel, tea.Cmd) {
       // Similar to creating
   }

   // handleConfirmation handles yes/no confirmation dialogs
   func (m DashboardModel) handleConfirmation(msg tea.KeyMsg) (DashboardModel, tea.Cmd) {
       switch msg.String() {
       case "y", "Y":
           return m.confirmAction()
       case "n", "N", "esc":
           return m.cancelConfirmation()
       }
       return m, nil
   }

   // handleNormalMode handles input in normal (non-modal) state
   func (m DashboardModel) handleNormalMode(msg tea.KeyMsg) (DashboardModel, tea.Cmd) {
       switch msg.String() {
       case "q", "ctrl+c":
           return m, tea.Quit
       case "s":
           return m.toggleSprint()
       case "n":
           return m.startCreatingGoal()
       case "N":
           return m.startCreatingSubtask()
       case "/":
           m.search.Start()
           return m, nil
       case "j", "down":
           return m.moveSelectionDown()
       case "k", "up":
           return m.moveSelectionUp()
       // ... more keybindings
       }
       return m, nil
   }
   ```

2. **Simplify main `Update()` method:**
   ```go
   func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
       // Handle window resize
       if msg, ok := msg.(tea.WindowSizeMsg); ok {
           m.width = msg.Width
           m.height = msg.Height
           return m, nil
       }

       // Handle tick messages (timer)
       if _, ok := msg.(tickMsg); ok {
           return m.handleTick()
       }

       // Handle key messages based on current mode
       if msg, ok := msg.(tea.KeyMsg); ok {
           return m.routeKeyMessage(msg)
       }

       return m, nil
   }

   func (m DashboardModel) routeKeyMessage(msg tea.KeyMsg) (DashboardModel, tea.Cmd) {
       // Route to appropriate handler based on current state
       switch {
       case m.lock.Locked:
           return m.handleLockedState(msg)
       case m.search.Active:
           return m.handleSearching(msg)
       case m.creatingGoal:
           return m.handleCreatingGoal(msg)
       case m.editingGoal:
           return m.handleEditingGoal(msg)
       case m.confirmingDelete || m.confirmingClearDB:
           return m.handleConfirmation(msg)
       default:
           return m.handleNormalMode(msg)
       }
   }
   ```

---

### Task 2.3: Split Render Method

**Problem:** `View()` in `internal/tui/dashboard_render.go` is 1184 lines.

**Target Structure:**

1. **Extract `internal/tui/render_components.go`:**
   ```go
   package tui

   // renderHeader renders the top bar with workspace and timer
   func (m DashboardModel) renderHeader() string {
       // Timer bar rendering logic
   }

   // renderFooter renders the status bar and help text
   func (m DashboardModel) renderFooter() string {
       // Status and help rendering
   }

   // renderGoalList renders the goal/task column
   func (m DashboardModel) renderGoalList(goals []models.Goal, title string) string {
       // Goal list rendering
   }

   // renderSprintColumn renders the current sprint tasks
   func (m DashboardModel) renderSprintColumn() string {
       // Sprint column rendering
   }

   // renderBacklogColumn renders the backlog tasks
   func (m DashboardModel) renderBacklogColumn() string {
       // Backlog column rendering
   }
   ```

2. **Extract `internal/tui/render_modals.go`:**
   ```go
   package tui

   // renderLockScreen renders the passphrase entry screen
   func (m DashboardModel) renderLockScreen() string {
       // Lock screen rendering
   }

   // renderSearchOverlay renders the search input and results
   func (m DashboardModel) renderSearchOverlay() string {
       // Search overlay rendering
   }

   // renderConfirmDialog renders yes/no confirmation dialogs
   func (m DashboardModel) renderConfirmDialog(message string) string {
       // Confirmation dialog rendering
   }

   // renderGoalEditor renders the goal create/edit form
   func (m DashboardModel) renderGoalEditor() string {
       // Goal editor rendering
   }
   ```

3. **Simplify main `View()` method:**
   ```go
   func (m DashboardModel) View() string {
       // Handle lock screen
       if m.lock.Locked {
           return m.renderLockScreen()
       }

       // Build main layout
       header := m.renderHeader()

       var body string
       switch {
       case m.search.Active:
           body = m.renderSearchOverlay()
       case m.creatingGoal || m.editingGoal:
           body = m.renderGoalEditor()
       case m.confirmingDelete:
           body = m.renderConfirmDialog("Delete this task?")
       default:
           body = lipgloss.JoinHorizontal(
               lipgloss.Top,
               m.renderBacklogColumn(),
               m.renderSprintColumn(),
           )
       }

       footer := m.renderFooter()

       return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
   }
   ```

---

## IV. Phase 3: Quality (Medium Priority)

### Task 3.1: Extract Database Query Helpers

**Problem:** 14-column SELECT lists repeated 7+ times in `internal/database/goal.go`.

**Implementation:**

```go
// internal/database/goal.go - Add at top of file

const goalColumns = `id, parent_id, description, status, rank, priority, effort,
    tags, recurrence_rule, created_at, archived_at, task_started_at,
    task_elapsed_seconds, task_active`

// scanGoal scans a row into a Goal struct
func scanGoal(row interface{ Scan(...interface{}) error }) (models.Goal, error) {
    var g models.Goal
    err := row.Scan(
        &g.ID, &g.ParentID, &g.Description, &g.Status, &g.Rank,
        &g.Priority, &g.Effort, &g.Tags, &g.RecurrenceRule,
        &g.CreatedAt, &g.ArchivedAt, &g.TaskStartedAt,
        &g.TaskElapsedSeconds, &g.TaskActive,
    )
    return g, err
}

// queryGoals executes a goal query and returns results
func (d *Database) queryGoals(whereClause string, args ...interface{}) ([]models.Goal, error) {
    query := fmt.Sprintf("SELECT %s FROM goals WHERE %s", goalColumns, whereClause)
    rows, err := d.DB.Query(query, args...)
    if err != nil {
        return nil, &GoalError{Op: "query", Err: err}
    }
    defer rows.Close()

    var goals []models.Goal
    for rows.Next() {
        g, err := scanGoal(rows)
        if err != nil {
            return nil, &GoalError{Op: "scan", Err: err}
        }
        goals = append(goals, g)
    }
    return goals, rows.Err()
}

// Usage examples:
func (d *Database) GetBacklogGoals(workspaceID int64) ([]models.Goal, error) {
    return d.queryGoals(
        "sprint_id IS NULL AND status NOT IN ('completed', 'archived') AND workspace_id = ? ORDER BY rank ASC",
        workspaceID,
    )
}

func (d *Database) GetGoalsForSprint(sprintID int64) ([]models.Goal, error) {
    return d.queryGoals(
        "sprint_id = ? ORDER BY rank ASC",
        sprintID,
    )
}
```

---

### Task 3.2: Add Transaction Helper

**Problem:** Transaction boilerplate repeated throughout database code.

**Implementation:**

```go
// internal/database/db.go - Add helper function

// WithTx executes a function within a transaction
func (d *Database) WithTx(fn func(*sql.Tx) error) error {
    tx, err := d.DB.Begin()
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }

    if err := fn(tx); err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
        }
        return err
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("commit transaction: %w", err)
    }
    return nil
}

// Usage example:
func (d *Database) MoveGoalToSprint(goalID, sprintID int64) error {
    return d.WithTx(func(tx *sql.Tx) error {
        // Get max rank in target sprint
        var maxRank int
        err := tx.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id = ?", sprintID).Scan(&maxRank)
        if err != nil {
            return err
        }

        // Update goal
        _, err = tx.Exec("UPDATE goals SET sprint_id = ?, rank = ? WHERE id = ?",
            sprintID, maxRank+1, goalID)
        return err
    })
}
```

---

### Task 3.3: Replace sql.Null* with Pointers

**Problem:** Verbose null handling throughout codebase.

**Current:**
```go
// internal/models/models.go
type Goal struct {
    ParentID       sql.NullInt64
    SprintID       sql.NullInt64
    Notes          sql.NullString
    CompletedAt    sql.NullTime
    // ... many more
}

// Usage is verbose:
if g.ParentID.Valid {
    parentID := g.ParentID.Int64
}
```

**Target:**
```go
type Goal struct {
    ParentID    *int64
    SprintID    *int64
    Notes       *string
    CompletedAt *time.Time
}

// Usage is cleaner:
if g.ParentID != nil {
    parentID := *g.ParentID
}
```

**Migration Steps:**

1. Update `internal/models/models.go` field types
2. Update scan helpers to use pointer scanning:
   ```go
   func scanGoal(row Scanner) (models.Goal, error) {
       var g models.Goal
       err := row.Scan(
           &g.ID,
           &g.ParentID,  // sql.Scan handles *int64 correctly
           // ...
       )
       return g, err
   }
   ```
3. Update INSERT/UPDATE statements to handle nil pointers
4. Update all field access throughout codebase

---

### Task 3.4: Separate UI State from Data Models

**Problem:** `models.Goal` contains UI helper fields that don't belong in DB model.

**Current (`internal/models/models.go:73-77`):**
```go
type Goal struct {
    // Database fields
    ID          int64
    Description string
    // ...

    // UI Helper fields (not in DB)
    Subtasks []Goal
    Expanded bool
    Level    int
    Blocked  bool
}
```

**Target:**
```go
// internal/models/models.go - Pure data model
type Goal struct {
    ID          int64
    Description string
    // ... only database fields
}

// internal/tui/goal_view.go - UI wrapper
type GoalView struct {
    *models.Goal
    Subtasks []GoalView
    Expanded bool
    Level    int
    Blocked  bool
}

// BuildGoalTree converts flat goals to tree structure
func BuildGoalTree(goals []models.Goal) []GoalView {
    // Tree building logic from hierarchy.go
}
```

---

## V. Phase 4: Polish (Lower Priority)

### Task 4.1: Add Database Indexes

**Location:** `internal/database/db.go` in `migrate()` function

**Add:**
```go
// Add after table creation in migrate()
indexStatements := []string{
    `CREATE INDEX IF NOT EXISTS idx_goals_workspace_status
     ON goals(workspace_id, status)`,

    `CREATE INDEX IF NOT EXISTS idx_goals_sprint_id
     ON goals(sprint_id) WHERE sprint_id IS NOT NULL`,

    `CREATE INDEX IF NOT EXISTS idx_goals_parent_id
     ON goals(parent_id) WHERE parent_id IS NOT NULL`,

    `CREATE INDEX IF NOT EXISTS idx_sprints_day_id
     ON sprints(day_id)`,

    `CREATE INDEX IF NOT EXISTS idx_journal_entries_sprint_id
     ON journal_entries(sprint_id) WHERE sprint_id IS NOT NULL`,

    `CREATE INDEX IF NOT EXISTS idx_journal_entries_goal_id
     ON journal_entries(goal_id) WHERE goal_id IS NOT NULL`,

    `CREATE INDEX IF NOT EXISTS idx_task_deps_goal_id
     ON task_deps(goal_id)`,

    `CREATE INDEX IF NOT EXISTS idx_task_deps_depends_on_id
     ON task_deps(depends_on_id)`,
}

for _, stmt := range indexStatements {
    if _, err := db.Exec(stmt); err != nil {
        return fmt.Errorf("create index: %w", err)
    }
}
```

---

### Task 4.2: Add Common Utility Functions

**Create `internal/util/helpers.go`:**
```go
package util

// BoolToInt converts a boolean to 0 or 1
func BoolToInt(b bool) int {
    if b {
        return 1
    }
    return 0
}

// IntToBool converts 0/1 to boolean
func IntToBool(i int) bool {
    return i != 0
}

// Ptr returns a pointer to the value
func Ptr[T any](v T) *T {
    return &v
}

// Deref safely dereferences a pointer, returning zero value if nil
func Deref[T any](p *T) T {
    if p == nil {
        var zero T
        return zero
    }
    return *p
}

// Clamp constrains a value to a range
func Clamp(value, min, max int) int {
    if value < min {
        return min
    }
    if value > max {
        return max
    }
    return value
}
```

---

### Task 4.3: Define Named Constants

**Create `internal/config/constants.go`:**
```go
package config

import "time"

// Timer durations
const (
    SprintDuration = 90 * time.Minute
    BreakDuration  = 30 * time.Minute
    AutoLockAfter  = 10 * time.Minute
)

// View modes
const (
    ViewModeAll = iota
    ViewModeFocused
    ViewModeMinimal
)

// Task statuses
const (
    StatusPending   = "pending"
    StatusActive    = "active"
    StatusCompleted = "completed"
    StatusArchived  = "archived"
)

// Priority levels
const (
    PriorityLow    = "low"
    PriorityMedium = "medium"
    PriorityHigh   = "high"
    PriorityCritical = "critical"
)

// Database
const (
    AppName       = "sspt"
    DBFileName    = "sspt.db"
    MaxPassphraseAttempts = 5
)
```

---

## VI. Code Patterns & Standards

### Error Handling Pattern

```go
// DO: Return errors with context
func (d *Database) GetGoal(id int64) (*models.Goal, error) {
    goal, err := d.queryGoalByID(id)
    if err != nil {
        return nil, &GoalError{Op: "get", ID: id, Err: err}
    }
    return goal, nil
}

// DO: Handle errors at appropriate level
func (m DashboardModel) loadGoals() tea.Cmd {
    return func() tea.Msg {
        goals, err := m.db.GetBacklogGoals(m.workspaceID)
        if err != nil {
            return errMsg{err}  // Return error message for Update to handle
        }
        return goalsLoadedMsg{goals}
    }
}

// DON'T: Silently ignore errors
goals, _ := m.db.GetBacklogGoals(m.workspaceID)  // Bad!
```

### Database Access Pattern

```go
// DO: Use receiver methods with injected database
func (m DashboardModel) saveGoal(goal *models.Goal) error {
    return m.db.UpdateGoal(goal)
}

// DON'T: Use global database access
func (m DashboardModel) saveGoal(goal *models.Goal) error {
    return database.UpdateGoal(goal)  // Bad - uses global
}
```

### TUI State Management Pattern

```go
// DO: Use sub-models for related state
type DashboardModel struct {
    timer  TimerModel
    lock   LockModel
    search SearchModel
}

// DO: Route messages to appropriate handlers
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if m.lock.Locked {
        return m.handleLockedState(msg)
    }
    // ...
}

// DON'T: Put all state in one flat struct
type DashboardModel struct {
    locked bool
    passphraseHash string
    passphraseInput textinput.Model
    // ... 100 more fields
}
```

---

## VII. Testing Strategy

### Database Layer Testing

```go
// internal/database/db_test.go

func setupTestDB(t *testing.T) *Database {
    t.Helper()
    db, err := Open(":memory:", "")  // In-memory SQLite
    if err != nil {
        t.Fatalf("Failed to open test database: %v", err)
    }
    t.Cleanup(func() { db.Close() })
    return db
}

func TestAddGoal(t *testing.T) {
    db := setupTestDB(t)

    // Setup: create workspace
    wsID, err := db.CreateWorkspace("Test Workspace")
    if err != nil {
        t.Fatalf("Failed to create workspace: %v", err)
    }

    // Test
    err = db.AddGoal(wsID, "Test Goal", 0)
    if err != nil {
        t.Errorf("AddGoal failed: %v", err)
    }

    // Verify
    goals, err := db.GetBacklogGoals(wsID)
    if err != nil {
        t.Fatalf("GetBacklogGoals failed: %v", err)
    }
    if len(goals) != 1 {
        t.Errorf("Expected 1 goal, got %d", len(goals))
    }
    if goals[0].Description != "Test Goal" {
        t.Errorf("Expected 'Test Goal', got '%s'", goals[0].Description)
    }
}

func TestGoalHierarchy(t *testing.T) {
    db := setupTestDB(t)
    wsID, _ := db.CreateWorkspace("Test")

    // Create parent and child goals
    parentID, _ := db.AddGoalReturningID(wsID, "Parent", 0)
    _ = db.AddSubtask(parentID, "Child 1")
    _ = db.AddSubtask(parentID, "Child 2")

    // Verify hierarchy
    goals, _ := db.GetBacklogGoals(wsID)
    parent := findGoalByID(goals, parentID)
    if len(parent.Subtasks) != 2 {
        t.Errorf("Expected 2 subtasks, got %d", len(parent.Subtasks))
    }
}
```

### TUI Component Testing

```go
// internal/tui/timer_model_test.go

func TestTimerModel_StartSprint(t *testing.T) {
    timer := NewTimerModel()
    sprint := &models.Sprint{ID: 1}

    timer.StartSprint(sprint)

    if timer.ActiveSprint != sprint {
        t.Error("ActiveSprint not set")
    }
    if timer.Paused {
        t.Error("Timer should not be paused after start")
    }
}

func TestTimerModel_PauseResume(t *testing.T) {
    timer := NewTimerModel()
    timer.StartSprint(&models.Sprint{ID: 1})

    // Wait a bit
    time.Sleep(10 * time.Millisecond)

    timer.PauseSprint()
    if !timer.Paused {
        t.Error("Timer should be paused")
    }

    elapsed1 := timer.ElapsedTime()
    time.Sleep(10 * time.Millisecond)
    elapsed2 := timer.ElapsedTime()

    // Elapsed should not change while paused
    if elapsed2 != elapsed1 {
        t.Error("Elapsed time should not change while paused")
    }

    timer.ResumeSprint()
    time.Sleep(10 * time.Millisecond)
    elapsed3 := timer.ElapsedTime()

    if elapsed3 <= elapsed1 {
        t.Error("Elapsed time should increase after resume")
    }
}
```

### Integration Testing

```go
// internal/tui/dashboard_test.go

func TestDashboard_CreateGoal(t *testing.T) {
    db := database.SetupTestDB(t)
    m := InitialModel(db)

    // Simulate pressing 'n' to create goal
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

    if !m.creatingGoal {
        t.Error("Should be in goal creation mode")
    }

    // Type goal description
    for _, r := range "Test Goal" {
        m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
    }

    // Press enter to submit
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

    if m.creatingGoal {
        t.Error("Should have exited goal creation mode")
    }

    // Verify goal was created
    goals, _ := db.GetBacklogGoals(m.currentWorkspace.ID)
    if len(goals) == 0 {
        t.Error("Goal should have been created")
    }
}
```

---

## Appendix: File Checklist

### Files to Create
- [ ] `internal/tui/timer_model.go`
- [ ] `internal/tui/lock_model.go`
- [ ] `internal/tui/search_model.go`
- [ ] `internal/tui/update_handlers.go`
- [ ] `internal/tui/render_components.go`
- [ ] `internal/tui/render_modals.go`
- [ ] `internal/database/errors.go`
- [ ] `internal/util/helpers.go`
- [ ] `internal/util/logging.go`
- [ ] `internal/config/constants.go`

### Files to Heavily Modify
- [ ] `cmd/app/main.go` - Dependency injection
- [ ] `internal/database/db.go` - Remove globals, add helpers
- [ ] `internal/database/goal.go` - Remove wrappers, add query helpers
- [ ] `internal/database/sprint.go` - Remove wrappers
- [ ] `internal/database/workspace.go` - Remove wrappers
- [ ] `internal/tui/dashboard.go` - Use sub-models
- [ ] `internal/tui/dashboard_update.go` - Extract handlers
- [ ] `internal/tui/dashboard_render.go` - Extract components
- [ ] `internal/models/models.go` - Replace sql.Null*, remove UI fields

### Files to Add Tests For
- [ ] `internal/database/goal_test.go`
- [ ] `internal/database/sprint_test.go`
- [ ] `internal/database/workspace_test.go`
- [ ] `internal/tui/timer_model_test.go`
- [ ] `internal/tui/lock_model_test.go`
- [ ] `internal/tui/search_model_test.go`
- [ ] `internal/tui/dashboard_integration_test.go`
