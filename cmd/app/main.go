package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

func main() {
	// 1. Initialize Database
	// We use a local file 'sprints.db' in the current directory for now.
	dbPath := "sprints.db"
	key := strings.TrimSpace(os.Getenv("SSPT_DB_KEY"))
	_, statErr := os.Stat(dbPath)
	dbExists := statErr == nil
	if !dbExists && key == "" && database.SQLCipherCompiled() {
		pass, perr := promptForKey("Set DB passphrase (leave empty to skip): ")
		if perr != nil {
			fmt.Printf("Alas, there's been an error: %v\n", perr)
			os.Exit(1)
		}
		if pass != "" {
			key = pass
		}
	}
	err := database.InitDB(dbPath, key)
	if key != "" && err != nil {
		errText := strings.ToLower(err.Error())
		if strings.Contains(errText, "file is not a database") || strings.Contains(errText, "file is encrypted") {
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
		errText := strings.ToLower(err.Error())
		if err == database.ErrEncrypted || strings.Contains(errText, "file is not a database") || strings.Contains(errText, "file is encrypted") {
			err = database.ErrEncrypted
		}
	}
	if err == database.ErrEncrypted && key == "" {
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
			if err == nil {
				break
			}
		}
	}
	if !dbExists && err == nil && key == "" {
		fmt.Println("No DB passphrase provided. You can set one inside the app.")
	}
	if err != nil {
		if err == database.ErrSQLCipherUnavailable {
			fmt.Println("SQLCipher support is unavailable in this build. Rebuild with SQLCipher to enable encryption.")
		} else {
			fmt.Printf("Alas, there's been an error: %v\n", err)
		}
		os.Exit(1)
	}
	if !dbExists && key != "" {
		_ = database.SetSetting("passphrase_hash", hashPassphrase(key))
	}
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

func hashPassphrase(pass string) string {
	sum := sha256.Sum256([]byte(pass))
	return hex.EncodeToString(sum[:])
}
