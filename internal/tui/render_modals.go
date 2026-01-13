package tui

import (
	"fmt"
	"strings"

	"github.com/akyairhashvil/SSPT/internal/config"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m DashboardModel) renderLockScreen() string {
	var lockContent strings.Builder
	title := "Locked"
	if m.lock.PassphraseHash == "" {
		title = "Set Passphrase"
	}
	logo := renderLogo()
	lockTitle := fmt.Sprintf("%s | %s v%s", title, logo, versionLabel())
	lockContent.WriteString(CurrentTheme.Focused.Render(lockTitle) + "\n\n")
	if m.lock.Message != "" {
		lockContent.WriteString(CurrentTheme.Dim.Render(m.lock.Message) + "\n")
	}
	lockContent.WriteString(CurrentTheme.Focused.Render("> ") + m.lock.PassphraseInput.View())

	lockFrame := Frames.Lock
	lockBox := lockFrame.Render(lockContent.String())
	return "\x1b[H\x1b[2J" + lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, lockBox)
}

func (m DashboardModel) renderJournalPane() string {
	var journalPane string
	if m.showAnalytics {
		var analyticsContent strings.Builder
		analyticsContent.WriteString(CurrentTheme.Focused.Render("Burndown") + "\n\n")
		if len(m.workspaces) == 0 {
			analyticsContent.WriteString(CurrentTheme.Dim.Render("  (no workspaces)\n"))
		} else {
			activeWS := m.workspaces[m.activeWorkspaceIdx]
			sprints, err := m.db.GetSprints(m.ctx, m.day.ID, activeWS.ID)
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
					total, completed, err := m.db.GetSprintGoalCounts(m.ctx, id)
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
					chartWidth := m.width - config.AnalyticsChartPadding
					if chartWidth > config.AnalyticsChartMaxWidth {
						chartWidth = config.AnalyticsChartMaxWidth
					}
					if chartWidth < config.AnalyticsChartMinWidth {
						chartWidth = config.AnalyticsChartMinWidth
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
		analyticsFrame := Frames.Floating.Copy().Padding(0, 1)
		analyticsExtraWidth := lipgloss.Width(analyticsFrame.Render(""))
		analyticsWidth := m.width - analyticsExtraWidth
		if analyticsWidth < 1 {
			analyticsWidth = 1
		}
		journalPane = analyticsFrame.Width(analyticsWidth).Render(analyticsContent.String())
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
		if m.lock.PassphraseHash != "" {
			passContent.WriteString(CurrentTheme.Dim.Render("Current") + "\n")
			passContent.WriteString(currentCursor + m.passphraseCurrent.View() + "\n")
		}
		passContent.WriteString(CurrentTheme.Dim.Render("New") + "\n")
		passContent.WriteString(newCursor + m.passphraseNew.View() + "\n")
		passContent.WriteString(CurrentTheme.Dim.Render("Confirm") + "\n")
		passContent.WriteString(confirmCursor + m.passphraseConfirm.View())

		passFrame := Frames.Modal.Copy().Padding(0, 1)
		passExtraWidth := lipgloss.Width(passFrame.Render(""))
		passWidth := m.width - passExtraWidth
		if passWidth < 1 {
			passWidth = 1
		}
		journalPane = passFrame.Width(passWidth).Render(passContent.String())
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

		recFrame := Frames.Modal.Copy().Padding(0, 1)
		recExtraWidth := lipgloss.Width(recFrame.Render(""))
		recWidth := m.width - recExtraWidth
		if recWidth < 1 {
			recWidth = 1
		}
		journalPane = recFrame.Width(recWidth).Render(recContent.String())
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

		depFrame := Frames.Modal.Copy().Padding(0, 1)
		depExtraWidth := lipgloss.Width(depFrame.Render(""))
		depWidth := m.width - depExtraWidth
		if depWidth < 1 {
			depWidth = 1
		}
		journalPane = depFrame.Width(depWidth).Render(depContent.String())
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
		themeFrame := Frames.Modal.Copy().Padding(0, 1)
		themeExtraWidth := lipgloss.Width(themeFrame.Render(""))
		themeWidth := m.width - themeExtraWidth
		if themeWidth < 1 {
			themeWidth = 1
		}
		journalPane = themeFrame.Width(themeWidth).Render(themeContent.String())
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

		tagFrame := Frames.Modal.Copy().Padding(0, 1)
		tagExtraWidth := lipgloss.Width(tagFrame.Render(""))
		tagWidth := m.width - tagExtraWidth
		if tagWidth < 1 {
			tagWidth = 1
		}
		journalPane = tagFrame.Width(tagWidth).Render(tagContent.String())
	} else if m.search.Active {
		var searchContent strings.Builder
		header := "Search Results"
		if m.search.ArchiveOnly {
			header = "Search Archived"
		}
		searchContent.WriteString(CurrentTheme.Focused.Render(header) + "\n")
		searchContent.WriteString(CurrentTheme.Focused.Render("/ ") + m.search.Input.View() + "\n\n")
		if len(m.search.Results) == 0 {
			searchContent.WriteString(CurrentTheme.Dim.Render("  (no results)"))
		} else {
			for i, g := range m.search.Results {
				status := g.Status
				if status == "" {
					status = "pending"
				}
				prefix := "  "
				style := CurrentTheme.Goal
				if i == m.search.Cursor {
					prefix = "> "
					style = CurrentTheme.Focused
				}
				line := fmt.Sprintf("%s %s", CurrentTheme.Dim.Render(string(status)), g.Description)
				searchContent.WriteString(prefix + style.Render(line) + "\n")
			}
		}
		journalFrame := Frames.Modal.Copy().Padding(0, 1)
		journalExtraWidth := lipgloss.Width(journalFrame.Render(""))
		journalWidth := m.width - journalExtraWidth
		if journalWidth < 1 {
			journalWidth = 1
		}
		journalPane = journalFrame.Width(journalWidth).Render(searchContent.String())
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
			if entry.SprintID != nil {
				for _, s := range m.sprints {
					if s.ID == *entry.SprintID {
						labels = append(labels, fmt.Sprintf("S%d", s.SprintNumber))
						break
					}
				}
			}
			if entry.GoalID != nil {
				labels = append(labels, fmt.Sprintf("TASK:%d", *entry.GoalID))
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
		journalFrame := Frames.Modal.Copy().Padding(0, 1)
		journalExtraWidth := lipgloss.Width(journalFrame.Render(""))
		journalWidth := m.width - journalExtraWidth
		if journalWidth < 1 {
			journalWidth = 1
		}
		journalPane = journalFrame.Width(journalWidth).Render(journalContent.String())
	}

	return journalPane
}
