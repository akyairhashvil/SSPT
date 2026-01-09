//go:build !sqlcipher

package sqlite3

/*
#include <sqlite3.h>
*/
import "C"

import "errors"

func applyKey(db *C.sqlite3, key string) error {
	if key == "" {
		return nil
	}
	return errors.New("sqlcipher is unavailable")
}
