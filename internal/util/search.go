package util

import (
	"regexp"
	"strings"
)

// SearchQuery represents the parsed components of a search string.
type SearchQuery struct {
	Tags      []string
	Status    []string
	Workspace []string
	Type      []string
	Text      []string
}

var (
	tagRegex       = regexp.MustCompile(`tag:(\w+)`)
	statusRegex    = regexp.MustCompile(`status:(\w+)`)
	workspaceRegex = regexp.MustCompile(`workspace:(\w+)`)
	typeRegex      = regexp.MustCompile(`type:(\w+)`)
)

// ParseSearchQuery breaks down a raw query string into its structured components.
func ParseSearchQuery(query string) SearchQuery {
	sq := SearchQuery{}
	
	extract := func(re *regexp.Regexp) []string {
		matches := re.FindAllStringSubmatch(query, -1)
		if matches == nil { return nil }
		var values []string
		for _, match := range matches {
			if len(match) > 1 {
				values = append(values, match[1])
			}
		}
		query = re.ReplaceAllString(query, "")
		return values
	}

	sq.Tags = extract(tagRegex)
	sq.Status = extract(statusRegex)
	sq.Workspace = extract(workspaceRegex)
	sq.Type = extract(typeRegex)
	sq.Text = strings.Fields(query)

	return sq
}
