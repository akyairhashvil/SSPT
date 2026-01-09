//go:build sqlcipher

package sqlite3

/*
#cgo CFLAGS: -I/usr/include/sqlcipher -DSQLITE_HAS_CODEC
#include <stdlib.h>
#include <sqlite3.h>
*/
import "C"

import "unsafe"

func applyKey(db *C.sqlite3, key string) error {
	if key == "" {
		return nil
	}
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	rv := C.sqlite3_key(db, unsafe.Pointer(cKey), C.int(len(key)))
	if rv != C.SQLITE_OK {
		return lastError(db)
	}
	return nil
}
