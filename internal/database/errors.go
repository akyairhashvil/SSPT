package database

import "errors"

var (
	ErrDatabaseEncrypted = errors.New("database is encrypted")
	ErrDatabaseCorrupted = errors.New("database file is corrupted")
	ErrWrongPassphrase   = errors.New("incorrect passphrase")
)
