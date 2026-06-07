package utils

// Deref returns the value pointed to by s, or the zero value of T if s is nil.
func Deref[T any](s *T) T {
	if s == nil {
		var zero T
		return zero
	}
	return *s
}

// MapSlice applies fn to each element of src and returns a new slice of the results.
func MapSlice[T any, R any](src []T, fn func(T) R) []R {
	result := make([]R, 0, len(src))
	for _, item := range src {
		result = append(result, fn(item))
	}
	return result
}
