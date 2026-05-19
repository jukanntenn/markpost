package utils

func Deref[T any](s *T) T {
	if s == nil {
		var zero T
		return zero
	}
	return *s
}

func MapSlice[T any, R any](src []T, fn func(T) R) []R {
	result := make([]R, 0, len(src))
	for _, item := range src {
		result = append(result, fn(item))
	}
	return result
}
