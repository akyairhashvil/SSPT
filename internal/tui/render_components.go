package tui

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/akyairhashvil/SSPT/internal/config"
	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type boardLayout struct {
	displayCount    int
	colFrame        lipgloss.Style
	colExtraHeight  int
	colContentWidth int
	visibleIndices  []int
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func trimLines(s string, max int) string {
	if max <= 0 || s == "" {
		return ""
	}
	lines := splitLines(s)
	if len(lines) <= max {
		return s
	}
	lines = lines[:max]
	return strings.Join(lines, "\n")
}

func (m DashboardModel) renderHeader() string {
	// Determine Timer Content
	var timerContent string
	var timerColor lipgloss.Style

	if m.timer.BreakActive {
		elapsed := time.Since(m.timer.BreakStart)
		rem := config.BreakDuration - elapsed
		if rem < 0 {
			rem = 0
		}
		timerContent = fmt.Sprintf("☕ BREAK TIME: %02d:%02d REMAINING", int(rem.Minutes()), int(rem.Seconds())%60)
		timerColor = m.theme.Break
	} else if m.timer.ActiveSprint != nil {
		startedAt := time.Now()
		if m.timer.ActiveSprint.StartTime != nil {
			startedAt = *m.timer.ActiveSprint.StartTime
		}
		elapsed := time.Since(startedAt) + (time.Duration(m.timer.ActiveSprint.ElapsedSeconds) * time.Second)
		rem := config.SprintDuration - elapsed
		if rem < 0 {
			rem = 0
		}
		timeStr := fmt.Sprintf("%02d:%02d", int(rem.Minutes()), int(rem.Seconds())%60)
		barView := m.progress.ViewAs(float64(elapsed) / float64(config.SprintDuration))
		timerContent = fmt.Sprintf("ACTIVE SPRINT: %d  |  %s  |  %s remaining", m.timer.ActiveSprint.SprintNumber, barView, timeStr)
		timerColor = m.theme.Focused
	} else {
		if len(m.workspaces) > 0 {
			// Safety index check
			idx := m.activeWorkspaceIdx
			if idx >= len(m.workspaces) {
				idx = 0
			}
			timerContent = fmt.Sprintf("[%s | %s] Select Sprint & Press 's' to Start", m.workspaces[idx].Name, m.day.Date)
		} else {
			timerContent = "No workspaces found."
		}
		timerColor = m.theme.Dim

		if m.view.focusedColIdx < len(m.sprints) {
			target := m.sprints[m.view.focusedColIdx]
			if target.Status == models.StatusPaused {
				elapsed := time.Duration(target.ElapsedSeconds) * time.Second
				rem := config.SprintDuration - elapsed
				timeStr := fmt.Sprintf("%02d:%02d", int(rem.Minutes()), int(rem.Seconds())%60)
				timerContent = fmt.Sprintf("PAUSED SPRINT: %d  |  %s remaining  |  [s] to Resume", target.SprintNumber, timeStr)
				timerColor = m.theme.Break
			}
		}
	}

	if timerContent == "" {
		timerContent = "SSPT - Ready"
		timerColor = m.theme.Dim
	}
	encInfo := m.db.EncryptionStatus()
	cipherOn, encrypted, cipherVer := encInfo.CipherAvailable, encInfo.DatabaseEncrypted, encInfo.CipherVersion
	dbLabel := "DB: sqlite"
	cipherLabel := "Cipher: none"
	if cipherOn {
		if cipherVer == "" {
			cipherLabel = "Cipher: unknown"
		} else {
			cipherLabel = "Cipher: " + cipherVer
		}
		if encrypted {
			dbLabel = "DB: enc"
		} else {
			dbLabel = "DB: plain"
		}
	} else if encrypted {
		dbLabel = "DB: enc"
	}
	logo := renderLogo()
	timerContent = fmt.Sprintf("%s  |  %s  |  %s  |  %s v%s", timerContent, dbLabel, cipherLabel, logo, versionLabel())

	// Render Header (Timer Box)
	headerFrame := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Border).
		Padding(0, 1)
	headerExtra := lipgloss.Width(headerFrame.Render(""))
	headerWidth := m.width - headerExtra
	if headerWidth < 1 {
		headerWidth = 1
	}
	return headerFrame.Width(headerWidth).Render(timerColor.Render(timerContent))
}

