package infra

import (
	"strings"

	"gorm.io/gorm"
)

var likeEscaper = strings.NewReplacer(
	`\`, `\\`,
	`%`, `\%`,
	`_`, `\_`,
)

func likeContains(s string) string {
	return "%" + likeEscaper.Replace(s) + "%"
}

func buildSearchCondition(search string, fields ...string) (string, []any) {
	if search == "" || len(fields) == 0 {
		return "", nil
	}
	parts := make([]string, len(fields))
	args := make([]any, len(fields))
	pattern := likeContains(search)
	for i, field := range fields {
		parts[i] = field + " LIKE ?"
		args[i] = pattern
	}
	return strings.Join(parts, " OR "), args
}

func applySearch(query *gorm.DB, search string, fields ...string) *gorm.DB {
	cond, args := buildSearchCondition(search, fields...)
	if cond == "" {
		return query
	}
	return query.Where(cond, args...)
}
