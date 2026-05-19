package utils

import "strings"

func Normalize(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func ApplyIfNonEmpty(target *string, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		*target = trimmed
	}
}
