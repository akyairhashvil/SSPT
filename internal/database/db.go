// Package database provides SQLite-backed persistence for SSPT.
// It handles database creation, migrations, encryption via SQLCipher,
// and all CRUD operations for workspaces, days, sprints, and goals.
//
// The package uses a single Database struct that wraps *sql.DB and
// provides transaction support through helper methods.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/akyairhashvil/SSPT/internal/util"
	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	DB              *sql.DB
	dbFile          string
	cipherAvailable bool
	dbEncrypted     bool
	cipherVersion   string
}

var (
	ErrSQLCipherUnavailable = errors.New("sqlcipher is unavailable")
)

const defaultDBTimeout = 5 * time.Second

// Open initializes the database connection and schema.
func Open(ctx context.Context, filepath, key string) (*Database, error) {
	return NewDatabase(ctx, filepath, key)
}

func NewDatabase(ctx context.Context, filepath, key string) (*Database, error) {
	d := &Database{dbFile: filepath}
	db, err := d.openDB(ctx, filepath, key)
	if err != nil {
		return nil, err
	}
	d.DB = db
	d.cipherAvailable, d.cipherVersion = d.detectSQLCipher(ctx)
	if sqlcipherCompiled() {
		d.cipherAvailable = true
	}
	if err := d.verifyDB(ctx, key); err != nil {
		return nil, err
	}
	if err := d.createTables(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

func NewTestDatabase(ctx context.Context) (*Database, error) {
	return NewDatabase(ctx, ":memory:", "")
}

func (d *Database) Close() error {
	if d == nil || d.DB == nil {
		return nil
	}
	return d.DB.Close()
}

func (d *Database) withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, timeout)
}

func (d *Database) createTables(ctx context.Context) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
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
			task_started_at DATETIME,
			task_elapsed_seconds INTEGER DEFAULT 0,
			task_active INTEGER DEFAULT 0,
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
		_, err := d.DB.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("create tables: %w (%s)", err, query)
		}
	}

	// Migrations for existing databases
	if err := d.migrate(ctx); err != nil {
		return err
	}
	return nil
}

func (d *Database) migrate(ctx context.Context) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	migrations := []string{
		// Add last_paused_at to sprints
		"ALTER TABLE sprints ADD COLUMN last_paused_at DATETIME",
		// Add elapsed_seconds to sprints
		"ALTER TABLE sprints ADD COLUMN elapsed_seconds INTEGER DEFAULT 0",
		// Add rank to goals
		"ALTER TABLE goals ADD COLUMN rank INTEGER DEFAULT 0",

		// V3 Migrations
		// Goals
		"ALTER TABLE goals ADD COLUMN parent_id INTEGER",
		"ALTER TABLE goals ADD COLUMN workspace_id INTEGER",
		"ALTER TABLE goals ADD COLUMN notes TEXT",
		"ALTER TABLE goals ADD COLUMN priority INTEGER DEFAULT 3",
		"ALTER TABLE goals ADD COLUMN recurrence_rule TEXT",
		"ALTER TABLE goals ADD COLUMN archived_at DATETIME",
		"ALTER TABLE goals ADD COLUMN effort TEXT DEFAULT 'M'",
		"ALTER TABLE goals ADD COLUMN tags TEXT",
		"ALTER TABLE goals ADD COLUMN links TEXT",
		"ALTER TABLE goals ADD COLUMN task_started_at DATETIME",
		"ALTER TABLE goals ADD COLUMN task_elapsed_seconds INTEGER DEFAULT 0",
		"ALTER TABLE goals ADD COLUMN task_active INTEGER DEFAULT 0",

		// Sprints
		"ALTER TABLE sprints ADD COLUMN workspace_id INTEGER",
		// Backfill legacy sprints to default workspace (1)
		"UPDATE sprints SET workspace_id = (SELECT id FROM workspaces WHERE slug = 'personal') WHERE workspace_id IS NULL",

		// Journal
		"ALTER TABLE journal_entries ADD COLUMN goal_id INTEGER",
		"ALTER TABLE journal_entries ADD COLUMN tags TEXT",
		"ALTER TABLE journal_entries ADD COLUMN workspace_id INTEGER",
		// Backfill legacy journal entries
		"UPDATE journal_entries SET workspace_id = (SELECT id FROM workspaces WHERE slug = 'personal') WHERE workspace_id IS NULL",

		// Backfill legacy goals
		"UPDATE goals SET workspace_id = (SELECT id FROM workspaces WHERE slug = 'personal') WHERE workspace_id IS NULL",

		// Workspaces
		"ALTER TABLE workspaces ADD COLUMN view_mode INTEGER DEFAULT 0",
		"ALTER TABLE workspaces ADD COLUMN theme TEXT DEFAULT 'default'",
		"ALTER TABLE workspaces ADD COLUMN show_backlog INTEGER DEFAULT 1",
		"ALTER TABLE workspaces ADD COLUMN show_completed INTEGER DEFAULT 1",
		"ALTER TABLE workspaces ADD COLUMN show_archived INTEGER DEFAULT 0",

		// Task dependencies
		`CREATE TABLE IF NOT EXISTS task_deps (
		goal_id INTEGER NOT NULL,
		depends_on_id INTEGER NOT NULL,
		PRIMARY KEY (goal_id, depends_on_id),
		FOREIGN KEY(goal_id) REFERENCES goals(id),
		FOREIGN KEY(depends_on_id) REFERENCES goals(id)
	);`,

		// Settings
		`CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);`,
	}

	for _, query := range migrations {
		if _, err := d.DB.ExecContext(ctx, query); err != nil {
			if isIgnorableMigrationErr(err) {
				continue
			}
			return fmt.Errorf("migration failed: %w (%s)", err, query)
		}
	}

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
		if _, err := d.DB.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("create index: %w (%s)", err, stmt)
		}
	}
	return nil
}

