package tui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Name          string
	Base          lipgloss.Style
	Border        lipgloss.Color
	Header        lipgloss.Style
	Goal          lipgloss.Style
	CompletedGoal lipgloss.Style
	Break         lipgloss.Style
	Input         lipgloss.Style
	TagUrgent     lipgloss.Style
	TagDocs       lipgloss.Style
	TagBlocked    lipgloss.Style
	TagBug        lipgloss.Style
	TagIdea       lipgloss.Style
	TagReview     lipgloss.Style
	TagFocus      lipgloss.Style
	TagLater      lipgloss.Style
	TagDefault    lipgloss.Style
	Focused       lipgloss.Style
	Dim           lipgloss.Style
	Highlight     lipgloss.Style
}

var Themes = map[string]Theme{
	"default": {
		Name:          "Default",
		Base:          lipgloss.NewStyle().Margin(1, 2),
		Border:        lipgloss.Color("63"),
		Header:        lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Align(lipgloss.Center),
		Goal:          lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		CompletedGoal: lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Strikethrough(true),
		Break:         lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true),
		Input:         lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("205")).Padding(0, 1).Width(50),
		TagUrgent:     lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true),
		TagDocs:       lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true),
		TagBlocked:    lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true),
		TagBug:        lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		TagIdea:       lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true),
		TagReview:     lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true),
		TagFocus:      lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true),
		TagLater:      lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		TagDefault:    lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
		Focused:       lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		Dim:           lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		Highlight:     lipgloss.NewStyle().Foreground(lipgloss.Color("63")),
	},
	"dracula": {
		Name:          "Dracula",
		Base:          lipgloss.NewStyle().Margin(1, 2),
		Border:        lipgloss.Color("62"),                                                                   // Purple
		Header:        lipgloss.NewStyle().Foreground(lipgloss.Color("50")).Bold(true).Align(lipgloss.Center), // Cyan
		Goal:          lipgloss.NewStyle().Foreground(lipgloss.Color("255")),                                  // White
		CompletedGoal: lipgloss.NewStyle().Foreground(lipgloss.Color("60")).Strikethrough(true),               // Comment
		Break:         lipgloss.NewStyle().Foreground(lipgloss.Color("215")).Bold(true),                       // Orange
		Input:         lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("50")).Padding(0, 1).Width(50),
		TagUrgent:     lipgloss.NewStyle().Foreground(lipgloss.Color("210")).Bold(true), // Red/Pink
		TagDocs:       lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Bold(true), // Cyan
		TagBlocked:    lipgloss.NewStyle().Foreground(lipgloss.Color("228")).Bold(true), // Yellow
		TagBug:        lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true), // Red
		TagIdea:       lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true), // Purple
		TagReview:     lipgloss.NewStyle().Foreground(lipgloss.Color("215")).Bold(true), // Orange
		TagFocus:      lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true),  // Blue
		TagLater:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")),            // Grey
		TagDefault:    lipgloss.NewStyle().Foreground(lipgloss.Color("120")),            // Green
		Focused:       lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true), // Pink
		Dim:           lipgloss.NewStyle().Foreground(lipgloss.Color("60")),
		Highlight:     lipgloss.NewStyle().Foreground(lipgloss.Color("62")),
	},
}

// CurrentTheme holds the currently active theme.
// We initialize it to default to avoid nil pointer dereferences.
var CurrentTheme = Themes["default"]

func SetTheme(name string) {
	if t, ok := Themes[name]; ok {
		CurrentTheme = t
	}
}
