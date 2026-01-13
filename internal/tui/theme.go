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
	"cyberpunk": {
		Name:          "Cyberpunk",
		Base:          lipgloss.NewStyle().Margin(1, 2),
		Border:        lipgloss.Color("201"),                                                                  // Neon magenta
		Header:        lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true).Align(lipgloss.Center), // Neon cyan
		Goal:          lipgloss.NewStyle().Foreground(lipgloss.Color("231")),                                  // Bright white
		CompletedGoal: lipgloss.NewStyle().Foreground(lipgloss.Color("239")).Strikethrough(true),
		Break:         lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true), // Neon orange
		Input:         lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("51")).Padding(0, 1).Width(50),
		TagUrgent:     lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true),
		TagDocs:       lipgloss.NewStyle().Foreground(lipgloss.Color("45")).Bold(true),
		TagBlocked:    lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Bold(true),
		TagBug:        lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		TagIdea:       lipgloss.NewStyle().Foreground(lipgloss.Color("177")).Bold(true),
		TagReview:     lipgloss.NewStyle().Foreground(lipgloss.Color("221")).Bold(true),
		TagFocus:      lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Bold(true),
		TagLater:      lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		TagDefault:    lipgloss.NewStyle().Foreground(lipgloss.Color("86")),
		Focused:       lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Bold(true),
		Dim:           lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		Highlight:     lipgloss.NewStyle().Foreground(lipgloss.Color("201")),
	},
	"solar": {
		Name:          "Solar",
		Base:          lipgloss.NewStyle().Margin(1, 2),
		Border:        lipgloss.Color("245"),
		Header:        lipgloss.NewStyle().Foreground(lipgloss.Color("166")).Bold(true).Align(lipgloss.Center),
		Goal:          lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		CompletedGoal: lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Strikethrough(true),
		Break:         lipgloss.NewStyle().Foreground(lipgloss.Color("173")).Bold(true),
		Input:         lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("166")).Padding(0, 1).Width(50),
		TagUrgent:     lipgloss.NewStyle().Foreground(lipgloss.Color("160")).Bold(true),
		TagDocs:       lipgloss.NewStyle().Foreground(lipgloss.Color("109")).Bold(true),
		TagBlocked:    lipgloss.NewStyle().Foreground(lipgloss.Color("136")).Bold(true),
		TagBug:        lipgloss.NewStyle().Foreground(lipgloss.Color("124")).Bold(true),
		TagIdea:       lipgloss.NewStyle().Foreground(lipgloss.Color("135")).Bold(true),
		TagReview:     lipgloss.NewStyle().Foreground(lipgloss.Color("172")).Bold(true),
		TagFocus:      lipgloss.NewStyle().Foreground(lipgloss.Color("66")).Bold(true),
		TagLater:      lipgloss.NewStyle().Foreground(lipgloss.Color("243")),
		TagDefault:    lipgloss.NewStyle().Foreground(lipgloss.Color("110")),
		Focused:       lipgloss.NewStyle().Foreground(lipgloss.Color("166")).Bold(true),
		Dim:           lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		Highlight:     lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
	},
	"paper": {
		Name:          "Paper",
		Base:          lipgloss.NewStyle().Margin(1, 2),
		Border:        lipgloss.Color("238"),
		Header:        lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Bold(true).Align(lipgloss.Center),
		Goal:          lipgloss.NewStyle().Foreground(lipgloss.Color("236")),
		CompletedGoal: lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Strikethrough(true),
		Break:         lipgloss.NewStyle().Foreground(lipgloss.Color("124")).Bold(true),
		Input:         lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1).Width(50),
		TagUrgent:     lipgloss.NewStyle().Foreground(lipgloss.Color("124")).Bold(true),
		TagDocs:       lipgloss.NewStyle().Foreground(lipgloss.Color("25")).Bold(true),
		TagBlocked:    lipgloss.NewStyle().Foreground(lipgloss.Color("88")).Bold(true),
		TagBug:        lipgloss.NewStyle().Foreground(lipgloss.Color("124")).Bold(true),
		TagIdea:       lipgloss.NewStyle().Foreground(lipgloss.Color("94")).Bold(true),
		TagReview:     lipgloss.NewStyle().Foreground(lipgloss.Color("130")).Bold(true),
		TagFocus:      lipgloss.NewStyle().Foreground(lipgloss.Color("28")).Bold(true),
		TagLater:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		TagDefault:    lipgloss.NewStyle().Foreground(lipgloss.Color("238")),
		Focused:       lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Bold(true),
		Dim:           lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
		Highlight:     lipgloss.NewStyle().Foreground(lipgloss.Color("238")),
	},
	"mint": {
		Name:          "Mint",
		Base:          lipgloss.NewStyle().Margin(1, 2),
		Border:        lipgloss.Color("30"),
		Header:        lipgloss.NewStyle().Foreground(lipgloss.Color("23")).Bold(true).Align(lipgloss.Center),
		Goal:          lipgloss.NewStyle().Foreground(lipgloss.Color("22")),
		CompletedGoal: lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Strikethrough(true),
		Break:         lipgloss.NewStyle().Foreground(lipgloss.Color("124")).Bold(true),
		Input:         lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("30")).Padding(0, 1).Width(50),
		TagUrgent:     lipgloss.NewStyle().Foreground(lipgloss.Color("160")).Bold(true),
		TagDocs:       lipgloss.NewStyle().Foreground(lipgloss.Color("24")).Bold(true),
		TagBlocked:    lipgloss.NewStyle().Foreground(lipgloss.Color("88")).Bold(true),
		TagBug:        lipgloss.NewStyle().Foreground(lipgloss.Color("160")).Bold(true),
		TagIdea:       lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true),
		TagReview:     lipgloss.NewStyle().Foreground(lipgloss.Color("130")).Bold(true),
		TagFocus:      lipgloss.NewStyle().Foreground(lipgloss.Color("28")).Bold(true),
		TagLater:      lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
		TagDefault:    lipgloss.NewStyle().Foreground(lipgloss.Color("23")),
		Focused:       lipgloss.NewStyle().Foreground(lipgloss.Color("22")).Bold(true),
		Dim:           lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
		Highlight:     lipgloss.NewStyle().Foreground(lipgloss.Color("30")),
	},
}

type FrameStyles struct {
	Modal    lipgloss.Style
	Lock     lipgloss.Style
	Dialog   lipgloss.Style
	Floating lipgloss.Style
}

func NewFrameStyles() FrameStyles {
	base := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2)

	return FrameStyles{
		Modal:    base.Copy().BorderForeground(lipgloss.Color("62")),
		Lock:     base.Copy().BorderForeground(lipgloss.Color("196")),
		Dialog:   base.Copy().BorderForeground(lipgloss.Color("39")),
		Floating: base.Copy().BorderForeground(lipgloss.Color("240")),
	}
}

var Frames = NewFrameStyles()

// CurrentTheme holds the currently active theme.
// We initialize it to default to avoid nil pointer dereferences.
var CurrentTheme = Themes["default"]

func SetTheme(name string) {
	if t, ok := Themes[name]; ok {
		CurrentTheme = t
	}
}
