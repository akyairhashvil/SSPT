package tui

import (
	"fmt"
	"path/filepath"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/go-pdf/fpdf"
)

func GeneratePDFReport(dayID int64) {
	day, _ := database.GetDay(dayID)
	sprints, _ := database.GetSprints(dayID)

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, fmt.Sprintf("Productivity Report: %s", day.Date))
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)

	totalCompleted := 0

	for _, s := range sprints {
		goals, _ := database.GetGoalsForSprint(s.ID)
		rootGoals := BuildHierarchy(goals)
		flatGoals := Flatten(rootGoals, 0, nil)

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
			pdf.Cell(0, 8, fmt.Sprintf("%s  %s %s", indent, status, g.Description))
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
	entries, _ := database.GetJournalEntries(dayID)
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

	filename := fmt.Sprintf("report_%s.pdf", day.Date)
	err := pdf.OutputFileAndClose(filename)

	absPath, _ := filepath.Abs(filename)
	if err == nil {
		fmt.Printf("\nPDF Report generated: %s\n", absPath)
	} else {
		fmt.Printf("\nError generating PDF: %v\n", err)
	}
}
