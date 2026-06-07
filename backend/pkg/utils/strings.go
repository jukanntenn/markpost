package utils

import "strings"

// Normalize trims whitespace and lowercases the input string.
func Normalize(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// ApplyIfNonEmpty assigns the trimmed value to target if it is non-empty after trimming.
func ApplyIfNonEmpty(target *string, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		*target = trimmed
	}
}
