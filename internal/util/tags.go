package util

import (
	"encoding/json"
	"regexp"
	"strings"
)

var hashtagRegex = regexp.MustCompile(`#(\w+)`)

// ExtractTags finds all #hashtags in a string and returns them as a slice of strings.
func ExtractTags(text string) []string {
	matches := hashtagRegex.FindAllStringSubmatch(text, -1)
	tags := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		tag := strings.ToLower(match[1])
		if !seen[tag] {
			tags = append(tags, tag)
			seen[tag] = true
		}
	}
	return tags
}

// TagsToJSON converts a slice of tags into a JSON array string.
func TagsToJSON(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	bytes, _ := json.Marshal(tags)
	return string(bytes)
}

// JSONToTags converts a JSON array string back into a slice of tags.
func JSONToTags(jsonStr string) []string {
	var tags []string
	if jsonStr == "" || jsonStr == "null" {
		return []string{}
	}
	json.Unmarshal([]byte(jsonStr), &tags)
	return tags
}
