package main

import (
	"fmt"
	"os"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// 1. Initialize Database
	// We use a local file 'sprints.db' in the current directory for now.
	database.InitDB("sprints.db")
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
