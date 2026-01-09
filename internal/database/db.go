package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB
var dbFile string

var (
	ErrEncrypted            = errors.New("database is encrypted")
	ErrSQLCipherUnavailable = errors.New("sqlcipher is unavailable")
)

var (
	cipherAvailable bool
	dbEncrypted     bool
	cipherVersion   string
)

// InitDB initializes the database connection and schema.
func InitDB(filepath, key string) error {
	var err error
	dbFile = filepath
	DB, err = openDB(filepath, key)
	if err != nil {
		return err
	}
	cipherAvailable, cipherVersion = detectSQLCipher()
	if sqlcipherCompiled() {
		cipherAvailable = true
	}
	if err := verifyDB(key); err != nil {
		return err
	}

	createTables()
	return nil
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS workspaces (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			slug TEXT UNIQUE,
			view_mode INTEGER DEFAULT 0,
			theme TEXT DEFAULT 'default',
			show_backlog INTEGER DEFAULT 1,
			show_completed INTEGER DEFAULT 1,
			show_archived INTEGER DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS days (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL UNIQUE,
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS sprints (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			day_id INTEGER NOT NULL,
			workspace_id INTEGER,
			sprint_number INTEGER NOT NULL,
			status TEXT DEFAULT 'pending',
			start_time DATETIME,
			end_time DATETIME,
			last_paused_at DATETIME,
			elapsed_seconds INTEGER DEFAULT 0,
			FOREIGN KEY(day_id) REFERENCES days(id),
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id)
		);`,
		`CREATE TABLE IF NOT EXISTS goals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_id INTEGER,
			workspace_id INTEGER,
			sprint_id INTEGER,
			description TEXT NOT NULL,
			notes TEXT,
			status TEXT DEFAULT 'pending',
			priority INTEGER DEFAULT 3,
			effort TEXT DEFAULT 'M',
			tags TEXT,
			recurrence_rule TEXT,
			links TEXT,
			rank INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			archived_at DATETIME,
			FOREIGN KEY(sprint_id) REFERENCES sprints(id),
			FOREIGN KEY(parent_id) REFERENCES goals(id),
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id)
		);`,
		`CREATE TABLE IF NOT EXISTS journal_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			day_id INTEGER NOT NULL,
			workspace_id INTEGER,
			sprint_id INTEGER,
			goal_id INTEGER,
			content TEXT NOT NULL,
			tags TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(day_id) REFERENCES days(id),
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id),
			FOREIGN KEY(sprint_id) REFERENCES sprints(id),
			FOREIGN KEY(goal_id) REFERENCES goals(id)
		);`,
		`CREATE TABLE IF NOT EXISTS task_deps (
			goal_id INTEGER NOT NULL,
			depends_on_id INTEGER NOT NULL,
			PRIMARY KEY (goal_id, depends_on_id),
			FOREIGN KEY(goal_id) REFERENCES goals(id),
			FOREIGN KEY(depends_on_id) REFERENCES goals(id)
		);`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT
		);`,
	}

	for _, query := range queries {
		_, err := DB.Exec(query)
		if err != nil {
			log.Fatalf("Error creating table: %q: %s\n", err, query)
		}
	}

	// Migrations for existing databases
	migrate()
}

func migrate() {
	// Add last_paused_at to sprints
	_, _ = DB.Exec("ALTER TABLE sprints ADD COLUMN last_paused_at DATETIME")
	// Add elapsed_seconds to sprints
	_, _ = DB.Exec("ALTER TABLE sprints ADD COLUMN elapsed_seconds INTEGER DEFAULT 0")
	// Add rank to goals
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN rank INTEGER DEFAULT 0")

	// V3 Migrations
	// Goals
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN parent_id INTEGER")
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN workspace_id INTEGER")
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN notes TEXT")
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN priority INTEGER DEFAULT 3")
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN recurrence_rule TEXT")
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN archived_at DATETIME")
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN effort TEXT DEFAULT 'M'")
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN tags TEXT")
	_, _ = DB.Exec("ALTER TABLE goals ADD COLUMN links TEXT")

	// Sprints
	_, _ = DB.Exec("ALTER TABLE sprints ADD COLUMN workspace_id INTEGER")
	// Backfill legacy sprints to default workspace (1)
	_, _ = DB.Exec("UPDATE sprints SET workspace_id = (SELECT id FROM workspaces WHERE slug = 'personal') WHERE workspace_id IS NULL")

	// Journal
	_, _ = DB.Exec("ALTER TABLE journal_entries ADD COLUMN goal_id INTEGER")
	_, _ = DB.Exec("ALTER TABLE journal_entries ADD COLUMN tags TEXT")
	_, _ = DB.Exec("ALTER TABLE journal_entries ADD COLUMN workspace_id INTEGER")
	// Backfill legacy journal entries
	_, _ = DB.Exec("UPDATE journal_entries SET workspace_id = (SELECT id FROM workspaces WHERE slug = 'personal') WHERE workspace_id IS NULL")

	// Backfill legacy goals
	_, _ = DB.Exec("UPDATE goals SET workspace_id = (SELECT id FROM workspaces WHERE slug = 'personal') WHERE workspace_id IS NULL")

	// Workspaces
	_, _ = DB.Exec("ALTER TABLE workspaces ADD COLUMN view_mode INTEGER DEFAULT 0")
	_, _ = DB.Exec("ALTER TABLE workspaces ADD COLUMN theme TEXT DEFAULT 'default'")
	_, _ = DB.Exec("ALTER TABLE workspaces ADD COLUMN show_backlog INTEGER DEFAULT 1")
	_, _ = DB.Exec("ALTER TABLE workspaces ADD COLUMN show_completed INTEGER DEFAULT 1")
	_, _ = DB.Exec("ALTER TABLE workspaces ADD COLUMN show_archived INTEGER DEFAULT 0")

	// Task dependencies
	_, _ = DB.Exec(`CREATE TABLE IF NOT EXISTS task_deps (
		goal_id INTEGER NOT NULL,
		depends_on_id INTEGER NOT NULL,
		PRIMARY KEY (goal_id, depends_on_id),
		FOREIGN KEY(goal_id) REFERENCES goals(id),
		FOREIGN KEY(depends_on_id) REFERENCES goals(id)
	);`)

	// Settings
	_, _ = DB.Exec(`CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);`)
}

