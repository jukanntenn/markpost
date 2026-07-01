package filter

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

func normalizeMatch(s string) string {
	return strings.ToLower(norm.NFC.String(s))
}

func containsSubstr(lowerTitle, lowerKeyword string) bool {
	return strings.Contains(lowerTitle, lowerKeyword)
}
