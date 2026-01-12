package tui

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/akyairhashvil/SSPT/internal/database"
	"github.com/akyairhashvil/SSPT/internal/models"
	"github.com/akyairhashvil/SSPT/internal/util"
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
	if goal.TaskActive && goal.TaskStartedAt.Valid {
		seconds += int(time.Since(goal.TaskStartedAt.Time).Seconds())
	}
	if seconds < 0 {
		seconds = 0
	}
	return time.Duration(seconds) * time.Second
}

func formatDuration(d time.Duration) string {
	total := int(d.Seconds())
	if total < 0 {
		total = 0
	}
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
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

	if m.locked {
		var lockContent strings.Builder
		title := "Locked"
		if m.passphraseHash == "" {
			title = "Set Passphrase"
		}
		logo := renderLogo()
		lockTitle := fmt.Sprintf("%s | %s v%s", title, logo, AppVersion)
		lockContent.WriteString(CurrentTheme.Focused.Render(lockTitle) + "\n\n")
		if m.lockMessage != "" {
			lockContent.WriteString(CurrentTheme.Dim.Render(m.lockMessage) + "\n")
		}
		lockContent.WriteString(CurrentTheme.Focused.Render("> ") + m.passphraseInput.View())

		lockFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(1, 2)
		lockBox := lockFrame.Render(lockContent.String())
		return "\x1b[H\x1b[2J" + lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, lockBox)
	}

	if m.err != nil {
		return fmt.Sprintf("\nError: %v\n\nPress any key to continue.", m.err)
	}
	// 1. Determine Timer Content
	var timerContent string
	var timerColor lipgloss.Style

	if m.breakActive {
		elapsed := time.Since(m.breakStart)
		rem := BreakDuration - elapsed
		if rem < 0 {
			rem = 0
		}
		timerContent = fmt.Sprintf("☕ BREAK TIME: %02d:%02d REMAINING", int(rem.Minutes()), int(rem.Seconds())%60)
		timerColor = CurrentTheme.Break
	} else if m.activeSprint != nil {
		elapsed := time.Since(m.activeSprint.StartTime.Time) + (time.Duration(m.activeSprint.ElapsedSeconds) * time.Second)
		rem := SprintDuration - elapsed
		if rem < 0 {
			rem = 0
		}
		timeStr := fmt.Sprintf("%02d:%02d", int(rem.Minutes()), int(rem.Seconds())%60)
		barView := m.progress.ViewAs(float64(elapsed) / float64(SprintDuration))
		timerContent = fmt.Sprintf("ACTIVE SPRINT: %d  |  %s  |  %s remaining", m.activeSprint.SprintNumber, barView, timeStr)
		timerColor = CurrentTheme.Focused
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
		timerColor = CurrentTheme.Dim

		if m.focusedColIdx < len(m.sprints) {
			target := m.sprints[m.focusedColIdx]
			if target.Status == "paused" {
				elapsed := time.Duration(target.ElapsedSeconds) * time.Second
				rem := SprintDuration - elapsed
				timeStr := fmt.Sprintf("%02d:%02d", int(rem.Minutes()), int(rem.Seconds())%60)
				timerContent = fmt.Sprintf("PAUSED SPRINT: %d  |  %s remaining  |  [s] to Resume", target.SprintNumber, timeStr)
				timerColor = CurrentTheme.Break
			}
		}
	}

	if timerContent == "" {
		timerContent = "SSPT - Ready"
		timerColor = CurrentTheme.Dim
	}
	cipherOn, encrypted, cipherVer := database.EncryptionStatus()
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
	timerContent = fmt.Sprintf("%s  |  %s  |  %s  |  %s v%s", timerContent, dbLabel, cipherLabel, logo, AppVersion)

	// 2. Render Header (Timer Box)
	headerFrame := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(CurrentTheme.Border).
		Padding(0, 1)
	headerExtra := lipgloss.Width(headerFrame.Render(""))
	headerWidth := m.width - headerExtra
	if headerWidth < 1 {
		headerWidth = 1
	}
	timerBox := headerFrame.Width(headerWidth).Render(timerColor.Render(timerContent))

	// 3. Render Footer
	var footer string
	var footerContent string
	var footerHelpLines []string
	var rawFooter string
	hasStatusMessage := m.statusMessage != ""
	if hasStatusMessage {
		statusStyle := CurrentTheme.Break.Copy().Foreground(lipgloss.Color("196"))
		if !m.statusIsError {
			statusStyle = CurrentTheme.Break.Copy().Foreground(lipgloss.Color("208"))
		}
		footerContent = statusStyle.Render(m.statusMessage)
	} else if m.Message != "" {
		footerContent = CurrentTheme.Break.Copy().Foreground(lipgloss.Color("208")).Render(m.Message)
	} else if m.creatingGoal || m.editingGoal || m.creatingWorkspace || m.initializingSprints {
		footerContent = CurrentTheme.Input.Render(m.textInput.View())
	} else if m.tagging {
		footerContent = CurrentTheme.Dim.Render("[Tab] Toggle Tag | [Enter] Save | [Esc] Cancel")
	} else if m.themePicking {
		footerContent = CurrentTheme.Dim.Render("[Enter] Apply Theme | [Esc] Cancel")
	} else if m.depPicking {
		footerContent = CurrentTheme.Dim.Render("[Space] Toggle | [Enter] Save | [Esc] Cancel")
	} else if m.settingRecurrence {
		footerContent = CurrentTheme.Dim.Render("[Tab] Next | [Space] Toggle | [Enter] Save | [Esc] Cancel")
	} else if m.confirmingDelete {
		footerContent = CurrentTheme.Focused.Render("Delete task? [d] Delete | [a] Archive | [Esc] Cancel")
	} else if m.confirmingClearDB {
		var lines []string
		lines = append(lines, CurrentTheme.Focused.Render("Clear database? This deletes all data."))
		if m.clearDBStatus != "" {
			lines = append(lines, CurrentTheme.Break.Render(m.clearDBStatus))
		}
		if m.clearDBNeedsPass {
			lines = append(lines, CurrentTheme.Dim.Render("Enter passphrase to confirm:"))
			lines = append(lines, CurrentTheme.Focused.Render("> ")+m.passphraseInput.View())
		} else {
			lines = append(lines, CurrentTheme.Dim.Render("[c] Clear | [Esc] Cancel"))
		}
		footerContent = lipgloss.JoinVertical(lipgloss.Left, lines...)
	} else if m.changingPassphrase {
		footerContent = CurrentTheme.Dim.Render("[Enter] Next | [Esc] Cancel")
	} else if m.journaling {
		// Only render journaling input in the journal pane, avoid duplicate
		// footer = fmt.Sprintf("%s", CurrentTheme.Input.Render(m.journalInput.View()))
		footerContent = CurrentTheme.Dim.Render("[Enter] to Save Log | [Esc] Cancel")
	} else if m.movingGoal {
		footerContent = CurrentTheme.Focused.Render("MOVE TO: [0] Backlog | [1-8] Sprint # | [Esc] Cancel")
	} else {
		baseHelp := "[n]New|[N]Sub|[e]Edit|[z]Toggle|[T]Task|[P]Priority|[+/-]Sprint|[w]Cycle|[W]New WS|[t]Tag|[m]Move|[D]Deps|[R]Repeat|[/]Search|[J]Journal|[I]Import|[G]Graph|[p]Passphrase|[d]Delete|[A]Archive|[u]Unarchive|[L]Lock|[C]Clear DB|[b]Backlog|[c]Completed|[a]Archived|[v]View|[Y]Theme"
		var timerHelp string
		if m.activeSprint != nil {
			timerHelp = "|[s]PAUSE|[x]STOP"
		} else {
			timerHelp = "|[s]Start"
		}
		fullHelp := baseHelp + timerHelp + "|[ctrl+e]Export|[ctrl+r]Report|[q]Quit"
		rawFooter = fullHelp
		footerContent = CurrentTheme.Dim.Render(fullHelp)
	}
	if footerContent != "" {
		boxed := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		innerWidth := m.width - lipgloss.Width(boxed.Render(""))
		if innerWidth < 1 {
			innerWidth = 1
		}
		content := footerContent
		if hasStatusMessage || m.Message != "" {
			content = lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, footerContent)
		} else if !m.creatingGoal && !m.editingGoal && !m.creatingWorkspace && !m.initializingSprints && !m.tagging && !m.themePicking && !m.depPicking && !m.settingRecurrence && !m.confirmingDelete && !m.confirmingClearDB && !m.changingPassphrase {
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
					footerHelpLines = append(footerHelpLines, lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, CurrentTheme.Dim.Render(line)))
				}
				content = lipgloss.JoinVertical(lipgloss.Left, footerHelpLines...)
			}
		} else if !m.confirmingDelete && !m.confirmingClearDB && !m.changingPassphrase && (m.creatingGoal || m.editingGoal || m.creatingWorkspace || m.initializingSprints || m.tagging || m.themePicking || m.depPicking || m.settingRecurrence) {
			content = footerContent
		} else if m.changingPassphrase {
			content = lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, footerContent)
		} else if m.confirmingDelete {
			content = lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, footerContent)
		} else if m.confirmingClearDB {
			content = footerContent
		}
		footer = boxed.Width(innerWidth).Render(content)
	}

	splitLines := func(s string) []string {
		if s == "" {
			return nil
		}
		return strings.Split(s, "\n")
	}
	trimLines := func(s string, max int) string {
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

	// 4. Render Journal/Search Pane
	var journalPane string
	journalHeight := 0
	if m.showAnalytics {
		var analyticsContent strings.Builder
		analyticsContent.WriteString(CurrentTheme.Focused.Render("Burndown") + "\n\n")
		if len(m.workspaces) == 0 {
			analyticsContent.WriteString(CurrentTheme.Dim.Render("  (no workspaces)\n"))
		} else {
			activeWS := m.workspaces[m.activeWorkspaceIdx]
			sprints, err := database.GetSprints(m.day.ID, activeWS.ID)
			if err != nil {
				analyticsContent.WriteString(CurrentTheme.Break.Render(fmt.Sprintf("  error loading sprints: %v\n", err)))
			} else {
				var sprintIDs []int64
				var sprintNums []int
				for _, s := range sprints {
					if s.SprintNumber > 0 {
						sprintIDs = append(sprintIDs, s.ID)
						sprintNums = append(sprintNums, s.SprintNumber)
					}
				}
				totalAll := 0
				completedAll := 0
				var perSprintTotals []int
				var perSprintCompleted []int
				for _, id := range sprintIDs {
					total, completed, err := database.GetSprintGoalCounts(id)
					if err != nil {
						continue
					}
					perSprintTotals = append(perSprintTotals, total)
					perSprintCompleted = append(perSprintCompleted, completed)
					totalAll += total
					completedAll += completed
				}
				if totalAll == 0 || len(sprintIDs) == 0 {
					analyticsContent.WriteString(CurrentTheme.Dim.Render("  (no sprint tasks)\n"))
				} else {
					analyticsContent.WriteString(CurrentTheme.Dim.Render(fmt.Sprintf("Total: %d  Completed: %d  Remaining: %d\n\n", totalAll, completedAll, totalAll-completedAll)))
					analyticsContent.WriteString(CurrentTheme.Dim.Render("Sprint  Done/All  Progress\n"))
					chartWidth := m.width - 24
					if chartWidth > 48 {
						chartWidth = 48
					}
					if chartWidth < 10 {
						chartWidth = 10
					}
					cumCompleted := 0
					for i := range sprintIDs {
						cumCompleted += perSprintCompleted[i]
						done := cumCompleted
						if done < 0 {
							done = 0
						}
						if done > totalAll {
							done = totalAll
						}
						filled := int(float64(done) / float64(totalAll) * float64(chartWidth))
						if filled < 0 {
							filled = 0
						}
						if filled > chartWidth {
							filled = chartWidth
						}
						bar := strings.Repeat("#", filled) + strings.Repeat(".", chartWidth-filled)
						label := fmt.Sprintf("S%d", sprintNums[i])
						analyticsContent.WriteString(fmt.Sprintf("%-3s %3d/%-3d %s\n", label, done, totalAll, bar))
					}
				}
			}
		}
		analyticsFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		analyticsExtraWidth := lipgloss.Width(analyticsFrame.Render(""))
		analyticsWidth := m.width - analyticsExtraWidth
		if analyticsWidth < 1 {
			analyticsWidth = 1
		}
		journalPane = analyticsFrame.Width(analyticsWidth).Render(analyticsContent.String())
		journalHeight = len(splitLines(journalPane))
	} else if m.changingPassphrase {
		var passContent strings.Builder
		passContent.WriteString(CurrentTheme.Focused.Render("Change Passphrase") + "\n\n")
		if m.passphraseStatus != "" {
			passContent.WriteString(CurrentTheme.Break.Render(m.passphraseStatus) + "\n")
		}
		currentCursor := "  "
		newCursor := "  "
		confirmCursor := "  "
		switch m.passphraseStage {
		case 0:
			currentCursor = "> "
		case 1:
			newCursor = "> "
		case 2:
			confirmCursor = "> "
		}
		if m.passphraseHash != "" {
			passContent.WriteString(CurrentTheme.Dim.Render("Current") + "\n")
			passContent.WriteString(currentCursor + m.passphraseCurrent.View() + "\n")
		}
		passContent.WriteString(CurrentTheme.Dim.Render("New") + "\n")
		passContent.WriteString(newCursor + m.passphraseNew.View() + "\n")
		passContent.WriteString(CurrentTheme.Dim.Render("Confirm") + "\n")
		passContent.WriteString(confirmCursor + m.passphraseConfirm.View())

		passFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		passExtraWidth := lipgloss.Width(passFrame.Render(""))
		passWidth := m.width - passExtraWidth
		if passWidth < 1 {
			passWidth = 1
		}
		journalPane = passFrame.Width(passWidth).Render(passContent.String())
		journalHeight = lipgloss.Height(journalPane)
	} else if m.settingRecurrence {
		var recContent strings.Builder
		recContent.WriteString(CurrentTheme.Focused.Render("Recurrence") + "\n")
		recContent.WriteString(CurrentTheme.Dim.Render("Tab next step | Space toggle | Enter save") + "\n\n")

		if m.recurrenceFocus == "mode" {
			recContent.WriteString(CurrentTheme.Focused.Render("Frequency") + "\n")
			for i, opt := range m.recurrenceOptions {
				cursor := "  "
				if i == m.recurrenceCursor {
					cursor = "> "
				}
				marker := " "
				if opt == m.recurrenceMode {
					marker = "*"
				}
				recContent.WriteString(fmt.Sprintf("%s[%s] %s\n", cursor, marker, opt))
			}
		} else if m.recurrenceMode == "weekly" {
			recContent.WriteString(CurrentTheme.Dim.Render("Frequency: weekly") + "\n\n")
			recContent.WriteString(CurrentTheme.Focused.Render("Weekdays") + "\n")
			for i, d := range m.weekdayOptions {
				cursor := "  "
				if m.recurrenceFocus == "items" && i == m.recurrenceItemCursor {
					cursor = "> "
				}
				check := "[ ]"
				if m.recurrenceSelected[d] {
					check = "[x]"
				}
				recContent.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, d))
			}
		} else if m.recurrenceMode == "monthly" {
			recContent.WriteString(CurrentTheme.Dim.Render("Frequency: monthly") + "\n\n")
			if m.recurrenceFocus == "items" {
				recContent.WriteString(CurrentTheme.Focused.Render("Months") + "\n")
				for i, mo := range m.monthOptions {
					cursor := "  "
					if m.recurrenceFocus == "items" && i == m.recurrenceItemCursor {
						cursor = "> "
					}
					check := "[ ]"
					if m.recurrenceSelected[mo] {
						check = "[x]"
					}
					recContent.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, mo))
				}
			} else if m.recurrenceFocus == "days" {
				recContent.WriteString(CurrentTheme.Focused.Render("Days") + "\n")
				maxDay := m.monthlyMaxDay()
				if maxDay <= 0 {
					recContent.WriteString(CurrentTheme.Dim.Render("  (select month(s) first)"))
				} else {
					if m.recurrenceDayCursor > maxDay-1 {
						m.recurrenceDayCursor = maxDay - 1
					}
					var entries []string
					for i := 0; i < maxDay; i++ {
						d := m.monthDayOptions[i]
						cursor := "  "
						if m.recurrenceFocus == "days" && i == m.recurrenceDayCursor {
							cursor = "> "
						}
						check := "[ ]"
						if m.recurrenceSelected["day:"+d] {
							check = "[x]"
						}
						entries = append(entries, fmt.Sprintf("%s%s %2s", cursor, check, d))
					}
					colWidth := 0
					for _, entry := range entries {
						w := ansi.StringWidth(entry)
						if w > colWidth {
							colWidth = w
						}
					}
					rows := (len(entries) + 1) / 2
					for i := 0; i < rows; i++ {
						left := entries[i]
						right := ""
						if i+rows < len(entries) {
							right = entries[i+rows]
						}
						padding := colWidth - ansi.StringWidth(left)
						if padding < 0 {
							padding = 0
						}
						line := left + strings.Repeat(" ", padding+2) + right
						recContent.WriteString(line + "\n")
					}
				}
			}
		}

		recFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		recExtraWidth := lipgloss.Width(recFrame.Render(""))
		recWidth := m.width - recExtraWidth
		if recWidth < 1 {
			recWidth = 1
		}
		journalPane = recFrame.Width(recWidth).Render(recContent.String())
		journalHeight = lipgloss.Height(journalPane)
	} else if m.depPicking {
		var depContent strings.Builder
		depContent.WriteString(CurrentTheme.Focused.Render("Dependencies") + "\n")
		depContent.WriteString(CurrentTheme.Dim.Render("Space to toggle, Enter to save") + "\n\n")
		if len(m.depOptions) == 0 {
			depContent.WriteString(CurrentTheme.Dim.Render("  (no tasks)\n"))
		} else {
			maxLines := m.height / 2
			if maxLines < 6 {
				maxLines = 6
			}
			start := 0
			if m.depCursor >= maxLines {
				start = m.depCursor - maxLines + 1
			}
			end := start + maxLines
			if end > len(m.depOptions) {
				end = len(m.depOptions)
			}
			if start > 0 {
				depContent.WriteString(CurrentTheme.Dim.Render("  ...\n"))
			}
			for i := start; i < end; i++ {
				opt := m.depOptions[i]
				cursor := "  "
				if i == m.depCursor {
					cursor = "> "
				}
				check := "[ ]"
				if m.depSelected[opt.ID] {
					check = "[x]"
				}
				depContent.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, opt.Label))
			}
			if end < len(m.depOptions) {
				depContent.WriteString(CurrentTheme.Dim.Render("  ...\n"))
			}
		}

		depFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		depExtraWidth := lipgloss.Width(depFrame.Render(""))
		depWidth := m.width - depExtraWidth
		if depWidth < 1 {
			depWidth = 1
		}
		journalPane = depFrame.Width(depWidth).Render(depContent.String())
		journalHeight = lipgloss.Height(journalPane)
	} else if m.themePicking {
		var themeContent strings.Builder
		themeContent.WriteString(CurrentTheme.Focused.Render("Themes") + "\n")
		themeContent.WriteString(CurrentTheme.Dim.Render("Use ↑/↓ to select, Enter to apply") + "\n\n")
		if len(m.themeNames) == 0 {
			themeContent.WriteString(CurrentTheme.Dim.Render("  (no themes)\n"))
		} else {
			for i, name := range m.themeNames {
				cursor := "  "
				if i == m.themeCursor {
					cursor = "> "
				}
				themeContent.WriteString(fmt.Sprintf("%s%s\n", cursor, name))
			}
		}
		themeFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		themeExtraWidth := lipgloss.Width(themeFrame.Render(""))
		themeWidth := m.width - themeExtraWidth
		if themeWidth < 1 {
			themeWidth = 1
		}
		journalPane = themeFrame.Width(themeWidth).Render(themeContent.String())
		journalHeight = lipgloss.Height(journalPane)
	} else if m.tagging {
		var tagContent strings.Builder
		tagContent.WriteString(CurrentTheme.Focused.Render("Tags") + "\n")
		tagContent.WriteString(CurrentTheme.Dim.Render("Use ↑/↓ to select, Tab to toggle, Enter to save") + "\n\n")
		for i, tag := range m.defaultTags {
			cursor := "  "
			if i == m.tagCursor {
				cursor = "> "
			}
			check := "[ ]"
			if m.tagSelected[tag] {
				check = "[x]"
			}
			tagContent.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, tag))
		}
		if len(m.defaultTags) == 0 {
			tagContent.WriteString(CurrentTheme.Dim.Render("  (no default tags)\n"))
		}
		tagContent.WriteString("\n" + CurrentTheme.Focused.Render("Custom") + "\n")
		tagContent.WriteString(CurrentTheme.Focused.Render("> ") + m.tagInput.View())

		tagFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		tagExtraWidth := lipgloss.Width(tagFrame.Render(""))
		tagWidth := m.width - tagExtraWidth
		if tagWidth < 1 {
			tagWidth = 1
		}
		journalPane = tagFrame.Width(tagWidth).Render(tagContent.String())
		journalHeight = lipgloss.Height(journalPane)
	} else if m.searching {
		var searchContent strings.Builder
		header := "Search Results"
		if m.searchArchiveOnly {
			header = "Search Archived"
		}
		searchContent.WriteString(CurrentTheme.Focused.Render(header) + "\n")
		searchContent.WriteString(CurrentTheme.Focused.Render("/ ") + m.searchInput.View() + "\n\n")
		if len(m.searchResults) == 0 {
			searchContent.WriteString(CurrentTheme.Dim.Render("  (no results)"))
		} else {
			for i, g := range m.searchResults {
				status := g.Status
				if status == "" {
					status = "pending"
				}
				prefix := "  "
				style := CurrentTheme.Goal
				if i == m.searchCursor {
					prefix = "> "
					style = CurrentTheme.Focused
				}
				line := fmt.Sprintf("%s %s", CurrentTheme.Dim.Render(status), g.Description)
				searchContent.WriteString(prefix + style.Render(line) + "\n")
			}
		}
		journalFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		journalExtraWidth := lipgloss.Width(journalFrame.Render(""))
		journalWidth := m.width - journalExtraWidth
		if journalWidth < 1 {
			journalWidth = 1
		}
		journalPane = journalFrame.Width(journalWidth).Render(searchContent.String())
		journalHeight = lipgloss.Height(journalPane)
	} else if len(m.journalEntries) > 0 || m.journaling {
		var journalContent strings.Builder
		journalContent.WriteString(CurrentTheme.Focused.Render("Journal") + "\n\n")
		start := len(m.journalEntries) - 3
		if start < 0 {
			start = 0
		}
		for i := start; i < len(m.journalEntries); i++ {
			entry := m.journalEntries[i]
			var labels []string
			if entry.SprintID.Valid {
				for _, s := range m.sprints {
					if s.ID == entry.SprintID.Int64 {
						labels = append(labels, fmt.Sprintf("S%d", s.SprintNumber))
						break
					}
				}
			}
			if entry.GoalID.Valid {
				labels = append(labels, fmt.Sprintf("TASK:%d", entry.GoalID.Int64))
			}
			labelStr := ""
			if len(labels) > 0 {
				labelStr = fmt.Sprintf("[%s] ", strings.Join(labels, "|"))
			}
			line := fmt.Sprintf("%s %s%s",
				CurrentTheme.Dim.Render(entry.CreatedAt.Format("15:04")),
				CurrentTheme.Highlight.Render(labelStr),
				entry.Content)
			journalContent.WriteString(line + "\n")
		}
		if m.journaling {
			journalContent.WriteString("\n" + CurrentTheme.Focused.Render("> ") + m.journalInput.View())
		}
		journalFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
		journalExtraWidth := lipgloss.Width(journalFrame.Render(""))
		journalWidth := m.width - journalExtraWidth
		if journalWidth < 1 {
			journalWidth = 1
		}
		journalPane = journalFrame.Width(journalWidth).Render(journalContent.String())
		journalHeight = lipgloss.Height(journalPane)
	}

	// 5. Calculate Layout Dimensions
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
		journalHeight = len(splitLines(journalPane))
	}
	columnHeight := availableLines - journalHeight
	if columnHeight < 0 {
		columnHeight = 0
	}

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
		if sprint.Status == "completed" && sprint.SprintNumber > 0 {
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

	displayCount := 4
	if m.viewMode == ViewModeMinimal {
		displayCount = 3
	}

	colFrame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(CurrentTheme.Dim.GetForeground()).
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
	if m.colScrollOffset > len(scrollableIndices)-displayCount {
		m.colScrollOffset = len(scrollableIndices) - displayCount
	}
	if m.colScrollOffset < 0 {
		m.colScrollOffset = 0
	}

	var visibleIndices []int
	for i := 0; i < displayCount; i++ {
		idx := m.colScrollOffset + i
		if idx < len(scrollableIndices) {
			visibleIndices = append(visibleIndices, scrollableIndices[idx])
		}
	}

	renderBoard := func(height int) string {
		if height <= 0 {
			return ""
		}
		contentHeightLocal := height - colExtraHeight
		if contentHeightLocal < 0 {
			contentHeightLocal = 0
		}
		dynColStyleLocal := colFrame.Copy().
			Width(colContentWidth).
			Height(contentHeightLocal).
			MaxHeight(contentHeightLocal)
		dynActiveColStyleLocal := dynColStyleLocal.Copy().
			BorderForeground(CurrentTheme.Border).
			BorderStyle(lipgloss.ThickBorder())

		var renderedCols []string
		if height > 4 { // Only render if we have minimal space
			for _, realIdx := range visibleIndices {
				sprint := m.sprints[realIdx]
				style := dynColStyleLocal
				if realIdx == m.focusedColIdx {
					style = dynActiveColStyleLocal
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

				if m.activeSprint != nil && sprint.ID == m.activeSprint.ID {
					title = "▶ " + title
				} else if sprint.Status == "paused" {
					title = "⏸ " + title
				}

				header := CurrentTheme.Header.Copy().Width(colContentWidth).Render(title)
				headerHeight := lipgloss.Height(header)

				// Render Goals
				visibleHeight := contentHeightLocal - headerHeight
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
					lines = []string{CurrentTheme.Dim.Render("  (empty)")}
				} else {
					ranges = make([]goalRange, len(sprint.Goals))
					isArchivedColumn := sprint.SprintNumber == -2
					lastArchiveDate := ""
					for j, g := range sprint.Goals {
						if isArchivedColumn {
							archiveDate := "Unknown"
							if g.ArchivedAt.Valid {
								archiveDate = g.ArchivedAt.Time.Format("2006-01-02")
							}
							if archiveDate != lastArchiveDate {
								lastArchiveDate = archiveDate
								lines = append(lines, CurrentTheme.Dim.Render(" "+archiveDate))
							}
						}
						start := len(lines)

						// Tags
						var tags []string
						var tagView string
						if g.Tags.Valid && g.Tags.String != "" && g.Tags.String != "[]" {
							tags = util.JSONToTags(g.Tags.String)
							sort.Strings(tags)
							for _, t := range tags {
								st := CurrentTheme.TagDefault
								switch t {
								case "urgent":
									st = CurrentTheme.TagUrgent
								case "docs":
									st = CurrentTheme.TagDocs
								case "blocked":
									st = CurrentTheme.TagBlocked
								case "bug":
									st = CurrentTheme.TagBug
								case "idea":
									st = CurrentTheme.TagIdea
								case "review":
									st = CurrentTheme.TagReview
								case "focus":
									st = CurrentTheme.TagFocus
								case "later":
									st = CurrentTheme.TagLater
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
						if g.RecurrenceRule.Valid && strings.TrimSpace(g.RecurrenceRule.String) != "" {
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
							taskSuffix = " ⏱" + formatDuration(taskElapsed(g))
						}
						rawLine := fmt.Sprintf("%s[P%d] %s%s #%d", prefix, priority, g.Description, taskSuffix, g.ID)
						isFocused := realIdx == m.focusedColIdx && j == m.focusedGoalIdx
						lead := "  "
						base := CurrentTheme.Goal.Copy()
						if g.Status == "completed" {
							base = CurrentTheme.CompletedGoal.Copy()
						}
						if isFocused {
							base = CurrentTheme.Focused.Copy()
							lead = "> "
						}

						leadWidth := ansi.StringWidth(lead)
						contentWidth := colContentWidth - leadWidth
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
				if len(ranges) > 0 && realIdx == m.focusedColIdx && m.focusedGoalIdx < len(ranges) {
					r := ranges[m.focusedGoalIdx]
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
						visibleLines[0] = CurrentTheme.Dim.Render("  ...")
					}
					if scrollStart+visibleHeight < len(lines) {
						visibleLines[len(visibleLines)-1] = CurrentTheme.Dim.Render("  ...")
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

	// 7. Assemble Final View
	// Use explicit block rendering to ensure order
	var finalView string

	// Header Block
	finalView = lipgloss.JoinVertical(lipgloss.Left, finalView, timerBox)

	boardHeight := columnHeight
	if footer != "" && boardHeight > 0 {
		boardHeight--
	}
	var board string
	if m.height > 0 {
		for boardHeight > 0 {
			board = strings.TrimRight(renderBoard(boardHeight), "\n")
			boardLines := len(splitLines(board))
			journalLines := len(splitLines(journalPane))
			total := len(headerLines) + boardLines + journalLines + footerGap + len(footerSplit)
			if total <= m.height {
				break
			}
			boardHeight--
		}
	} else {
		board = renderBoard(boardHeight)
	}
	if boardHeight == 0 {
		board = renderBoard(0)
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
