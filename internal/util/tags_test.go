package util

import (
	"reflect"
	"testing"
)

func TestExtractTagsUniqueLowercase(t *testing.T) {
	input := "Fix #Bug and #bug then #Test"
	got := ExtractTags(input)
	want := []string{"bug", "test"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractTags() = %v, want %v", got, want)
	}
}

func TestTagsJSONRoundTrip(t *testing.T) {
	tags := []string{"one", "two"}
	jsonStr := TagsToJSON(tags)
	got := JSONToTags(jsonStr)
	if !reflect.DeepEqual(got, tags) {
		t.Fatalf("JSONToTags(TagsToJSON()) = %v, want %v", got, tags)
	}
}

func TestJSONToTagsEmptyInput(t *testing.T) {
	if got := JSONToTags(""); len(got) != 0 {
		t.Fatalf("JSONToTags(\"\") = %v, want empty", got)
	}
}
