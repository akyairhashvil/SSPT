package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/tui"
	"github.com/akyairhashvil/SSPT/internal/util"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

func main() {
	// 1. Initialize Database
	dbRoot := util.DataDir("sspt")
	_ = os.MkdirAll(dbRoot, 0o755)
	dbPath := filepath.Join(dbRoot, "sprints.db")
	cleanupStaleDBArtifacts(dbPath)
	key := strings.TrimSpace(os.Getenv("SSPT_DB_KEY"))
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
	err := database.InitDB(dbPath, key)
	if key != "" && err != nil {
		if errors.Is(err, database.ErrWrongPassphrase) || errors.Is(err, database.ErrDatabaseEncrypted) || errors.Is(err, database.ErrDatabaseCorrupted) {
			enc, encErr := database.IsEncryptedFile(dbPath)
			if encErr == nil && !enc {
				if database.DB != nil {
					_ = database.DB.Close()
				}
				if initErr := database.InitDB(dbPath, ""); initErr == nil {
					if encErr := database.EncryptDatabase(key); encErr == nil {
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
			if database.DB != nil {
				_ = database.DB.Close()
			}
			err = database.InitDB(dbPath, pass)
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
		if errors.Is(err, database.ErrSQLCipherUnavailable) {
			fmt.Println("SQLCipher support is unavailable in this build. Rebuild with SQLCipher to enable encryption.")
		} else {
			fmt.Printf("Alas, there's been an error: %v\n", err)
		}
		os.Exit(1)
	}
	if !dbExists && key != "" {
		_ = database.SetSetting("passphrase_hash", util.HashPassphrase(key))
	}
	key = ""
	defer database.DB.Close()

	// 2. Initialize the Main Model
	// We pass the DB connection, though it's also available via the global in database package
	// Passing it explicitly is often cleaner for testing.
	model := tui.NewMainModel(database.DB)

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
	_ = os.Remove(dbPath + ".enc")
	_ = os.Remove(dbPath + ".bak")
}