func isIgnorableMigrationErr(err error) bool {
	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "duplicate column")
}

func rollbackWithLog(tx *sql.Tx, originalErr error) error {
	if rbErr := tx.Rollback(); rbErr != nil {
		util.LogError("rollback failed", fmt.Errorf("rollback error: %w (original: %v)", rbErr, originalErr))
	}
	return originalErr
}

// WithTx executes fn in a transaction and commits on success.
func (d *Database) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := fn(tx); err != nil {
		return rollbackWithLog(tx, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func logCleanupError(context string, err error) {
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		util.LogError(context, err)
	}
}

func (d *Database) RekeyDB(ctx context.Context, key string) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	if key != "" {
		if _, err := d.DB.ExecContext(ctx, "PRAGMA key = ''"); err != nil {
			util.LogError("reset pragma key failed", err)
		}
	}
	escapedKey, err := pragmaValue(key)
	if err != nil {
		return err
	}
	_, err = d.DB.ExecContext(ctx, fmt.Sprintf("PRAGMA rekey = '%s'", escapedKey))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such pragma") {
			return ErrSQLCipherUnavailable
		}
		return err
	}
	d.cipherAvailable, d.cipherVersion = d.detectSQLCipher(ctx)
	if sqlcipherCompiled() {
		d.cipherAvailable = true
	}
	if key != "" && d.dbFile != "" {
		enc, encErr := isEncryptedFile(ctx, d.dbFile)
		if encErr != nil {
			return encErr
		}
		if !enc {
			return fmt.Errorf("rekey did not encrypt database")
		}
	}
	d.dbEncrypted = key != ""
	return nil
}

