package config

// Layout constants.
const (
	// DefaultFocusColumn is the initially focused column (1 = current sprint).
	DefaultFocusColumn = 1

	// MinColumnWidth is the minimum width for a goal column.
	MinColumnWidth = 10

	// CompactModeThreshold triggers compact rendering below this width.
	CompactModeThreshold = 60

	// TargetTitleWidth is the preferred width for goal titles.
	TargetTitleWidth = 30

	// MinTitleWidth is the minimum width for goal titles.
	MinTitleWidth = 10
)

// Display limits.
const (
	// MaxVisibleGoals limits goals shown per column before scrolling.
	MaxVisibleGoals = 15

	// MaxTagsDisplayed limits inline tag display.
	MaxTagsDisplayed = 3

	// TruncationSuffix appended to truncated strings.
	TruncationSuffix = "..."
)

// Input constraints.
const (
	// MaxTitleLength is the maximum goal title length.
	MaxTitleLength = 100

	// MaxDescriptionLength is the maximum description length.
	MaxDescriptionLength = 500

	// MaxTagLength is the maximum tag name length.
	MaxTagLength = 20
)
