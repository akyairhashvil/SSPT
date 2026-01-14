package database

import (
	"errors"
	"fmt"
)

var (
	ErrDatabaseEncrypted  = errors.New("database is encrypted")
	ErrDatabaseCorrupted  = errors.New("database file is corrupted")
	ErrWrongPassphrase    = errors.New("incorrect passphrase")
	ErrCircularDependency = errors.New("circular dependency detected")
)

const (
	OpAdd     = "add"
	OpUpdate  = "update"
	OpDelete  = "delete"
	OpGet     = "get"
	OpList    = "list"
	OpExists  = "exists"
	OpMove    = "move"
	OpReorder = "reorder"
)

const (
	EntityGoal      = "goal"
	EntitySprint    = "sprint"
	EntityWorkspace = "workspace"
	EntityTag       = "tag"
	EntityJournal   = "journal"
	EntitySetting   = "setting"
)

type OpError struct {
	Op       string
	Resource string
	ID       int64
	Err      error
}

func (e *OpError) Error() string {
	if e == nil {
		return ""
	}
	if e.ID > 0 {
		return fmt.Sprintf("%s %s %d: %v", e.Op, e.Resource, e.ID, e.Err)
	}
	return fmt.Sprintf("%s %s: %v", e.Op, e.Resource, e.Err)
}

func (e *OpError) Unwrap() error { return e.Err }

func wrapErr(entity, op string, id int64, err error) error {
	if err == nil {
		return nil
	}
	return &OpError{Op: op, Resource: entity, ID: id, Err: err}
}