func (m DashboardModel) renderFooter() string {
	var footer string
	var footerContent string
	var footerHelpLines []string
	var rawFooter string
	hasStatusMessage := m.statusMessage != ""
	if hasStatusMessage {
		statusStyle := m.theme.Break.Foreground(lipgloss.Color("196"))
		if !m.statusIsError {
			statusStyle = m.theme.Break.Foreground(lipgloss.Color("208"))
		}
		footerContent = statusStyle.Render(m.statusMessage)
	} else if m.Message != "" {
		footerContent = m.theme.Break.Foreground(lipgloss.Color("208")).Render(m.Message)
	} else if m.modal.creatingGoal || m.modal.editingGoal || m.modal.creatingWorkspace || m.modal.initializingSprints {
		footerContent = m.theme.Input.Render(m.inputs.textInput.View())
	} else if m.modal.tagging {
		footerContent = m.theme.Dim.Render("[Tab] Toggle Tag | [Enter] Save | [Esc] Cancel")
	} else if m.modal.themePicking {
		footerContent = m.theme.Dim.Render("[Enter] Apply Theme | [Esc] Cancel")
	} else if m.modal.depPicking {
		footerContent = m.theme.Dim.Render("[Space] Toggle | [Enter] Save | [Esc] Cancel")
	} else if m.modal.settingRecurrence {
		footerContent = m.theme.Dim.Render("[Tab] Next | [Space] Toggle | [Enter] Save | [Esc] Cancel")
	} else if m.modal.confirmingDelete {
		footerContent = m.theme.Focused.Render("Delete task? [d] Delete | [a] Archive | [Esc] Cancel")
	} else if m.security.confirmingClearDB {
		var lines []string
		lines = append(lines, m.theme.Focused.Render("Clear database? This deletes all data."))
		if m.security.clearDBStatus != "" {
			lines = append(lines, m.theme.Break.Render(m.security.clearDBStatus))
		}
		if m.security.clearDBNeedsPass {
			lines = append(lines, m.theme.Dim.Render("Enter passphrase to confirm:"))
			lines = append(lines, m.theme.Focused.Render("> ")+m.security.lock.PassphraseInput.View())
		} else {
			lines = append(lines, m.theme.Dim.Render("[c] Clear | [Esc] Cancel"))
		}
		footerContent = lipgloss.JoinVertical(lipgloss.Left, lines...)
	} else if m.security.changingPassphrase {
		footerContent = m.theme.Dim.Render("[Enter] Next | [Esc] Cancel")
	} else if m.modal.journaling {
		// Only render journaling input in the journal pane, avoid duplicate
		footerContent = m.theme.Dim.Render("[Enter] to Save Log | [Esc] Cancel")
	} else if m.modal.movingGoal {
		footerContent = m.theme.Focused.Render("MOVE TO: [0] Backlog | [1-8] Sprint # | [Esc] Cancel")
	} else {
		baseHelp := "[n]New|[N]Sub|[e]Edit|[z]Toggle|[T]Task|[P]Priority|[+/-]Sprint|[w]Cycle|[W]New WS|[t]Tag|[m]Move|[D]Deps|[R]Repeat|[/]Search|[J]Journal|[I]Import|[G]Graph|[p]Passphrase|[d]Delete|[A]Archive|[u]Unarchive|[L]Lock|[C]Clear DB|[b]Backlog|[c]Completed|[a]Archived|[v]View|[Y]Theme"
		var timerHelp string
		if m.timer.ActiveSprint != nil {
			timerHelp = "|[s]PAUSE|[x]STOP"
		} else {
			timerHelp = "|[s]Start"
		}
		fullHelp := baseHelp + timerHelp + "|[ctrl+e]Export|[ctrl+r]Report|[q]Quit"
		rawFooter = fullHelp
		footerContent = m.theme.Dim.Render(fullHelp)
	}
	if footerContent != "" {
		boxed := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.Border).
			Padding(0, 1)
		innerWidth := m.width - lipgloss.Width(boxed.Render(""))
		if innerWidth < 1 {
			innerWidth = 1
		}
		content := footerContent
		if hasStatusMessage || m.Message != "" {
			content = lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, footerContent)
		} else if !m.modal.creatingGoal && !m.modal.editingGoal && !m.modal.creatingWorkspace && !m.modal.initializingSprints && !m.modal.tagging && !m.modal.themePicking && !m.modal.depPicking && !m.modal.settingRecurrence && !m.modal.confirmingDelete && !m.security.confirmingClearDB && !m.security.changingPassphrase {
			tokens := strings.Split(rawFooter, "|")
			const sep = " | "
			sepWidth := ansi.StringWidth(sep)
			var widths []int
			sumWidths := 0
			var lines []string
			var currentTokens []string
			currentWidth := 0
			for _, token := range tokens {
				token = strings.TrimSpace(token)
				if token == "" {
					continue
				}
				w := ansi.StringWidth(token)
				widths = append(widths, w)
				sumWidths += w
			}
			if len(widths) == 0 {
				content = ""
			} else {
				totalWidth := sumWidths + sepWidth*(len(widths)-1)
				linesTarget := int(math.Ceil(float64(totalWidth) / float64(innerWidth)))
				if linesTarget < 1 {
					linesTarget = 1
				}
				sumRemaining := sumWidths
				tokensRemaining := len(widths)
				linesRemaining := linesTarget
				idx := 0
				for _, token := range tokens {
					token = strings.TrimSpace(token)
					if token == "" {
						continue
					}
					tokenWidth := widths[idx]
					remainingTotal := sumRemaining + sepWidth*(tokensRemaining-1)
					idealMax := int(math.Ceil(float64(remainingTotal) / float64(linesRemaining)))
					if idealMax > innerWidth {
						idealMax = innerWidth
					}
					if currentWidth == 0 {
						currentTokens = append(currentTokens, token)
						currentWidth = tokenWidth
					} else {
						candidateWidth := currentWidth + sepWidth + tokenWidth
						if candidateWidth <= idealMax || linesRemaining == 1 {
							currentTokens = append(currentTokens, token)
							currentWidth = candidateWidth
						} else {
							lines = append(lines, strings.Join(currentTokens, sep))
							linesRemaining--
							currentTokens = []string{token}
							currentWidth = tokenWidth
						}
					}
					sumRemaining -= tokenWidth
					tokensRemaining--
					idx++
				}
				if len(currentTokens) > 0 {
					lines = append(lines, strings.Join(currentTokens, sep))
				}
				for _, line := range lines {
					footerHelpLines = append(footerHelpLines, lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, m.theme.Dim.Render(line)))
				}
				content = lipgloss.JoinVertical(lipgloss.Left, footerHelpLines...)
			}
		} else if !m.modal.confirmingDelete && !m.security.confirmingClearDB && !m.security.changingPassphrase && (m.modal.creatingGoal || m.modal.editingGoal || m.modal.creatingWorkspace || m.modal.initializingSprints || m.modal.tagging || m.modal.themePicking || m.modal.depPicking || m.modal.settingRecurrence) {
			content = footerContent
		} else if m.security.changingPassphrase {
			content = lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, footerContent)
		} else if m.modal.confirmingDelete {
			content = lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, footerContent)
		} else if m.security.confirmingClearDB {
			content = footerContent
		}
		footer = boxed.Width(innerWidth).Render(content)
	}

	return footer
}

