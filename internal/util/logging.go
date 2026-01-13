// Package util provides common utilities including logging helpers,
// file system operations, and string manipulation functions.
package util

import (
	"fmt"
	"log"
	"os"
)

// LogError logs an error with context if it is non-nil.
func LogError(context string, err error) {
	if err != nil {
		log.Printf("%s: %v", context, err)
	}
}

// StartupError represents a fatal error during initialization.
type StartupError struct {
	Context string
	Err     error
}

func (e *StartupError) Error() string {
	return fmt.Sprintf("%s: %v", e.Context, e.Err)
}

// CheckStartup returns a StartupError if err is not nil.
// Use this during initialization phases only.
func CheckStartup(context string, err error) error {
	if err != nil {
		return &StartupError{Context: context, Err: err}
	}
	return nil
}

// MustSucceed should only be used in main() before the TUI starts.
// After TUI initialization, use error returns instead.
func MustSucceed(context string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: %s: %v\n", context, err)
		os.Exit(1)
	}
}
