package util

// BoolToInt converts a boolean to 0 or 1.
func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// IntToBool converts 0/1 to boolean.
func IntToBool(i int) bool {
	return i != 0
}

// Ptr returns a pointer to the value.
func Ptr[T any](v T) *T {
	return &v
}

// Deref safely dereferences a pointer, returning the zero value if nil.
func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// Clamp constrains a value to a range.
func Clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
