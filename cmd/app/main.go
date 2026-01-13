package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/akyairhashvil/SSPT/internal/config"
	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/tui"
	"github.com/akyairhashvil/SSPT/internal/util"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

func main() {
	ctx := context.Background()
	// 1. Initialize Database
	dbRoot := util.DataDir(config.AppName)
	_ = os.MkdirAll(dbRoot, 0o755)
	dbPath := filepath.Join(dbRoot, config.DBFileName)
	cleanupStaleDBArtifacts(dbPath)
	key := strings.TrimSpace(os.Getenv("SSPT_DB_KEY"))
	if key != "" {
		fmt.Fprintln(os.Stderr, "Warning: passphrase set via environment variable is visible in process listing")
	}
	_, statErr := os.Stat(dbPath)
	dbExists := statErr == nil
	if !dbExists && key == "" && database.SQLCipherCompiled() {
		for {
			pass, perr := promptForKey("Set DB passphrase (leave empty to skip): ")
			if perr != nil {
				fmt.Printf("Alas, there's been an error: %v\n", perr)
				os.Exit(1)
			}
			if pass == "" {
				break
			}
			if err := util.ValidatePassphrase(pass); err != nil {
				fmt.Printf("Passphrase too weak: %v\n", err)
				pass = ""
				continue
			}
			key = pass
			pass = ""
			break
		}
	}
	db, err := database.Open(ctx, dbPath, key)
	if key != "" && err != nil {
		if errors.Is(err, database.ErrWrongPassphrase) || errors.Is(err, database.ErrDatabaseEncrypted) || errors.Is(err, database.ErrDatabaseCorrupted) {
			enc, encErr := database.IsEncryptedFile(ctx, dbPath)
			if encErr == nil && !enc {
				if db != nil {
					_ = db.Close()
					db = nil
				}
				if initDB, initErr := database.Open(ctx, dbPath, ""); initErr == nil {
					if encErr := initDB.EncryptDatabase(ctx, key); encErr == nil {
						db = initDB
						err = nil
					} else {
						err = encErr
					}
				} else {
					err = initErr
				}
			}
		}
	}
	if key == "" && err != nil {
		if errors.Is(err, database.ErrDatabaseEncrypted) {
			err = database.ErrDatabaseEncrypted
		}
	}
	if errors.Is(err, database.ErrDatabaseEncrypted) && key == "" {
		for tries := 0; tries < 3; tries++ {
			pass, perr := promptForKey("Enter DB passphrase: ")
			if perr != nil {
				fmt.Printf("Alas, there's been an error: %v\n", perr)
				os.Exit(1)
			}
			if pass == "" {
				fmt.Println("Empty passphrase. Exiting.")
				os.Exit(1)
			}
			if db != nil {
				_ = db.Close()
				db = nil
			}
			db, err = database.Open(ctx, dbPath, pass)
			pass = ""
			if err == nil {
				break
			}
		}
	}
	if !dbExists && err == nil && key == "" {
		fmt.Println("No DB passphrase provided. You can set one inside the app.")
	}
	if err != nil {
		var opErr *database.OpError
		if errors.As(err, &opErr) {
			err = opErr
		}
		if errors.Is(err, database.ErrSQLCipherUnavailable) {
			fmt.Println("SQLCipher support is unavailable in this build. Rebuild with SQLCipher to enable encryption.")
		} else {
			fmt.Printf("Alas, there's been an error: %v\n", err)
		}
		os.Exit(1)
	}
	if !dbExists && key != "" {
		_ = db.SetSetting(ctx, "passphrase_hash", util.HashPassphrase(key))
	}
	key = ""
	defer db.Close()

	// 2. Initialize the Main Model
	// We pass the DB connection, though it's also available via the global in database package
	// Passing it explicitly is often cleaner for testing.
	model := tui.NewMainModel(ctx, db)

	// 3. Enable Mouse Support & Start Program
	p := tea.NewProgram(model, tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

func promptForKey(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	pass, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return strings.TrimSpace(string(pass)), err
}

func cleanupStaleDBArtifacts(dbPath string) {
	cleanupFiles(dbPath+".enc", dbPath+".bak")
}

func cleanupFiles(paths ...string) {
	for _, p := range paths {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			util.LogError("cleanup "+p, err)
		}
	}
}
