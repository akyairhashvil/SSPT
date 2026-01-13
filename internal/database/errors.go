package database

import (
	"errors"
	"fmt"
)

var (
	ErrDatabaseEncrypted = errors.New("database is encrypted")
	ErrDatabaseCorrupted = errors.New("database file is corrupted")
	ErrWrongPassphrase   = errors.New("incorrect passphrase")
)

type GoalError struct {
	Op  string
	ID  int64
	Err error
}

func (e *GoalError) Error() string {
	if e.ID > 0 {
		return fmt.Sprintf("goal %s (id=%d): %v", e.Op, e.ID, e.Err)
	}
	return fmt.Sprintf("goal %s: %v", e.Op, e.Err)
}

func (e *GoalError) Unwrap() error { return e.Err }

type SprintError struct {
	Op  string
	ID  int64
	Err error
}

func (e *SprintError) Error() string {
	if e.ID > 0 {
		return fmt.Sprintf("sprint %s (id=%d): %v", e.Op, e.ID, e.Err)
	}
	return fmt.Sprintf("sprint %s: %v", e.Op, e.Err)
}

func (e *SprintError) Unwrap() error { return e.Err }

type WorkspaceError struct {
	Op  string
	ID  int64
	Err error
}

func (e *WorkspaceError) Error() string {
	if e.ID > 0 {
		return fmt.Sprintf("workspace %s (id=%d): %v", e.Op, e.ID, e.Err)
	}
	return fmt.Sprintf("workspace %s: %v", e.Op, e.Err)
}

func (e *WorkspaceError) Unwrap() error { return e.Err }