func (d *Database) EncryptDatabase(ctx context.Context, key string) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	if key == "" {
		return fmt.Errorf("passphrase required")
	}
	if d.dbFile == "" {
		return fmt.Errorf("database path unavailable")
	}
	if !sqlcipherCompiled() {
		return ErrSQLCipherUnavailable
	}
	if err := d.DB.Close(); err != nil {
		return err
	}

	tempPath := d.dbFile + ".enc"
	backupPath := d.dbFile + ".bak"
	defer func() {
		logCleanupError("cleanup temp db", os.Remove(tempPath))
		logCleanupError("cleanup backup db", os.Remove(backupPath))
	}()
	logCleanupError("cleanup temp db", os.Remove(tempPath))
	logCleanupError("cleanup backup db", os.Remove(backupPath))

	// Open plaintext DB without a key.
	plainDB, err := sql.Open("sqlite3", d.dbFile)
	if err != nil {
		return err
	}

	if err := plainDB.PingContext(ctx); err != nil {
		logCleanupError("plaintext close failed", plainDB.Close())
		return fmt.Errorf("plaintext ping failed: %w", err)
	}
	var count int
	if err := plainDB.QueryRowContext(ctx, "SELECT COUNT(1) FROM sqlite_master").Scan(&count); err != nil {
		logCleanupError("plaintext close failed", plainDB.Close())
		return fmt.Errorf("plaintext check failed: %w", err)
	}

	if _, err := plainDB.ExecContext(ctx, "ATTACH DATABASE ':memory:' AS probe"); err == nil {
		if _, detErr := plainDB.ExecContext(ctx, "DETACH DATABASE probe"); detErr != nil {
			util.LogError("detach probe failed", detErr)
		}
	}
	attach := fmt.Sprintf("ATTACH DATABASE '%s' AS enc KEY '%s'", escapeSQLiteString(tempPath), escapeSQLiteString(key))
	if _, err := plainDB.ExecContext(ctx, attach); err != nil {
		logCleanupError("plaintext close failed", plainDB.Close())
		return fmt.Errorf("attach encrypted failed: %w", err)
	}
	if _, err := plainDB.ExecContext(ctx, "SELECT sqlcipher_export('enc')"); err != nil {
		if _, detErr := plainDB.ExecContext(ctx, "DETACH DATABASE enc"); detErr != nil {
			util.LogError("detach encrypted failed", detErr)
		}
		logCleanupError("plaintext close failed", plainDB.Close())
		return fmt.Errorf("sqlcipher_export failed: %w", err)
	}
	if _, err := plainDB.ExecContext(ctx, "DETACH DATABASE enc"); err != nil {
		logCleanupError("plaintext close failed", plainDB.Close())
		return fmt.Errorf("detach encrypted failed: %w", err)
	}
	if err := plainDB.Close(); err != nil {
		return fmt.Errorf("plaintext close failed: %w", err)
	}

	if enc, encErr := isEncryptedFile(ctx, tempPath); encErr == nil && !enc {
		return fmt.Errorf("export produced plaintext (encryption not applied)")
	} else if encErr != nil {
		return fmt.Errorf("encrypted probe failed: %w", encErr)
	}

	encDB, err := d.openDB(ctx, tempPath, key)
	if err != nil {
		return fmt.Errorf("encrypted open failed: %w", err)
	}
	if _, err := encDB.ExecContext(ctx, "PRAGMA cipher_migrate"); err != nil {
		logCleanupError("encrypted close failed", encDB.Close())
		return fmt.Errorf("encrypted migrate failed: %w", err)
	}
	if err := encDB.QueryRowContext(ctx, "SELECT COUNT(1) FROM sqlite_master").Scan(&count); err != nil {
		logCleanupError("encrypted close failed", encDB.Close())
		return fmt.Errorf("encrypted verify failed: %w", err)
	}
	if err := encDB.Close(); err != nil {
		return fmt.Errorf("encrypted close failed: %w", err)
	}

	if err := os.Rename(d.dbFile, backupPath); err != nil {
		return fmt.Errorf("backup rename failed: %w", err)
	}
	if err := os.Rename(tempPath, d.dbFile); err != nil {
		logCleanupError("restore backup db", os.Rename(backupPath, d.dbFile))
		logCleanupError("cleanup temp db", os.Remove(tempPath))
		return fmt.Errorf("encrypted rename failed: %w", err)
	}
	if err := d.reopenEncrypted(ctx, key); err != nil {
		logCleanupError("preserve failed encrypted db", os.Rename(d.dbFile, tempPath))
		logCleanupError("restore backup db", os.Rename(backupPath, d.dbFile))
		logCleanupError("cleanup temp db", os.Remove(tempPath))
		return fmt.Errorf("reopen encrypted failed: %w", err)
	}
	logCleanupError("cleanup backup db", os.Remove(backupPath))
	logCleanupError("cleanup temp db", os.Remove(tempPath))
	return nil
}

func (d *Database) reopenEncrypted(ctx context.Context, key string) error {
	db, err := d.openDB(ctx, d.dbFile, key)
	if err != nil {
		return err
	}
	d.DB = db
	d.cipherAvailable, d.cipherVersion = d.detectSQLCipher(ctx)
	if sqlcipherCompiled() {
		d.cipherAvailable = true
	}
	if err := d.verifyDB(ctx, key); err != nil {
		return err
	}
	if err := d.createTables(ctx); err != nil {
		return err
	}
	return nil
}

func (d *Database) openDB(ctx context.Context, filepath, key string) (*sql.DB, error) {
	if ctx == nil {
		ctx = context.Background()
	}
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
	if err := db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			util.LogError("close db after ping failed", closeErr)
		}
		if msg := strings.ToLower(err.Error()); strings.Contains(msg, "file is encrypted") || strings.Contains(msg, "not a database") {
			if key == "" {
				return nil, ErrDatabaseEncrypted
			}
			return nil, ErrWrongPassphrase
		}
		return nil, err
	}
	return db, nil
}