func (m DashboardModel) buildBoardLayout() boardLayout {
	// Determine visible columns based on ViewMode
	var scrollableIndices []int
	showBacklog := true
	showCompleted := true
	showArchived := false
	if len(m.workspaces) > 0 && m.activeWorkspaceIdx < len(m.workspaces) {
		showBacklog = m.workspaces[m.activeWorkspaceIdx].ShowBacklog
		showCompleted = m.workspaces[m.activeWorkspaceIdx].ShowCompleted
		showArchived = m.workspaces[m.activeWorkspaceIdx].ShowArchived
	}
	for i := 0; i < len(m.sprints); i++ {
		sprint := m.sprints[i]
		if sprint.Status == models.StatusCompleted && sprint.SprintNumber > 0 {
			continue
		}
		if sprint.SprintNumber == -1 && (!showCompleted || m.viewMode == ViewModeFocused) {
			continue
		}
		if sprint.SprintNumber == 0 && (!showBacklog || m.viewMode == ViewModeMinimal) {
			continue
		}
		if sprint.SprintNumber == -2 && !showArchived {
			continue
		}
		if m.viewMode == ViewModeMinimal && sprint.SprintNumber < 0 {
			continue
		}
		scrollableIndices = append(scrollableIndices, i)
	}

	displayCount := config.MaxDisplayColumns
	if m.viewMode == ViewModeMinimal {
		displayCount = config.MinDisplayColumns
	}

	colFrame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Dim.GetForeground()).
		Padding(0, 1)
	colExtra := lipgloss.Width(colFrame.Render(""))
	// Dynamic Width Calculation
	availableWidth := m.width // Total available width across the terminal
	availablePerCol := availableWidth / displayCount
	colContentWidth := availablePerCol - colExtra
	if colContentWidth < 0 {
		colContentWidth = 0
	}
	colExtraHeight := lipgloss.Height(colFrame.Render(""))
	// Scroll Logic
	colScrollOffset := m.view.colScrollOffset
	if colScrollOffset > len(scrollableIndices)-displayCount {
		colScrollOffset = len(scrollableIndices) - displayCount
	}
	if colScrollOffset < 0 {
		colScrollOffset = 0
	}

	var visibleIndices []int
	for i := 0; i < displayCount; i++ {
		idx := colScrollOffset + i
		if idx < len(scrollableIndices) {
			visibleIndices = append(visibleIndices, scrollableIndices[idx])
		}
	}

	return boardLayout{
		displayCount:    displayCount,
		colFrame:        colFrame,
		colExtraHeight:  colExtraHeight,
		colContentWidth: colContentWidth,
		visibleIndices:  visibleIndices,
	}
}

