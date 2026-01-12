package util

import (
	"reflect"
	"testing"
)

func TestParseSearchQuery(t *testing.T) {
	query := "tag:urgent status:completed workspace:personal type:goal some words"
	got := ParseSearchQuery(query)

	if !reflect.DeepEqual(got.Tags, []string{"urgent"}) {
		t.Fatalf("Tags = %v, want %v", got.Tags, []string{"urgent"})
	}
	if !reflect.DeepEqual(got.Status, []string{"completed"}) {
		t.Fatalf("Status = %v, want %v", got.Status, []string{"completed"})
	}
	if !reflect.DeepEqual(got.Workspace, []string{"personal"}) {
		t.Fatalf("Workspace = %v, want %v", got.Workspace, []string{"personal"})
	}
	if !reflect.DeepEqual(got.Type, []string{"goal"}) {
		t.Fatalf("Type = %v, want %v", got.Type, []string{"goal"})
	}
	if !reflect.DeepEqual(got.Text, []string{"some", "words"}) {
		t.Fatalf("Text = %v, want %v", got.Text, []string{"some", "words"})
	}
}
