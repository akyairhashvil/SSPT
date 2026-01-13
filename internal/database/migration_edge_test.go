package database

import (
	"errors"
	"testing"
)

func TestIsIgnorableMigrationErr(t *testing.T) {
	if !isIgnorableMigrationErr(errors.New("duplicate column name: rank")) {
		t.Fatalf("expected duplicate column error to be ignorable")
	}
	if isIgnorableMigrationErr(errors.New("no such table: goals")) {
		t.Fatalf("expected non-duplicate error to be non-ignorable")
	}
}
