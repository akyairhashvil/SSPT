package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func renderLogo() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render("S") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true).Render("S") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true).Render("P") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true).Render("T")
}

func taskElapsed(goal models.Goal) time.Duration {
	seconds := goal.TaskElapsedSec
	if goal.TaskActive && goal.TaskStartedAt != nil {
		seconds += int(time.Since(*goal.TaskStartedAt).Seconds())
	}
	if seconds < 0 {
		seconds = 0
	}
	return time.Duration(seconds) * time.Second
}

func truncateLabel(text string, max int) string {
	if max <= 0 {
		return ""
	}
	if ansi.StringWidth(text) <= max {
		return text
	}
	return ansi.Truncate(text, max, "…")
}

func (m DashboardModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	if m.security.lock.Locked {
		return m.renderLockScreen()
	}

	if m.err != nil {
		return fmt.Sprintf("\nError: %v\n\nPress any key to continue.", m.err)
	}

	timerBox := m.renderHeader()
	footer := m.renderFooter()
	journalPane := m.renderJournalPane()

	footerGap := 0
	if footer != "" {
		footerGap = 1
	}
	headerLines := splitLines(timerBox)
	footerSplit := splitLines(footer)
	availableLines := m.height - len(headerLines) - len(footerSplit) - footerGap
	if availableLines < 0 {
		availableLines = 0
	}
	minBoardHeight := 3
	if journalPane != "" {
		journalCap := availableLines - minBoardHeight
		if journalCap < 0 {
			journalCap = 0
		}
		journalPane = trimLines(journalPane, journalCap)
	}
	journalHeight := len(splitLines(journalPane))
	columnHeight := availableLines - journalHeight
	if columnHeight < 0 {
		columnHeight = 0
	}

	layout := m.buildBoardLayout()

	// Assemble Final View
	boardHeight := columnHeight
	if footer != "" && boardHeight > 0 {
		boardHeight--
	}
	var board string
	if m.height > 0 {
		for boardHeight > 0 {
			board = strings.TrimRight(m.renderBoard(boardHeight, layout), "\n")
			boardLines := len(splitLines(board))
			journalLines := len(splitLines(journalPane))
			total := len(headerLines) + boardLines + journalLines + footerGap + len(footerSplit)
			if total <= m.height {
				break
			}
			boardHeight--
		}
	} else {
		board = m.renderBoard(boardHeight, layout)
	}
	if boardHeight == 0 {
		board = m.renderBoard(0, layout)
	}

	var lines []string
	lines = append(lines, headerLines...)
	if board != "" {
		lines = append(lines, splitLines(board)...)
	} else {
		lines = append(lines, "  (Window too small)")
	}
	if journalPane != "" {
		lines = append(lines, splitLines(journalPane)...)
	}
	if footer != "" && footerGap > 0 {
		lines = append(lines, "")
	}
	if footer != "" {
		lines = append(lines, footerSplit...)
	}
	if m.height > 0 {
		if len(lines) > m.height {
			lines = lines[:m.height]
		} else if len(lines) < m.height {
			lines = append(lines, make([]string, m.height-len(lines))...)
		}
	}
	return "\x1b[H\x1b[2J" + strings.Join(lines, "\n")
}

func containsTag(tags []string, target string) bool {
	for _, t := range tags {
		if t == target {
			return true
		}
	}
	return false
}

func tagIcon(tag string) (string, bool) {
	switch tag {
	case "urgent":
		return "⚡", true
	case "docs":
		return "📄", true
	case "blocked":
		return "⛔", true
	case "waiting":
		return "⏳", true
	case "bug":
		return "🐞", true
	case "idea":
		return "💡", true
	case "review":
		return "🔎", true
	case "focus":
		return "🎯", true
	case "later":
		return "💤", true
	default:
		return "", false
	}
}
