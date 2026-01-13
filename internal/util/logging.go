package util

import "log"

// LogError logs an error with context if it is non-nil.
func LogError(context string, err error) {
	if err != nil {
		log.Printf("%s: %v", context, err)
	}
}

// MustSucceed logs and exits on error. Use sparingly.
func MustSucceed(context string, err error) {
	if err != nil {
		log.Fatalf("%s: %v", context, err)
	}
}
