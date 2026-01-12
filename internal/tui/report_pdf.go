package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
	"github.com/go-pdf/fpdf"
)

func GeneratePDFReport(dayID int64, workspaceID int64) (string, error) {
	day, err := database.GetDay(dayID)
	if err != nil {
		return "", err
	}
	sprints, err := database.GetSprints(dayID, workspaceID)
	if err != nil {
		return "", err
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, fmt.Sprintf("Productivity Report: %s", day.Date))
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)

	totalCompleted := 0

	// Fetch ALL goals to build complete context
	allGoals, err := database.GetAllGoals()
	if err != nil {
		return "", err
	}
	masterTree := BuildHierarchy(allGoals)

	// Helper to check relevancy
	var isRelevant func(g models.Goal, sprintID int64) bool
	isRelevant = func(g models.Goal, sprintID int64) bool {
		if g.SprintID.Valid && g.SprintID.Int64 == sprintID {
			return true
		}
		for _, sub := range g.Subtasks {
			if isRelevant(sub, sprintID) {
				return true
			}
		}
		return false
	}

	formatElapsed := func(seconds int) string {
		if seconds <= 0 {
			return ""
		}
		h := seconds / 3600
		m := (seconds % 3600) / 60
		s := seconds % 60
		return fmt.Sprintf(" [time %02d:%02d:%02d]", h, m, s)
	}

	for _, s := range sprints {
		// Filter MasterTree for this sprint
		var relevantRoots []models.Goal
		for _, root := range masterTree {
			if isRelevant(root, s.ID) {
				relevantRoots = append(relevantRoots, root)
			}
		}
		flatGoals := Flatten(relevantRoots, 0, nil, 0)

		// Header
		pdf.SetFont("Arial", "B", 14)
		header := fmt.Sprintf("Sprint %d", s.SprintNumber)
		if s.Status == "completed" {
			header += " (Completed)"
		} else {
			header += " (" + s.Status + ")"
		}
		pdf.Cell(0, 10, header)
		pdf.Ln(8)

		// Goals
		pdf.SetFont("Arial", "", 12)
		if len(flatGoals) == 0 {
			pdf.Cell(0, 8, "  - No goals assigned.")
			pdf.Ln(8)
		}

		for _, g := range flatGoals {
			status := "[ ]"
			if g.Status == "completed" {
				status = "[x]"
				totalCompleted++
			}
			indent := ""
			for k := 0; k < g.Level; k++ {
				indent += "    "
			}
			pdf.Cell(0, 8, fmt.Sprintf("%s  %s %s%s", indent, status, g.Description, formatElapsed(g.TaskElapsedSec)))
			pdf.Ln(6)
		}
		pdf.Ln(4)
	}

	// Summary
	pdf.Ln(10)
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 10, fmt.Sprintf("Total Goals Completed: %d", totalCompleted))
	pdf.Ln(10)

	// Journaling
	entries, err := database.GetJournalEntries(dayID, workspaceID)
	if err != nil {
		return "", err
	}
	if len(entries) > 0 {
		pdf.SetFont("Arial", "B", 14)
		pdf.Cell(0, 10, "Journal")
		pdf.Ln(8)
		pdf.SetFont("Arial", "", 12)
		for _, e := range entries {
			timeStr := e.CreatedAt.Format("15:04")
			content := fmt.Sprintf("[%s] %s", timeStr, e.Content)
			pdf.MultiCell(0, 8, content, "", "", false)
		}
	}

	reportRoot := util.ReportsDir("sspt")
	if err := os.MkdirAll(reportRoot, 0o755); err != nil {
		return "", err
	}
	filename := filepath.Join(reportRoot, fmt.Sprintf("report_%s.pdf", day.Date))
	if err := pdf.OutputFileAndClose(filename); err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}
	return absPath, nil
}
