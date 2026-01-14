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
	if m.security.lock.PassphraseHash == "" {
		title = "Set Passphrase"
	}
	logo := renderLogo()
	lockTitle := fmt.Sprintf("%s | %s v%s", title, logo, versionLabel())
	lockContent.WriteString(m.theme.Focused.Render(lockTitle) + "\n\n")
	if m.security.lock.Message != "" {
		lockContent.WriteString(m.theme.Dim.Render(m.security.lock.Message) + "\n")
	}
	lockContent.WriteString(m.theme.Focused.Render("> ") + m.security.lock.PassphraseInput.View())

	lockFrame := Frames.Lock
	lockBox := lockFrame.Render(lockContent.String())
	return "\x1b[H\x1b[2J" + lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, lockBox)
}

func (m DashboardModel) renderJournalPane() string {
	var journalPane string
	if m.showAnalytics {
		var analyticsContent strings.Builder
		analyticsContent.WriteString(m.theme.Focused.Render("Burndown") + "\n\n")
		if len(m.workspaces) == 0 {
			analyticsContent.WriteString(m.theme.Dim.Render("  (no workspaces)\n"))
		} else {
			activeWS := m.workspaces[m.activeWorkspaceIdx]
			sprints, err := m.db.GetSprints(m.ctx, m.day.ID, activeWS.ID)
			if err != nil {
				analyticsContent.WriteString(m.theme.Break.Render(fmt.Sprintf("  error loading sprints: %v\n", err)))
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
				var perSprintCompleted []int
				for _, id := range sprintIDs {
					total, completed, err := m.db.GetSprintGoalCounts(m.ctx, id)
					if err != nil {
						continue
					}
					perSprintCompleted = append(perSprintCompleted, completed)
					totalAll += total
					completedAll += completed
				}
				if totalAll == 0 || len(sprintIDs) == 0 {
					analyticsContent.WriteString(m.theme.Dim.Render("  (no sprint tasks)\n"))
				} else {
					analyticsContent.WriteString(m.theme.Dim.Render(fmt.Sprintf("Total: %d  Completed: %d  Remaining: %d\n\n", totalAll, completedAll, totalAll-completedAll)))
					analyticsContent.WriteString(m.theme.Dim.Render("Sprint  Done/All  Progress\n"))
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
		analyticsFrame := Frames.Floating.Padding(0, 1)
		analyticsExtraWidth := lipgloss.Width(analyticsFrame.Render(""))
		analyticsWidth := m.width - analyticsExtraWidth
		if analyticsWidth < 1 {
			analyticsWidth = 1
		}
		journalPane = analyticsFrame.Width(analyticsWidth).Render(analyticsContent.String())
	} else if m.security.changingPassphrase {
		var passContent strings.Builder
		passContent.WriteString(m.theme.Focused.Render("Change Passphrase") + "\n\n")
		if m.security.passphraseStatus != "" {
			passContent.WriteString(m.theme.Break.Render(m.security.passphraseStatus) + "\n")
		}
		currentCursor := "  "
		newCursor := "  "
		confirmCursor := "  "
		switch m.security.passphraseStage {
		case 0:
			currentCursor = "> "
		case 1:
			newCursor = "> "
		case 2:
			confirmCursor = "> "
		}
		if m.security.lock.PassphraseHash != "" {
			passContent.WriteString(m.theme.Dim.Render("Current") + "\n")
			passContent.WriteString(currentCursor + m.inputs.passphraseCurrent.View() + "\n")
		}
		passContent.WriteString(m.theme.Dim.Render("New") + "\n")
		passContent.WriteString(newCursor + m.inputs.passphraseNew.View() + "\n")
		passContent.WriteString(m.theme.Dim.Render("Confirm") + "\n")
		passContent.WriteString(confirmCursor + m.inputs.passphraseConfirm.View())

		passFrame := Frames.Modal.Padding(0, 1)
		passExtraWidth := lipgloss.Width(passFrame.Render(""))
		passWidth := m.width - passExtraWidth
		if passWidth < 1 {
			passWidth = 1
		}
		journalPane = passFrame.Width(passWidth).Render(passContent.String())
	} else if m.modal.settingRecurrence {
		var recContent strings.Builder
		recContent.WriteString(m.theme.Focused.Render("Recurrence") + "\n")
		recContent.WriteString(m.theme.Dim.Render("Tab next step | Space toggle | Enter save") + "\n\n")

		if m.modal.recurrenceFocus == "mode" {
			recContent.WriteString(m.theme.Focused.Render("Frequency") + "\n")
			for i, opt := range m.modal.recurrenceOptions {
				cursor := "  "
				if i == m.modal.recurrenceCursor {
					cursor = "> "
				}
				marker := " "
				if opt == m.modal.recurrenceMode {
					marker = "*"
				}
				recContent.WriteString(fmt.Sprintf("%s[%s] %s\n", cursor, marker, opt))
			}
		} else if m.modal.recurrenceMode == "weekly" {
			recContent.WriteString(m.theme.Dim.Render("Frequency: weekly") + "\n\n")
			recContent.WriteString(m.theme.Focused.Render("Weekdays") + "\n")
			for i, d := range m.modal.weekdayOptions {
				cursor := "  "
				if m.modal.recurrenceFocus == "items" && i == m.modal.recurrenceItemCursor {
					cursor = "> "
				}
				check := "[ ]"
				if m.modal.recurrenceSelected[d] {
					check = "[x]"
				}
				recContent.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, d))
			}
		} else if m.modal.recurrenceMode == "monthly" {
			recContent.WriteString(m.theme.Dim.Render("Frequency: monthly") + "\n\n")
			if m.modal.recurrenceFocus == "items" {
				recContent.WriteString(m.theme.Focused.Render("Months") + "\n")
				for i, mo := range m.modal.monthOptions {
					cursor := "  "
					if m.modal.recurrenceFocus == "items" && i == m.modal.recurrenceItemCursor {
						cursor = "> "
					}
					check := "[ ]"
					if m.modal.recurrenceSelected[mo] {
						check = "[x]"
					}
					recContent.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, mo))
				}
			} else if m.modal.recurrenceFocus == "days" {
				recContent.WriteString(m.theme.Focused.Render("Days") + "\n")
				maxDay := m.monthlyMaxDay()
				if maxDay <= 0 {
					recContent.WriteString(m.theme.Dim.Render("  (select month(s) first)"))
				} else {
					if m.modal.recurrenceDayCursor > maxDay-1 {
						m.modal.recurrenceDayCursor = maxDay - 1
					}
					var entries []string
					for i := 0; i < maxDay; i++ {
						d := m.modal.monthDayOptions[i]
						cursor := "  "
						if m.modal.recurrenceFocus == "days" && i == m.modal.recurrenceDayCursor {
							cursor = "> "
						}
						check := "[ ]"
						if m.modal.recurrenceSelected["day:"+d] {
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

		recFrame := Frames.Modal.Padding(0, 1)
		recExtraWidth := lipgloss.Width(recFrame.Render(""))
		recWidth := m.width - recExtraWidth
		if recWidth < 1 {
			recWidth = 1
		}
		journalPane = recFrame.Width(recWidth).Render(recContent.String())
	} else if m.modal.depPicking {
		var depContent strings.Builder
		depContent.WriteString(m.theme.Focused.Render("Dependencies") + "\n")
		depContent.WriteString(m.theme.Dim.Render("Space to toggle, Enter to save") + "\n\n")
		if len(m.modal.depOptions) == 0 {
			depContent.WriteString(m.theme.Dim.Render("  (no tasks)\n"))
		} else {
			maxLines := m.height / 2
			if maxLines < 6 {
				maxLines = 6
			}
			start := 0
			if m.modal.depCursor >= maxLines {
				start = m.modal.depCursor - maxLines + 1
			}
			end := start + maxLines
			if end > len(m.modal.depOptions) {
				end = len(m.modal.depOptions)
			}
			if start > 0 {
				depContent.WriteString(m.theme.Dim.Render("  ...\n"))
			}
			for i := start; i < end; i++ {
				opt := m.modal.depOptions[i]
				cursor := "  "
				if i == m.modal.depCursor {
					cursor = "> "
				}
				check := "[ ]"
				if m.modal.depSelected[opt.ID] {
					check = "[x]"
				}
				depContent.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, opt.Label))
			}
			if end < len(m.modal.depOptions) {
				depContent.WriteString(m.theme.Dim.Render("  ...\n"))
			}
		}

		depFrame := Frames.Modal.Padding(0, 1)
		depExtraWidth := lipgloss.Width(depFrame.Render(""))
		depWidth := m.width - depExtraWidth
		if depWidth < 1 {
			depWidth = 1
		}
		journalPane = depFrame.Width(depWidth).Render(depContent.String())
	} else if m.modal.themePicking {
		var themeContent strings.Builder
		themeContent.WriteString(m.theme.Focused.Render("Themes") + "\n")
		themeContent.WriteString(m.theme.Dim.Render("Use ↑/↓ to select, Enter to apply") + "\n\n")
		if len(m.modal.themeNames) == 0 {
			themeContent.WriteString(m.theme.Dim.Render("  (no themes)\n"))
		} else {
			for i, name := range m.modal.themeNames {
				cursor := "  "
				if i == m.modal.themeCursor {
					cursor = "> "
				}
				themeContent.WriteString(fmt.Sprintf("%s%s\n", cursor, name))
			}
		}
		themeFrame := Frames.Modal.Padding(0, 1)
		themeExtraWidth := lipgloss.Width(themeFrame.Render(""))
		themeWidth := m.width - themeExtraWidth
		if themeWidth < 1 {
			themeWidth = 1
		}
		journalPane = themeFrame.Width(themeWidth).Render(themeContent.String())
	} else if m.modal.tagging {
		var tagContent strings.Builder
		tagContent.WriteString(m.theme.Focused.Render("Tags") + "\n")
		tagContent.WriteString(m.theme.Dim.Render("Use ↑/↓ to select, Tab to toggle, Enter to save") + "\n\n")
		for i, tag := range m.modal.defaultTags {
			cursor := "  "
			if i == m.modal.tagCursor {
				cursor = "> "
			}
			check := "[ ]"
			if m.modal.tagSelected[tag] {
				check = "[x]"
			}
			tagContent.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, tag))
		}
		if len(m.modal.defaultTags) == 0 {
			tagContent.WriteString(m.theme.Dim.Render("  (no default tags)\n"))
		}
		tagContent.WriteString("\n" + m.theme.Focused.Render("Custom") + "\n")
		tagContent.WriteString(m.theme.Focused.Render("> ") + m.inputs.tagInput.View())

		tagFrame := Frames.Modal.Padding(0, 1)
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
		searchContent.WriteString(m.theme.Focused.Render(header) + "\n")
		searchContent.WriteString(m.theme.Focused.Render("/ ") + m.search.Input.View() + "\n\n")
		if len(m.search.Results) == 0 {
			searchContent.WriteString(m.theme.Dim.Render("  (no results)"))
		} else {
			for i, g := range m.search.Results {
				status := g.Status
				if status == "" {
					status = "pending"
				}
				prefix := "  "
				style := m.theme.Goal
				if i == m.search.Cursor {
					prefix = "> "
					style = m.theme.Focused
				}
				line := fmt.Sprintf("%s %s", m.theme.Dim.Render(string(status)), g.Description)
				searchContent.WriteString(prefix + style.Render(line) + "\n")
			}
		}
		journalFrame := Frames.Modal.Padding(0, 1)
		journalExtraWidth := lipgloss.Width(journalFrame.Render(""))
		journalWidth := m.width - journalExtraWidth
		if journalWidth < 1 {
			journalWidth = 1
		}
		journalPane = journalFrame.Width(journalWidth).Render(searchContent.String())
	} else if len(m.journalEntries) > 0 || m.modal.journaling {
		var journalContent strings.Builder
		journalContent.WriteString(m.theme.Focused.Render("Journal") + "\n\n")
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
				m.theme.Dim.Render(entry.CreatedAt.Format("15:04")),
				m.theme.Highlight.Render(labelStr),
				entry.Content)
			journalContent.WriteString(line + "\n")
		}
		if m.modal.journaling {
			journalContent.WriteString("\n" + m.theme.Focused.Render("> ") + m.inputs.journalInput.View())
		}
		journalFrame := Frames.Modal.Padding(0, 1)
		journalExtraWidth := lipgloss.Width(journalFrame.Render(""))
		journalWidth := m.width - journalExtraWidth
		if journalWidth < 1 {
			journalWidth = 1
		}
		journalPane = journalFrame.Width(journalWidth).Render(journalContent.String())
	}

	return journalPane
}