func (m DashboardModel) renderBoard(height int, layout boardLayout) string {
	if height <= 0 {
		return ""
	}
	contentHeight := height - layout.colExtraHeight
	if contentHeight < 0 {
		contentHeight = 0
	}
	dynColStyle := layout.colFrame.
		Width(layout.colContentWidth).
		Height(contentHeight).
		MaxHeight(contentHeight)
	dynActiveColStyle := dynColStyle.
		BorderForeground(m.theme.Border).
		BorderStyle(lipgloss.ThickBorder())

	var renderedCols []string
	if height > 4 { // Only render if we have minimal space
		for _, realIdx := range layout.visibleIndices {
			sprint := m.sprints[realIdx]
			style := dynColStyle
			if realIdx == m.view.focusedColIdx {
				style = dynActiveColStyle
			}

			var title string
			switch sprint.SprintNumber {
			case -1:
				title = "Completed"
			case 0:
				title = "Backlog"
			case -2:
				title = "Archived"
			default:
				title = fmt.Sprintf("Sprint %d", sprint.SprintNumber)
			}

			if m.timer.ActiveSprint != nil && sprint.ID == m.timer.ActiveSprint.ID {
				title = "▶ " + title
			} else if sprint.Status == models.StatusPaused {
				title = "⏸ " + title
			}

			header := m.theme.Header.Width(layout.colContentWidth).Render(title)
			headerHeight := lipgloss.Height(header)

			// Render Goals
			visibleHeight := contentHeight - headerHeight
			if visibleHeight < 0 {
				visibleHeight = 0
			}
			type goalRange struct {
				start int
				end   int
			}
			var lines []string
			var ranges []goalRange
			if len(sprint.Goals) == 0 {
				lines = []string{m.theme.Dim.Render("  (empty)")}
			} else {
				ranges = make([]goalRange, len(sprint.Goals))
				isArchivedColumn := sprint.SprintNumber == -2
				lastArchiveDate := ""
				for j, g := range sprint.Goals {
					if isArchivedColumn {
						archiveDate := "Unknown"
						if g.ArchivedAt != nil {
							archiveDate = g.ArchivedAt.Format("2006-01-02")
						}
						if archiveDate != lastArchiveDate {
							lastArchiveDate = archiveDate
							lines = append(lines, m.theme.Dim.Render(" "+archiveDate))
						}
					}
					start := len(lines)

					// Tags
					var tags []string
					var tagView string
					if g.Tags != nil && *g.Tags != "" && *g.Tags != "[]" {
						tags = util.JSONToTags(*g.Tags)
						sort.Strings(tags)
						for _, t := range tags {
							st := m.theme.TagDefault
							switch t {
							case "urgent":
								st = m.theme.TagUrgent
							case "docs":
								st = m.theme.TagDocs
							case "blocked":
								st = m.theme.TagBlocked
							case "bug":
								st = m.theme.TagBug
							case "idea":
								st = m.theme.TagIdea
							case "review":
								st = m.theme.TagReview
							case "focus":
								st = m.theme.TagFocus
							case "later":
								st = m.theme.TagLater
							}
							tagView += " " + st.Render("#"+t)
						}
					}

					// Indentation & Icon
					indicator := "•"
					if len(g.Subtasks) > 0 {
						indicator = "▶"
						if g.Expanded {
							indicator = "▼"
						}
					}
					var icons []string
					for _, t := range tags {
						if icon, ok := tagIcon(t); ok {
							icons = append(icons, icon)
						}
					}
					if g.Blocked {
						icons = append(icons, "⛔")
					}
					if g.RecurrenceRule != nil && strings.TrimSpace(*g.RecurrenceRule) != "" {
						icons = append(icons, "↻")
					}
					if g.TaskActive {
						icons = append(icons, "⏱")
					}
					prefix := indicator
					if len(icons) > 0 {
						prefix = indicator + " " + strings.Join(icons, "")
					}
					prefix = fmt.Sprintf("%s%s ", strings.Repeat("  ", g.Level), prefix)

					// Goal Description
					priority := g.Priority
					if priority == 0 {
						priority = 3
					}
					taskSuffix := ""
					if g.TaskActive {
						taskSuffix = " ⏱" + formatDuration(taskElapsed(g.Goal))
					}
					rawLine := fmt.Sprintf("%s[P%d] %s%s #%d", prefix, priority, g.Description, taskSuffix, g.ID)
					isFocused := realIdx == m.view.focusedColIdx && j == m.view.focusedGoalIdx
					lead := "  "
					base := m.theme.Goal
					if g.Status == models.GoalStatusCompleted {
						base = m.theme.CompletedGoal
					}
					if isFocused {
						base = m.theme.Focused
						lead = "> "
					}

					leadWidth := ansi.StringWidth(lead)
					contentWidth := layout.colContentWidth - leadWidth
					if contentWidth < 1 {
						contentWidth = 1
					}
					combined := base.Render(rawLine) + tagView
					wrapped := ansi.Wrap(combined, contentWidth, "")
					goalLines := strings.Split(wrapped, "\n")
					if len(goalLines) == 0 {
						goalLines = []string{""}
					}
					indent := strings.Repeat(" ", leadWidth)
					for i := range goalLines {
						if i == 0 {
							goalLines[i] = lead + goalLines[i]
						} else {
							goalLines[i] = indent + goalLines[i]
						}
					}

					lines = append(lines, goalLines...)
					ranges[j] = goalRange{start: start, end: len(lines)}
				}
			}

			scrollStart := 0
			if len(ranges) > 0 && realIdx == m.view.focusedColIdx && m.view.focusedGoalIdx < len(ranges) {
				r := ranges[m.view.focusedGoalIdx]
				if visibleHeight > 0 {
					if r.end-r.start >= visibleHeight {
						scrollStart = r.start
					} else {
						if r.start < scrollStart {
							scrollStart = r.start
						}
						if r.end > scrollStart+visibleHeight {
							scrollStart = r.end - visibleHeight
						}
					}
				}
			}
			maxStart := len(lines) - visibleHeight
			if maxStart < 0 {
				maxStart = 0
			}
			if scrollStart > maxStart {
				scrollStart = maxStart
			}

			var visibleLines []string
			if visibleHeight > 0 {
				end := scrollStart + visibleHeight
				if end > len(lines) {
					end = len(lines)
				}
				visibleLines = append(visibleLines, lines[scrollStart:end]...)
			}
			for len(visibleLines) < visibleHeight {
				visibleLines = append(visibleLines, "")
			}
			if visibleHeight > 0 && len(lines) > visibleHeight {
				if scrollStart > 0 {
					visibleLines[0] = m.theme.Dim.Render("  ...")
				}
				if scrollStart+visibleHeight < len(lines) {
					visibleLines[len(visibleLines)-1] = m.theme.Dim.Render("  ...")
				}
			}

			goalContent := strings.Join(visibleLines, "\n")

			var colBody string
			if goalContent == "" {
				colBody = header
			} else {
				colBody = lipgloss.JoinVertical(lipgloss.Left, header, goalContent)
			}
			renderedCols = append(renderedCols, style.Render(colBody))
		}
	}

	if len(renderedCols) == 0 {
		return ""
	}
	board := lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...)
	board = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, board)
	return lipgloss.NewStyle().
		Height(height).
		MaxHeight(height).
		Render(board)
}
