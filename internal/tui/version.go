package tui

import "fmt"

func versionLabel() string {
	label := AppVersion
	if GitCommit != "unknown" || BuildTime != "unknown" {
		label = fmt.Sprintf("%s (%s %s)", AppVersion, GitCommit, BuildTime)
	}
	return label
}
