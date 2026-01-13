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

func wrapGoalErr(op string, id int64, err error) error {
	if err == nil {
		return nil
	}
	return &OpError{Op: op, Resource: "goal", ID: id, Err: err}
}

func wrapSprintErr(op string, id int64, err error) error {
	if err == nil {
		return nil
	}
	return &OpError{Op: op, Resource: "sprint", ID: id, Err: err}
}

func wrapWorkspaceErr(op string, id int64, err error) error {
	if err == nil {
		return nil
	}
	return &OpError{Op: op, Resource: "workspace", ID: id, Err: err}
}