func RekeyDB(key string) error {
	if key != "" {
		_, _ = DB.Exec("PRAGMA key = ''")
	}
	_, err := DB.Exec(fmt.Sprintf("PRAGMA rekey = '%s'", escapeSQLiteString(key)))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such pragma") {
			return ErrSQLCipherUnavailable
		}
		return err
	}
	cipherAvailable, cipherVersion = detectSQLCipher()
	if sqlcipherCompiled() {
		cipherAvailable = true
	}
	if key != "" && dbFile != "" {
		enc, encErr := isEncryptedFile(dbFile)
		if encErr != nil {
			return encErr
		}
		if !enc {
			return fmt.Errorf("rekey did not encrypt database")
		}
	}
	dbEncrypted = key != ""
	return nil
}

func EncryptDatabase(key string) error {
	if key == "" {
		return fmt.Errorf("passphrase required")
	}
	if dbFile == "" {
		return fmt.Errorf("database path unavailable")
	}
	if !sqlcipherCompiled() {
		return ErrSQLCipherUnavailable
	}
	if err := DB.Close(); err != nil {
		return err
	}

	tempPath := dbFile + ".enc"
	backupPath := dbFile + ".bak"
	_ = os.Remove(tempPath)
	_ = os.Remove(backupPath)

	// Open plaintext DB without a key.
	plainDB, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return err
	}

	if err := plainDB.Ping(); err != nil {
		_ = plainDB.Close()
		return fmt.Errorf("plaintext ping failed: %w", err)
	}
	var count int
	if err := plainDB.QueryRow("SELECT COUNT(1) FROM sqlite_master").Scan(&count); err != nil {
		_ = plainDB.Close()
		return fmt.Errorf("plaintext check failed: %w", err)
	}

	if _, err := plainDB.Exec("ATTACH DATABASE ':memory:' AS probe"); err == nil {
		_, _ = plainDB.Exec("DETACH DATABASE probe")
	}
	attach := fmt.Sprintf("ATTACH DATABASE '%s' AS enc KEY '%s'", escapeSQLiteString(tempPath), escapeSQLiteString(key))
	if _, err := plainDB.Exec(attach); err != nil {
		_ = plainDB.Close()
		return fmt.Errorf("attach encrypted failed: %w", err)
	}
	if _, err := plainDB.Exec("SELECT sqlcipher_export('enc')"); err != nil {
		_, _ = plainDB.Exec("DETACH DATABASE enc")
		_ = plainDB.Close()
		return fmt.Errorf("sqlcipher_export failed: %w", err)
	}
	if _, err := plainDB.Exec("DETACH DATABASE enc"); err != nil {
		_ = plainDB.Close()
		return fmt.Errorf("detach encrypted failed: %w", err)
	}
	if err := plainDB.Close(); err != nil {
		return fmt.Errorf("plaintext close failed: %w", err)
	}

	if enc, encErr := isEncryptedFile(tempPath); encErr == nil && !enc {
		return fmt.Errorf("export produced plaintext (encryption not applied)")
	} else if encErr != nil {
		return fmt.Errorf("encrypted probe failed: %w", encErr)
	}

	encDB, err := openDB(tempPath, key)
	if err != nil {
		return fmt.Errorf("encrypted open failed: %w", err)
	}
	if _, err := encDB.Exec("PRAGMA cipher_migrate"); err != nil {
		_ = encDB.Close()
		return fmt.Errorf("encrypted migrate failed: %w", err)
	}
	if err := encDB.QueryRow("SELECT COUNT(1) FROM sqlite_master").Scan(&count); err != nil {
		_ = encDB.Close()
		return fmt.Errorf("encrypted verify failed: %w", err)
	}
	if err := encDB.Close(); err != nil {
		return fmt.Errorf("encrypted close failed: %w", err)
	}

	if err := os.Rename(dbFile, backupPath); err != nil {
		return fmt.Errorf("backup rename failed: %w", err)
	}
	if err := os.Rename(tempPath, dbFile); err != nil {
		_ = os.Rename(backupPath, dbFile)
		_ = os.Remove(tempPath)
		return fmt.Errorf("encrypted rename failed: %w", err)
	}
	if err := InitDB(dbFile, key); err != nil {
		_ = os.Rename(dbFile, tempPath)
		_ = os.Rename(backupPath, dbFile)
		_ = os.Remove(tempPath)
		return fmt.Errorf("reopen encrypted failed: %w", err)
	}
	_ = os.Remove(backupPath)
	_ = os.Remove(tempPath)
	return nil
}

