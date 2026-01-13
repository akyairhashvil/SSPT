package database

import "database/sql"

// nullableInt64 converts an int64 to sql.NullInt64 for optional fields.
// Values <= 0 are treated as NULL.
func nullableInt64(v int64) sql.NullInt64 {
	return sql.NullInt64{Int64: v, Valid: v > 0}
}

// nullableString converts a string to sql.NullString for optional fields.
// Empty strings are treated as NULL.
func nullableString(v string) sql.NullString {
	return sql.NullString{String: v, Valid: v != ""}
}

// toNullableArg converts a pointer to an interface{} suitable for SQL args.
// Returns nil if pointer is nil, otherwise returns the dereferenced value.
func toNullableArg[T any](v *T) interface{} {
	if v == nil {
		return nil
	}
	return *v
}