func (d *Database) verifyDB(ctx context.Context, key string) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var count int
	err := d.DB.QueryRowContext(ctx, "SELECT COUNT(1) FROM sqlite_master").Scan(&count)
	if err == nil {
		if key != "" {
			d.dbEncrypted = true
		} else if d.dbFile != "" {
			if enc, encErr := isEncryptedFile(ctx, d.dbFile); encErr == nil {
				d.dbEncrypted = enc
			}
		}
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "file is encrypted") || strings.Contains(msg, "not a database") {
		enc := false
		if d.dbFile != "" {
			if encVal, encErr := isEncryptedFile(ctx, d.dbFile); encErr == nil {
				enc = encVal
			}
		}
		if key == "" {
			if enc {
				d.dbEncrypted = true
				return ErrDatabaseEncrypted
			}
			return ErrDatabaseCorrupted
		}
		if !enc && d.dbFile != "" {
			return ErrDatabaseCorrupted
		}
		return ErrWrongPassphrase
	}
	return err
}

func (d *Database) detectSQLCipher(ctx context.Context) (bool, string) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var version string
	err := d.DB.QueryRowContext(ctx, "PRAGMA cipher_version").Scan(&version)
	if err == nil {
		version = strings.TrimSpace(version)
		if version != "" {
			return true, version
		}
	}
	rows, err := d.DB.QueryContext(ctx, "PRAGMA compile_options")
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

type EncryptionInfo struct {
	CipherAvailable   bool
	DatabaseEncrypted bool
	CipherVersion     string
}

func (d *Database) EncryptionStatus() EncryptionInfo {
	return EncryptionInfo{
		CipherAvailable:   d.cipherAvailable,
		DatabaseEncrypted: d.dbEncrypted,
		CipherVersion:     d.cipherVersion,
	}
}

// escapeSQLiteString escapes single quotes for use in PRAGMA commands.
// IMPORTANT: This is ONLY safe for PRAGMA commands which do not support
// parameterized queries. NEVER use this for regular SQL queries.
// All user data queries MUST use parameterized queries (?, $1, etc.).
func escapeSQLiteString(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

// pragmaValue validates and escapes a value for use in PRAGMA commands.
// Returns error if value contains unexpected characters.
func pragmaValue(value string) (string, error) {
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != ' ' && r != '-' && r != '_' && r != '.' {
			return "", fmt.Errorf("invalid character in PRAGMA value: %q", r)
		}
	}
	return escapeSQLiteString(value), nil
}

func isEncryptedFile(ctx context.Context, path string) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	var count int
	err = conn.QueryRowContext(ctx, "SELECT COUNT(1) FROM sqlite_master").Scan(&count)
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
func IsEncryptedFile(ctx context.Context, path string) (bool, error) {
	return isEncryptedFile(ctx, path)
}

func (d *Database) DatabaseHasData(ctx context.Context) bool {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var count int
	if err := d.DB.QueryRowContext(ctx, "SELECT COUNT(1) FROM goals").Scan(&count); err == nil && count > 0 {
		return true
	}
	if err := d.DB.QueryRowContext(ctx, "SELECT COUNT(1) FROM sprints").Scan(&count); err == nil && count > 0 {
		return true
	}
	if err := d.DB.QueryRowContext(ctx, "SELECT COUNT(1) FROM journal_entries").Scan(&count); err == nil && count > 0 {
		return true
	}
	return false
}

func (d *Database) RecreateEncryptedDatabase(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("passphrase required")
	}
	if d.dbFile == "" {
		return fmt.Errorf("database path unavailable")
	}
	if !sqlcipherCompiled() {
		return ErrSQLCipherUnavailable
	}
	backupPath := d.dbFile + ".bak"
	defer func() {
		logCleanupError("cleanup backup db", os.Remove(backupPath))
		logCleanupError("cleanup encrypted temp db", os.Remove(d.dbFile+".enc"))
	}()
	logCleanupError("cleanup backup db", os.Remove(backupPath))
	if err := d.DB.Close(); err != nil {
		return err
	}
	if err := os.Rename(d.dbFile, backupPath); err != nil {
		return fmt.Errorf("backup rename failed: %w", err)
	}
	if err := d.reopenEncrypted(ctx, key); err != nil {
		logCleanupError("cleanup failed db", os.Remove(d.dbFile))
		logCleanupError("restore backup db", os.Rename(backupPath, d.dbFile))
		return fmt.Errorf("recreate encrypted failed: %w", err)
	}
	logCleanupError("cleanup backup db", os.Remove(backupPath))
	return nil
}

func (d *Database) ClearDatabase(ctx context.Context) error {
	if d.dbFile == "" {
		return fmt.Errorf("database path unavailable")
	}
	if d.DB != nil {
		if err := d.DB.Close(); err != nil {
			return err
		}
	}
	logCleanupError("cleanup db", os.Remove(d.dbFile))
	logCleanupError("cleanup backup db", os.Remove(d.dbFile+".bak"))
	logCleanupError("cleanup encrypted db", os.Remove(d.dbFile+".enc"))
	return d.reopenEncrypted(ctx, "")
}