func openDB(filepath, key string) (*sql.DB, error) {
	dsn := filepath
	if sqlcipherCompiled() {
		dsn = fmt.Sprintf("file:%s?mode=rwc&cache=shared", filepath)
		if key != "" {
			dsn = dsn + "&_key=" + url.QueryEscape(key)
		}
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	return db, nil
}

func verifyDB(key string) error {
	var count int
	err := DB.QueryRow("SELECT COUNT(1) FROM sqlite_master").Scan(&count)
	if err == nil {
		if key != "" {
			dbEncrypted = true
		} else if dbFile != "" {
			if enc, encErr := isEncryptedFile(dbFile); encErr == nil {
				dbEncrypted = enc
			}
		}
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "file is encrypted") || strings.Contains(msg, "not a database") {
		if key == "" {
			dbEncrypted = true
			return ErrEncrypted
		}
		return err
	}
	return err
}

func detectSQLCipher() (bool, string) {
	var version string
	err := DB.QueryRow("PRAGMA cipher_version").Scan(&version)
	if err == nil {
		version = strings.TrimSpace(version)
		if version != "" {
			return true, version
		}
	}
	rows, err := DB.Query("PRAGMA compile_options")
	if err != nil {
		return false, ""
	}
	defer rows.Close()
	hasCodec := false
	for rows.Next() {
		var opt string
		if scanErr := rows.Scan(&opt); scanErr != nil {
			continue
		}
		opt = strings.ToUpper(opt)
		if strings.Contains(opt, "SQLCIPHER") || strings.Contains(opt, "SQLITE_HAS_CODEC") {
			hasCodec = true
			break
		}
	}
	if hasCodec {
		return true, ""
	}
	return false, ""
}

func EncryptionStatus() (bool, bool, string) {
	return cipherAvailable, dbEncrypted, cipherVersion
}

func escapeSQLiteString(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func isEncryptedFile(path string) (bool, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	var count int
	err = conn.QueryRow("SELECT COUNT(1) FROM sqlite_master").Scan(&count)
	if err == nil {
		return false, nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "file is encrypted") || strings.Contains(msg, "not a database") {
		return true, nil
	}
	return false, err
}

// IsEncryptedFile reports whether the database file appears encrypted.
func IsEncryptedFile(path string) (bool, error) {
	return isEncryptedFile(path)
}

func DatabaseHasData() bool {
	var count int
	if err := DB.QueryRow("SELECT COUNT(1) FROM goals").Scan(&count); err == nil && count > 0 {
		return true
	}
	if err := DB.QueryRow("SELECT COUNT(1) FROM sprints").Scan(&count); err == nil && count > 0 {
		return true
	}
	if err := DB.QueryRow("SELECT COUNT(1) FROM journal_entries").Scan(&count); err == nil && count > 0 {
		return true
	}
	return false
}

func RecreateEncryptedDatabase(key string) error {
	if key == "" {
		return fmt.Errorf("passphrase required")
	}
	if dbFile == "" {
		return fmt.Errorf("database path unavailable")
	}
	if !sqlcipherCompiled() {
		return ErrSQLCipherUnavailable
	}
	backupPath := dbFile + ".bak"
	_ = os.Remove(backupPath)
	if err := DB.Close(); err != nil {
		return err
	}
	if err := os.Rename(dbFile, backupPath); err != nil {
		return fmt.Errorf("backup rename failed: %w", err)
	}
	if err := InitDB(dbFile, key); err != nil {
		_ = os.Remove(dbFile)
		_ = os.Rename(backupPath, dbFile)
		return fmt.Errorf("recreate encrypted failed: %w", err)
	}
	_ = os.Remove(backupPath)
	return nil
}

func ClearDatabase() error {
	if dbFile == "" {
		return fmt.Errorf("database path unavailable")
	}
	if DB != nil {
		if err := DB.Close(); err != nil {
			return err
		}
	}
	_ = os.Remove(dbFile)
	_ = os.Remove(dbFile + ".bak")
	_ = os.Remove(dbFile + ".enc")
	return InitDB(dbFile, "")
}

// --- Settings ---

func GetSetting(key string) (string, bool) {
	var value sql.NullString
	err := DB.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", false
	}
	if value.Valid {
		return value.String, true
	}
	return "", false
}

func SetSetting(key, value string) error {
	_, err := DB.Exec("INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
	return err
}

// --- Workspaces ---

func GetWorkspaces() ([]models.Workspace, error) {
	rows, err := DB.Query("SELECT id, name, slug, view_mode, theme, show_backlog, show_completed, show_archived FROM workspaces ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ws []models.Workspace
	for rows.Next() {
		var w models.Workspace
		var viewMode sql.NullInt64
		var theme sql.NullString
		var showBacklog, showCompleted, showArchived sql.NullInt64

		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &viewMode, &theme, &showBacklog, &showCompleted, &showArchived); err != nil {
			return nil, err
		}

		if viewMode.Valid {
			w.ViewMode = int(viewMode.Int64)
		} else {
			w.ViewMode = 0
		}

		if theme.Valid {
			w.Theme = theme.String
		} else {
			w.Theme = "default"
		}
		if showBacklog.Valid {
			w.ShowBacklog = showBacklog.Int64 != 0
		} else {
			w.ShowBacklog = true
		}
		if showCompleted.Valid {
			w.ShowCompleted = showCompleted.Int64 != 0
		} else {
			w.ShowCompleted = true
		}
		if showArchived.Valid {
			w.ShowArchived = showArchived.Int64 != 0
		} else {
			w.ShowArchived = false
		}

		ws = append(ws, w)
	}
	return ws, nil
}

func UpdateWorkspaceViewMode(workspaceID int64, mode int) error {
	_, err := DB.Exec("UPDATE workspaces SET view_mode = ? WHERE id = ?", mode, workspaceID)
	return err
}

func UpdateWorkspaceTheme(workspaceID int64, theme string) error {
	_, err := DB.Exec("UPDATE workspaces SET theme = ? WHERE id = ?", theme, workspaceID)
	return err
}

func UpdateWorkspacePaneVisibility(workspaceID int64, showBacklog, showCompleted, showArchived bool) error {
	backlog := 0
	completed := 0
	archived := 0
	if showBacklog {
		backlog = 1
	}
	if showCompleted {
		completed = 1
	}
	if showArchived {
		archived = 1
	}
	_, err := DB.Exec("UPDATE workspaces SET show_backlog = ?, show_completed = ?, show_archived = ? WHERE id = ?", backlog, completed, archived, workspaceID)
	return err
}

// --- Dependencies ---

func AddGoalDependency(goalID, dependsOnID int64) error {
	goalWS, ok := getGoalWorkspaceID(goalID)
	if !ok {
		return nil
	}
	depWS, ok := getGoalWorkspaceID(dependsOnID)
	if !ok || depWS != goalWS {
		return nil
	}
	_, err := DB.Exec("INSERT OR IGNORE INTO task_deps (goal_id, depends_on_id) VALUES (?, ?)", goalID, dependsOnID)
	return err
}

func RemoveGoalDependency(goalID, dependsOnID int64) error {
	_, err := DB.Exec("DELETE FROM task_deps WHERE goal_id = ? AND depends_on_id = ?", goalID, dependsOnID)
	return err
}

func GetGoalDependencies(goalID int64) (map[int64]bool, error) {
	rows, err := DB.Query("SELECT depends_on_id FROM task_deps WHERE goal_id = ?", goalID)
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
	goalWS, ok := getGoalWorkspaceID(goalID)
	if !ok {
		return nil
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM task_deps WHERE goal_id = ?", goalID); err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, id := range deps {
		if id == goalID {
			continue
		}
		depWS, ok := getGoalWorkspaceID(id)
		if !ok || depWS != goalWS {
			continue
		}
		if _, err := tx.Exec("INSERT OR IGNORE INTO task_deps (goal_id, depends_on_id) VALUES (?, ?)", goalID, id); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func regenerateRecurringGoal(goalID int64) error {
	var g models.Goal
	err := DB.QueryRow(`
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
		_ = DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id IS NULL AND workspace_id = ?", g.WorkspaceID.Int64).Scan(&maxRank)
	}
	var wsID interface{} = nil
	if g.WorkspaceID.Valid {
		wsID = g.WorkspaceID.Int64
	}
	_, err = DB.Exec(`INSERT INTO goals (workspace_id, description, sprint_id, status, rank, tags, notes, priority, effort, recurrence_rule)
		VALUES (?, ?, NULL, 'pending', ?, ?, ?, ?, ?, ?)`,
		wsID, g.Description, maxRank+1, g.Tags, g.Notes, g.Priority, g.Effort, g.RecurrenceRule,
	)
	return err
}

func getGoalWorkspaceID(goalID int64) (int64, bool) {
	var wsID sql.NullInt64
	err := DB.QueryRow("SELECT workspace_id FROM goals WHERE id = ?", goalID).Scan(&wsID)
	if err != nil || !wsID.Valid {
		return 0, false
	}
	return wsID.Int64, true
}

func IsGoalBlocked(goalID int64) (bool, error) {
	var count int
	err := DB.QueryRow(`
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
	rows, err := DB.Query(`
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

func EnsureDefaultWorkspace() (int64, error) {
	var id int64
	err := DB.QueryRow("SELECT id FROM workspaces WHERE slug = 'personal'").Scan(&id)
	if err == sql.ErrNoRows {
		res, err := DB.Exec("INSERT INTO workspaces (name, slug) VALUES ('Personal', 'personal')")
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	return id, err
}

func CreateWorkspace(name, slug string) (int64, error) {
	res, err := DB.Exec("INSERT INTO workspaces (name, slug) VALUES (?, ?)", name, slug)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func GetWorkspaceIDBySlug(slug string) (int64, bool, error) {
	var id int64
	err := DB.QueryRow("SELECT id FROM workspaces WHERE slug = ?", slug).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

// --- Day / Sprint Management ---

// GetAdjacentDay finds the previous (direction < 0) or next (direction > 0) day ID relative to the current one.
// Returns the new Day ID and its Date string.
func GetAdjacentDay(currentDayID int64, direction int) (int64, string, error) {
	var query string
	if direction < 0 {
		query = "SELECT id, date FROM days WHERE id < ? ORDER BY id DESC LIMIT 1"
	} else {
		query = "SELECT id, date FROM days WHERE id > ? ORDER BY id ASC LIMIT 1"
	}

	var id int64
	var date string
	err := DB.QueryRow(query, currentDayID).Scan(&id, &date)
	if err != nil {
		return 0, "", err
	}
	return id, date, nil
}

// CheckCurrentDay returns the Day ID if it exists for the current date.
func CheckCurrentDay() int64 {
	dateStr := time.Now().Format("2006-01-02")
	var id int64
	err := DB.QueryRow("SELECT id FROM days WHERE date = ?", dateStr).Scan(&id)
	if err == sql.ErrNoRows {
		return 0
	} else if err != nil {
		log.Printf("Error checking day: %v", err)
		return 0
	}
	return id
}

// BootstrapDay creates the day record and pre-allocates the chosen number of sprints for a workspace.
func BootstrapDay(workspaceID int64, numSprints int) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}

	dateStr := time.Now().Format("2006-01-02")
	_, err = tx.Exec("INSERT OR IGNORE INTO days (date) VALUES (?)", dateStr)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to ensure day: %w", err)
	}

	var dayID int64
	err = tx.QueryRow("SELECT id FROM days WHERE date = ?", dateStr).Scan(&dayID)
	if err != nil {
		tx.Rollback()
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO sprints (day_id, workspace_id, sprint_number) VALUES (?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for i := 1; i <= numSprints; i++ {
		_, err = stmt.Exec(dayID, workspaceID, i)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert sprint %d: %w", i, err)
		}
	}

	return tx.Commit()
}

// GetDay retrieves the full Day struct by ID.
func GetDay(id int64) (models.Day, error) {
	var d models.Day
	var dateStr string

	err := DB.QueryRow("SELECT id, date, started_at FROM days WHERE id = ?", id).
		Scan(&d.ID, &dateStr, &d.StartedAt)

	if err != nil {
		return d, err
	}
	d.Date = dateStr
	return d, nil
}

// GetSprints retrieves all sprints for a given day and workspace, ordered by number.
func GetSprints(dayID int64, workspaceID int64) ([]models.Sprint, error) {
	rows, err := DB.Query(`
		SELECT id, day_id, workspace_id, sprint_number, status, start_time, end_time, last_paused_at, elapsed_seconds
		FROM sprints 
		WHERE day_id = ? AND workspace_id = ?
		ORDER BY sprint_number ASC`, dayID, workspaceID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sprints []models.Sprint
	for rows.Next() {
		var s models.Sprint
		err := rows.Scan(
			&s.ID,
			&s.DayID,
			&s.WorkspaceID,
			&s.SprintNumber,
			&s.Status,
			&s.StartTime,
			&s.EndTime,
			&s.LastPausedAt,
			&s.ElapsedSeconds,
		)
		if err != nil {
			return nil, err
		}
		sprints = append(sprints, s)
	}
	return sprints, nil
}

// --- Goals (Tasks) ---

// GetBacklogGoals retrieves goals that are not assigned to any sprint and belong to the workspace.
func GetBacklogGoals(workspaceID int64) ([]models.Goal, error) {
	rows, err := DB.Query(`
		SELECT id, parent_id, description, status, rank, priority, effort, tags, recurrence_rule, created_at, archived_at 
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
		if err := rows.Scan(&g.ID, &g.ParentID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.Tags, &g.RecurrenceRule, &g.CreatedAt, &g.ArchivedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

// GetGoalsForSprint retrieves goals for a specific sprint ID.
func GetGoalsForSprint(sprintID int64) ([]models.Goal, error) {
	rows, err := DB.Query(`
		SELECT id, parent_id, sprint_id, description, status, rank, priority, effort, tags, recurrence_rule, created_at, archived_at 
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
		if err := rows.Scan(&g.ID, &g.ParentID, &g.SprintID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.Tags, &g.RecurrenceRule, &g.CreatedAt, &g.ArchivedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

// AddGoal inserts a new goal into the database.
func AddGoal(workspaceID int64, description string, sprintID int64) error {
	var maxRank int
	var err error
	if sprintID > 0 {
		err = DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id = ?", sprintID).Scan(&maxRank)
	} else {
		err = DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id IS NULL AND workspace_id = ?", workspaceID).Scan(&maxRank)
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

	_, err = DB.Exec(query, workspaceID, description, sprintIDArg, maxRank+1, tags)
	return err
}

func UpdateGoalPriority(goalID int64, priority int) error {
	if priority < 1 {
		priority = 1
	}
	if priority > 5 {
		priority = 5
	}
	_, err := DB.Exec("UPDATE goals SET priority = ? WHERE id = ?", priority, goalID)
	return err
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
	var maxRank int
	var err error
	if sprintID > 0 {
		err = DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id = ?", sprintID).Scan(&maxRank)
	} else {
		err = DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE sprint_id IS NULL AND workspace_id = ?", workspaceID).Scan(&maxRank)
	}
	if err != nil {
		return err
	}

	priority := seed.Priority
	if priority == 0 {
		priority = 3
	}
	effort := strings.TrimSpace(seed.Effort)
	if effort == "" {
		effort = "M"
	}
	tags := seed.Tags
	if len(tags) == 0 {
		tags = util.ExtractTags(seed.Description)
	}
	tagsJSON := util.TagsToJSON(tags)
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

	_, err = DB.Exec(`INSERT INTO goals (workspace_id, description, sprint_id, status, rank, tags, priority, effort, notes, recurrence_rule, links)
		VALUES (?, ?, ?, 'pending', ?, ?, ?, ?, ?, ?, ?)`,
		workspaceID, seed.Description, sprintIDArg, maxRank+1, tagsJSON, priority, effort, notesArg, recurrenceArg, string(linksJSON))
	return err
}

// GetCompletedGoalsForDay retrieves all goals completed on a specific day and workspace across all sprints.
func GetCompletedGoalsForDay(dayID int64, workspaceID int64) ([]models.Goal, error) {
	dateStr := ""
	err := DB.QueryRow("SELECT date FROM days WHERE id = ?", dayID).Scan(&dateStr)
	if err != nil {
		return nil, err
	}

	rows, err := DB.Query(`
		SELECT id, parent_id, description, status, rank, priority, effort, tags, recurrence_rule, created_at, archived_at 
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
		if err := rows.Scan(&g.ID, &g.ParentID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.Tags, &g.RecurrenceRule, &g.CreatedAt, &g.ArchivedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

// AddSubtask inserts a new subtask linked to a parent goal.
func AddSubtask(description string, parentID int64) error {
	// Inherit sprint_id and workspace_id from parent
	var sprintID sql.NullInt64
	var workspaceID sql.NullInt64
	err := DB.QueryRow("SELECT sprint_id, workspace_id FROM goals WHERE id = ?", parentID).Scan(&sprintID, &workspaceID)
	if err != nil {
		return err
	}

	// Calculate rank among siblings
	var maxRank int
	err = DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE parent_id = ?", parentID).Scan(&maxRank)
	if err != nil {
		return err
	}

	tags := util.TagsToJSON(util.ExtractTags(description))
	_, err = DB.Exec(`INSERT INTO goals (description, parent_id, sprint_id, workspace_id, status, rank, tags) VALUES (?, ?, ?, ?, 'pending', ?, ?)`,
		description, parentID, sprintID, workspaceID, maxRank+1, tags)
	return err
}

func AddSubtaskDetailed(parentID int64, seed GoalSeed) error {
	var sprintID sql.NullInt64
	var workspaceID sql.NullInt64
	err := DB.QueryRow("SELECT sprint_id, workspace_id FROM goals WHERE id = ?", parentID).Scan(&sprintID, &workspaceID)
	if err != nil {
		return err
	}

	var maxRank int
	if err := DB.QueryRow("SELECT COALESCE(MAX(rank), 0) FROM goals WHERE parent_id = ?", parentID).Scan(&maxRank); err != nil {
		return err
	}

	priority := seed.Priority
	if priority == 0 {
		priority = 3
	}
	effort := strings.TrimSpace(seed.Effort)
	if effort == "" {
		effort = "M"
	}
	tags := seed.Tags
	if len(tags) == 0 {
		tags = util.ExtractTags(seed.Description)
	}
	tagsJSON := util.TagsToJSON(tags)
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

	_, err = DB.Exec(`INSERT INTO goals (description, parent_id, sprint_id, workspace_id, status, rank, tags, priority, effort, notes, recurrence_rule, links)
		VALUES (?, ?, ?, ?, 'pending', ?, ?, ?, ?, ?, ?, ?)`,
		seed.Description, parentID, sprintID, workspaceID, maxRank+1, tagsJSON, priority, effort, notesArg, recurrenceArg, string(linksJSON))
	return err
}

// --- Sprint Lifecycle ---

func StartSprint(sprintID int64) error {
	_, err := DB.Exec("UPDATE sprints SET status = 'active', start_time = CURRENT_TIMESTAMP WHERE id = ?", sprintID)
	return err
}

func PauseSprint(sprintID int64, elapsedSeconds int) error {
	_, err := DB.Exec(`
		UPDATE sprints 
		SET status = 'paused', 
		    elapsed_seconds = ?, 
		    last_paused_at = CURRENT_TIMESTAMP 
		WHERE id = ?`, elapsedSeconds, sprintID)
	return err
}

func CompleteSprint(sprintID int64) error {
	_, err := DB.Exec("UPDATE sprints SET status = 'completed', end_time = CURRENT_TIMESTAMP WHERE id = ?", sprintID)
	return err
}

func ResetSprint(sprintID int64) error {
	_, err := DB.Exec("UPDATE sprints SET status = 'pending', start_time = NULL, elapsed_seconds = 0, last_paused_at = NULL WHERE id = ?", sprintID)
	return err
}

// --- Task Management ---

func UpdateGoalStatus(goalID int64, status string) error {
	completedAt := "NULL"
	if status == "completed" {
		completedAt = "CURRENT_TIMESTAMP"
	}
	query := fmt.Sprintf("UPDATE goals SET status = ?, completed_at = %s WHERE id = ?", completedAt)
	_, err := DB.Exec(query, status, goalID)
	if err != nil {
		return err
	}
	if status == "completed" {
		return regenerateRecurringGoal(goalID)
	}
	return nil
}

func SwapGoalRanks(goalID1, goalID2 int64) error {
	var rank1, rank2 int
	err := DB.QueryRow("SELECT rank FROM goals WHERE id = ?", goalID1).Scan(&rank1)
	if err != nil {
		return err
	}
	err = DB.QueryRow("SELECT rank FROM goals WHERE id = ?", goalID2).Scan(&rank2)
	if err != nil {
		return err
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE goals SET rank = ? WHERE id = ?", rank2, goalID1)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("UPDATE goals SET rank = ? WHERE id = ?", rank1, goalID2)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func AppendSprint(dayID int64, workspaceID int64) error {
	var lastSprintNum int
	err := DB.QueryRow("SELECT COALESCE(MAX(sprint_number), 0) FROM sprints WHERE day_id = ? AND workspace_id = ?", dayID, workspaceID).Scan(&lastSprintNum)
	if err != nil {
		return err
	}

	_, err = DB.Exec("INSERT INTO sprints (day_id, workspace_id, sprint_number) VALUES (?, ?, ?)", dayID, workspaceID, lastSprintNum+1)
	return err
}

func MoveGoal(goalID int64, targetSprintID int64) error {
	var sprintArg interface{}
	if targetSprintID == 0 {
		sprintArg = nil // SQL NULL for Backlog
	} else {
		sprintArg = targetSprintID
	}

	_, err := DB.Exec("UPDATE goals SET sprint_id = ? WHERE id = ?", sprintArg, goalID)
	return err
}

func EditGoal(goalID int64, newDescription string) error {
	tags := util.TagsToJSON(util.ExtractTags(newDescription))
	_, err := DB.Exec("UPDATE goals SET description = ?, tags = ? WHERE id = ?", newDescription, tags, goalID)
	return err
}

func UpdateGoalRecurrence(goalID int64, rule string) error {
	if strings.TrimSpace(rule) == "" {
		_, err := DB.Exec("UPDATE goals SET recurrence_rule = NULL WHERE id = ?", goalID)
		return err
	}
	_, err := DB.Exec("UPDATE goals SET recurrence_rule = ? WHERE id = ?", rule, goalID)
	return err
}

func DeleteGoal(goalID int64) error {
	_, err := DB.Exec("DELETE FROM goals WHERE id = ?", goalID)
	return err
}

// AddTagsToGoal appends new tags to a goal, avoiding duplicates.
func AddTagsToGoal(goalID int64, tagsToAdd []string) error {
	var currentTagsJSON sql.NullString
	err := DB.QueryRow("SELECT tags FROM goals WHERE id = ?", goalID).Scan(&currentTagsJSON)
	if err != nil {
		return err
	}

	currentTags := util.JSONToTags(currentTagsJSON.String)
	tagSet := make(map[string]bool)
	for _, t := range currentTags {
		tagSet[t] = true
	}
	for _, t := range tagsToAdd {
		tagSet[strings.ToLower(t)] = true
	}

	var newTags []string
	for t := range tagSet {
		newTags = append(newTags, t)
	}

	newTagsJSON := util.TagsToJSON(newTags)
	_, err = DB.Exec("UPDATE goals SET tags = ? WHERE id = ?", newTagsJSON, goalID)
	return err
}

// SetGoalTags replaces all tags for a goal.
func SetGoalTags(goalID int64, tags []string) error {
	tagSet := make(map[string]bool)
	for _, t := range tags {
		t = strings.TrimSpace(strings.ToLower(strings.TrimPrefix(t, "#")))
		if t != "" {
			tagSet[t] = true
		}
	}
	var out []string
	for t := range tagSet {
		out = append(out, t)
	}
	newTagsJSON := util.TagsToJSON(out)
	_, err := DB.Exec("UPDATE goals SET tags = ? WHERE id = ?", newTagsJSON, goalID)
	return err
}

// Search performs a dynamic search on goals based on structured query filters and workspace isolation.
func Search(query util.SearchQuery, workspaceID int64) ([]models.Goal, error) {
	var args []interface{}
	sql := `SELECT id, parent_id, sprint_id, description, status, rank, priority, effort, tags, recurrence_rule, created_at, archived_at 
	        FROM goals WHERE workspace_id = ?`
	args = append(args, workspaceID)

	if len(query.Tags) > 0 {
		for _, tag := range query.Tags {
			sql += ` AND EXISTS (SELECT 1 FROM json_each(COALESCE(tags, '[]')) WHERE value = ?)`
			args = append(args, tag)
		}
	}

	if len(query.Status) > 0 {
		placeholders := strings.Repeat(",?", len(query.Status)-1)
		sql += fmt.Sprintf(` AND status IN (?%s)`, placeholders)
		for _, s := range query.Status {
			args = append(args, s)
		}
	}

	if len(query.Text) > 0 {
		for _, text := range query.Text {
			sql += ` AND description LIKE ?`
			args = append(args, "%"+text+"%")
		}
	}

	sql += " ORDER BY created_at DESC LIMIT 50"

	return scanGoals(sql, args...)
}

func scanGoals(query string, args ...interface{}) ([]models.Goal, error) {
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		if err := rows.Scan(&g.ID, &g.ParentID, &g.SprintID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.Tags, &g.RecurrenceRule, &g.CreatedAt, &g.ArchivedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

func GetAllGoals() ([]models.Goal, error) {
	return scanGoals(`
		SELECT id, parent_id, sprint_id, description, status, rank, priority, effort, tags, recurrence_rule, created_at, archived_at 
		FROM goals 
		ORDER BY rank ASC, created_at ASC`)
}

func MovePendingToBacklog(sprintID int64) error {
	_, err := DB.Exec("UPDATE goals SET sprint_id = NULL WHERE sprint_id = ? AND status != 'completed'", sprintID)
	return err
}

// Archived goals
func GetArchivedGoals(workspaceID int64) ([]models.Goal, error) {
	rows, err := DB.Query(`
		SELECT id, parent_id, sprint_id, description, status, rank, priority, effort, tags, recurrence_rule, created_at, archived_at 
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
		if err := rows.Scan(&g.ID, &g.ParentID, &g.SprintID, &g.Description, &g.Status, &g.Rank, &g.Priority, &g.Effort, &g.Tags, &g.RecurrenceRule, &g.CreatedAt, &g.ArchivedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

func ArchiveGoal(goalID int64) error {
	_, err := DB.Exec("UPDATE goals SET status = 'archived', archived_at = CURRENT_TIMESTAMP WHERE id = ?", goalID)
	return err
}

func UnarchiveGoal(goalID int64) error {
	_, err := DB.Exec("UPDATE goals SET status = 'pending', archived_at = NULL, sprint_id = NULL WHERE id = ?", goalID)
	return err
}

// --- Journaling ---

func AddJournalEntry(dayID int64, workspaceID int64, sprintID sql.NullInt64, goalID sql.NullInt64, content string) error {
	_, err := DB.Exec("INSERT INTO journal_entries (day_id, workspace_id, sprint_id, goal_id, content) VALUES (?, ?, ?, ?, ?)", dayID, workspaceID, sprintID, goalID, content)
	return err
}

func GetJournalEntries(dayID int64, workspaceID int64) ([]models.JournalEntry, error) {
	rows, err := DB.Query(`
		SELECT id, day_id, workspace_id, sprint_id, goal_id, content, created_at 
		FROM journal_entries 
		WHERE day_id = ? AND workspace_id = ?
		ORDER BY created_at ASC`, dayID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.JournalEntry
	for rows.Next() {
		var e models.JournalEntry
		if err := rows.Scan(&e.ID, &e.DayID, &e.WorkspaceID, &e.SprintID, &e.GoalID, &e.Content, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}
